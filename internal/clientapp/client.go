package clientapp

import (
	"crypto/mlkem"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/BNKDuy/BroChat/internal/chat"
	"github.com/gorilla/websocket"
)

type ChatClient struct {
	username         string
	recipient        string
	app              fyne.App
	secretKey        atomic.Pointer[[]byte]
	decapsulationKey *mlkem.DecapsulationKey768
	handshakeMutex   sync.Mutex
	conn             *websocket.Conn
	socketWriteMu    sync.Mutex

	// Windows
	chatWindow        fyne.Window
	chatHistory       *fyne.Container
	chatHistoryScroll *container.Scroll
}

func NewChatClient(url string, username string, recipient string) *ChatClient {
	client := &ChatClient{
		app:       app.New(),
		username:  username,
		recipient: recipient,
	}
	client.NewChatWindow()

	err := client.newSocketConnection(url)
	if err != nil {
		os.Exit(1)
	} else {
		go client.readPump()
	}
	return client
}

func (c *ChatClient) Start() {
	c.chatWindow.ShowAndRun()
}

func (c *ChatClient) SendMessage(text string) {
	key := c.secretKey.Load()
	if key == nil {
		fmt.Println("Send failed!")
		return
	}

	encryptedMessage, err := encrypt(text, *key)
	if err != nil {
		fmt.Println("Send failed!")
		return
	}

	OutGoingMessage := chat.ChatMessage{
		From:    c.username,
		To:      c.recipient,
		Type:    chat.MessageTypeChat,
		Content: encryptedMessage,
	}

	msg, err := json.Marshal(OutGoingMessage)
	if err != nil {
		fmt.Println("Failed to sent message (Encoding error)")
	}

	err = c.safeWrite(msg)
	if err != nil {
		log.Println("Write error:", err)
		return
	}
}

func (c *ChatClient) readPump() {
	defer c.conn.Close()
	for {
		_, response, err := c.conn.ReadMessage()
		if err != nil {
			// Update UI to let user know we lost connection
			c.addMessageToChat("System: Disconnected from server.")
			return
		}

		// Handle each message in its own goroutine to keep the pump moving
		go c.handleIncomingMessage(response)
	}
}

func (c *ChatClient) handleIncomingMessage(response []byte) {
	var chatMsg chat.ChatMessage
	if err := json.Unmarshal(response, &chatMsg); err != nil {
		fmt.Println("Failed to receive message!")
		return
	}

	switch chatMsg.Type {
	case chat.MessageTypeChat:
		key := c.secretKey.Load()
		if key == nil {
			fmt.Println("Message recieve failed: Decrypt key not found.")
			return
		}
		decryptedMessage, err := decrypt(chatMsg.Content, *key)
		if err != nil {
			return
		}
		c.addMessageToChat(chatMsg.From + ": " + decryptedMessage)
	case chat.MessageTypeSystem:
		c.addMessageToChat("System: " + chatMsg.Content)
	case chat.MessageTypeMLKEMInit:
		c.PrepareHybridHandshake()
	case chat.MessageTypeMLKEMEncapsulationKey:
		c.ResponseHybridHandshake(chatMsg.Content)
	case chat.MessageTypeMLKEMCiphertext:
		c.FinalizeHybridHandshake(chatMsg.Content)
	case chat.MessageTypeStopEncryptedSession:
		fmt.Println("[SYSTEM] Peer disconnected. Encrypted session stopped.")
		c.secretKey.Store(nil)
	}
}
