package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/BNKDuy/BroChat/internal/chat"
	"github.com/gorilla/websocket"
)

func main() {
	username := os.Args[1]
	targetUser := os.Args[2]

	url := "ws://localhost:8080/ws/" + username
	fmt.Printf("Connecting to server at %s...\n", url)

	// 1. Connect to the WebSocket server
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatal("Dial error:", err)
	}
	defer c.Close()

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

			fmt.Printf("%s: %s\n", chatMsg.From, chatMsg.Content)
		}
	}()

	// Goroutine to WRITE messages from your terminal
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			if text == "" {
				continue
			}

			OutGoingMessage := chat.ChatMessage{
				From:    username,
				To:      targetUser,
				Content: text,
			}

			msg, err := json.Marshal(OutGoingMessage)
			if err != nil {
				fmt.Println("Failed to sent message (Encoding error)")
			}

			err = c.WriteMessage(websocket.TextMessage, msg)
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
