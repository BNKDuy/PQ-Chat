package clientapp

import (
	"fmt"

	"github.com/gorilla/websocket"
)

func (c *ChatClient) newSocketConnection(url string) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("Dial error:", err)
	}
	c.conn = conn
	return err
}

func (c *ChatClient) safeWrite(message []byte) error {
	c.socketWriteMu.Lock()
	defer c.socketWriteMu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, message)
}
