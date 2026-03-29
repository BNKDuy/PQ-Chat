package chat

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID                  string
	hub                 *Hub
	websocketConnection *websocket.Conn
	send                chan []byte
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 3 << 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregisterClient <- c
		c.websocketConnection.Close()
	}()
	c.websocketConnection.SetReadLimit(maxMessageSize)
	c.websocketConnection.SetReadDeadline(time.Now().Add(pongWait))
	c.websocketConnection.SetPongHandler(func(string) error {
		c.websocketConnection.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.websocketConnection.ReadMessage()
		if err != nil {
			log.Println("Failed to read message from socket: ", err)
			return
		}
		c.hub.msgBuffer <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.websocketConnection.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.websocketConnection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.websocketConnection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			nextWriter, err := c.websocketConnection.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println("Failed to open a new NextWriter: ", err)
				return
			}
			nextWriter.Write(message)

			n := len(c.send)
			for range n {
				nextWriter.Write([]byte{'\n'})
				nextWriter.Write(<-c.send)
			}

			if err := nextWriter.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.websocketConnection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.websocketConnection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Failed to wirte ping message: ", err)
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	username := r.PathValue("username")

	if username == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	websocketConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to Upgrade connection to Websocket: ", err)
		return
	}

	client := &Client{
		ID:                  username,
		hub:                 hub,
		websocketConnection: websocketConnection,
		send:                make(chan []byte, 256),
	}
	client.hub.registerClient <- client

	go client.readPump()
	go client.writePump()
}
