package clientapp

import (
	"crypto/mlkem"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/BNKDuy/BroChat/internal/chat"
	"golang.org/x/crypto/chacha20poly1305"
)

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

func (c *ChatClient) PrepareHybridHandshake() {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	c.secretKey.Store(nil)

	var err error
	c.decapsulationKey, err = mlkem.GenerateKey768()
	if err != nil {
		log.Fatal(err)
	}

	encapsulationKey := c.decapsulationKey.EncapsulationKey().Bytes()

	OutGoingMessage := chat.ChatMessage{
		From:    c.username,
		To:      c.recipient,
		Type:    chat.MessageTypeMLKEMEncapsulationKey,
		Content: base64.StdEncoding.EncodeToString(encapsulationKey),
	}

	msg, err := json.Marshal(OutGoingMessage)
	if err != nil {
		fmt.Println("Failed to initialize Key Exchange")
		os.Exit(1)
	}

	err = c.safeWrite(msg)
	if err != nil {
		log.Println("Write error:", err)
		return
	}
}

func (c *ChatClient) ResponseHybridHandshake(request string) {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	encapsulationKey, err := base64.StdEncoding.DecodeString(request)
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
	c.secretKey.Store(&newKey)

	OutGoingMessage := chat.ChatMessage{
		From:    c.username,
		To:      c.recipient,
		Type:    chat.MessageTypeMLKEMCiphertext,
		Content: base64.StdEncoding.EncodeToString(ciphertext),
	}

	msg, err := json.Marshal(OutGoingMessage)
	if err != nil {
		fmt.Println("Key exchange failed")
		os.Exit(1)
	}

	err = c.safeWrite(msg)
	if err != nil {
		log.Println("Write error:", err)
		return
	}
}

func (c *ChatClient) FinalizeHybridHandshake(response string) {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	ciphertext, err := base64.StdEncoding.DecodeString(response)
	if err != nil || c.decapsulationKey == nil {
		fmt.Println("Key exchange failed")
		os.Exit(1)
	}

	newKey, err := c.decapsulationKey.Decapsulate(ciphertext)
	if err != nil {
		fmt.Println("Key exchange failed")
		os.Exit(1)
	}
	c.decapsulationKey = nil
	c.secretKey.Store(&newKey)
}
