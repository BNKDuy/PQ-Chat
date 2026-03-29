package chat

import "encoding/json"

type Hub struct {
	// Online Client
	onlineClients map[string]*Client

	// buffer for current msg
	msgBuffer chan []byte

	// Register a client when they are online
	registerClient chan *Client

	// Unregister a client when they disconnect
	unregisterClient chan *Client
}

func NewHub() *Hub {
	return &Hub{
		onlineClients:    make(map[string]*Client),
		msgBuffer:        make(chan []byte),
		registerClient:   make(chan *Client),
		unregisterClient: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.registerClient:
			h.onlineClients[client.ID] = client
		case client := <-h.unregisterClient:
			if _, ok := h.onlineClients[client.ID]; ok {
				close(client.send)
				delete(h.onlineClients, client.ID)
			}
		case message := <-h.msgBuffer:
			var chatMsg ChatMessage
			err := json.Unmarshal(message, &chatMsg)
			if err != nil {
				continue
			}

			client, ok := h.onlineClients[chatMsg.To]
			if !ok {
				continue
			}

			select {
			case client.send <- chatMsg.Content:
			default:
				close(client.send)
				delete(h.onlineClients, client.ID)
			}
		}
	}
}
