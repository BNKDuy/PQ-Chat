package main

import (
	"bufio"
	"crypto/mlkem"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/BNKDuy/BroChat/internal/chat"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/chacha20poly1305"
)

func main() {
	username := os.Args[1]
	targetUser := os.Args[2]

	var secretKey atomic.Pointer[[]byte]
	var decapsulationKey *mlkem.DecapsulationKey768

	environment := os.Getenv("Environment")
	var url string
	if environment == "local" {
		localport := os.Getenv("PORT")
		url = "ws://localhost:" + localport + "/ws/" + username + "/" + targetUser
	} else {
		prodUrl := os.Getenv("ProdURL")
		url = "wss://" + prodUrl + "/ws/" + username + "/" + targetUser
	}

	fmt.Printf("Connecting to server at %s...\n", url)

	// Connect to the WebSocket server
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer c.Close()

	// Mutex to prevent concurrent writes to the websocket
	var writeMu sync.Mutex
	safeWrite := func(msg []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return c.WriteMessage(websocket.TextMessage, msg)
	}

	fmt.Println("Connected! Type your message and press Enter.")
	fmt.Println("------------------------------------------------")

	// Goroutine to READ messages from the server
	go func() {
		var chatMsg chat.ChatMessage
		for {
			_, response, err := c.ReadMessage()
			if err != nil {
				log.Println("\nDisconnected from server:", err)
				os.Exit(0)
			}

			if err := json.Unmarshal(response, &chatMsg); err != nil {
				fmt.Println("Failed to receive message!")
				continue
			}

			switch {
			case chatMsg.Type == chat.MessageTypeChat:
				key := secretKey.Load()
				if key == nil {
					fmt.Println("Message recieve failed: Decrypt key not found.")
					continue
				}
				decryptedMessage, err := decrypt(chatMsg.Content, *key)
				if err != nil {
					continue
				}
				fmt.Printf("%s: %s\n", chatMsg.From, decryptedMessage)
			case chatMsg.Type == chat.MessageTypeSystem:
				fmt.Printf("System: %s\n", chatMsg.Content)
			case chatMsg.Type == chat.MessageTypeMLKEMInit:
				secretKey.Store(nil)

				decapsulationKey, err = mlkem.GenerateKey768()
				if err != nil {
					log.Fatal(err)
				}

				encapsulationKey := decapsulationKey.EncapsulationKey().Bytes()

				OutGoingMessage := chat.ChatMessage{
					From:    username,
					To:      targetUser,
					Type:    chat.MessageTypeMLKEMEncapsulationKey,
					Content: base64.StdEncoding.EncodeToString(encapsulationKey),
				}

				msg, err := json.Marshal(OutGoingMessage)
				if err != nil {
					fmt.Println("Failed to initialize Key Exchange")
					os.Exit(1)
				}

				err = safeWrite(msg)
				if err != nil {
					log.Println("Write error:", err)
					return
				}
			case chatMsg.Type == chat.MessageTypeMLKEMEncapsulationKey:
				encapsulationKey, err := base64.StdEncoding.DecodeString(chatMsg.Content)
				if err != nil {
					fmt.Println("Key exchange failed")
					os.Exit(1)
				}

				ek, err := mlkem.NewEncapsulationKey768(encapsulationKey)
				if err != nil {
					fmt.Println("Key exchange failed")
					os.Exit(1)
				}

				newKey, ciphertext := ek.Encapsulate()
				fmt.Println("Shared key established: ", newKey)
				secretKey.Store(&newKey)

				OutGoingMessage := chat.ChatMessage{
					From:    username,
					To:      targetUser,
					Type:    chat.MessageTypeMLKEMCiphertext,
					Content: base64.StdEncoding.EncodeToString(ciphertext),
				}

				msg, err := json.Marshal(OutGoingMessage)
				if err != nil {
					fmt.Println("Key exchange failed")
					os.Exit(1)
				}

				err = safeWrite(msg)
				if err != nil {
					log.Println("Write error:", err)
					return
				}
			case chatMsg.Type == chat.MessageTypeMLKEMCiphertext:
				ciphertext, err := base64.StdEncoding.DecodeString(chatMsg.Content)
				if err != nil || decapsulationKey == nil {
					fmt.Println("Key exchange failed")
					os.Exit(1)
				}

				newKey, err := decapsulationKey.Decapsulate(ciphertext)
				if err != nil {
					fmt.Println("Key exchange failed")
					os.Exit(1)
				}
				decapsulationKey = nil
				fmt.Println("Shared key established: ", newKey)
				secretKey.Store(&newKey)
			case chatMsg.Type == chat.MessageTypeStopEncryptedSession:
				fmt.Println("[SYSTEM] Peer disconnected. Encrypted session stopped.")
				secretKey.Store(nil)
			}
		}
	}()

	// Goroutine to WRITE messages from the terminal
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			if text == "" {
				continue
			}

			key := secretKey.Load()
			if key == nil {
				fmt.Println("Send failed!")
				continue
			}

			encryptedMessage, err := encrypt(text, *key)
			if err != nil {
				continue
			}

			OutGoingMessage := chat.ChatMessage{
				From:    username,
				To:      targetUser,
				Type:    chat.MessageTypeChat,
				Content: encryptedMessage,
			}

			msg, err := json.Marshal(OutGoingMessage)
			if err != nil {
				fmt.Println("Failed to sent message (Encoding error)")
			}

			err = safeWrite(msg)
			if err != nil {
				log.Println("Write error:", err)
				return
			}
		}
	}()

	// 5. Keep the client running until you press Ctrl+C
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	<-interrupt

	fmt.Println("\nClosing connection gracefully...")
	c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
}

func encrypt(message string, key []byte) (ciphertext string, err error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		fmt.Println("Message send failed: Failed to encrypt message.")
		return "", err
	}
	// Select a random nonce, and leave capacity for the ciphertext.
	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(message)+aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		fmt.Println("Message send failed: Failed to encrypt message.")
		return "", err
	}

	// Encrypt the message and append the ciphertext to the nonce.
	encryptedMsg := aead.Seal(nonce, nonce, []byte(message), nil)

	return base64.StdEncoding.EncodeToString(encryptedMsg), nil
}

func decrypt(encryptedMessage string, key []byte) (decryptedMessage string, err error) {
	msg, err := base64.StdEncoding.DecodeString(encryptedMessage)
	if err != nil {
		fmt.Println("Message recieve failed: Failed to decrypt message.")
		return "", err
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		fmt.Println("Message recieve failed: Failed to decrypt message.")
		return "", err
	}

	if len(msg) < aead.NonceSize() {
		fmt.Println("Message recieve failed: Failed to decrypt message.")
		return "", fmt.Errorf("ciphertext length is too short")
	}

	// Split nonce and ciphertext.
	nonce, ciphertext := msg[:aead.NonceSize()], msg[aead.NonceSize():]

	// Decrypt the message and check it wasn't tampered with.
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Println("Message recieve failed: Failed to decrypt message.")
		return "", err
	}

	return string(plaintext), nil
}
