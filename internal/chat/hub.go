package chat

import (
	"log"
)

type Hub struct {
	// Online Client
	onlineClients map[string]*Client

	// buffer for current msg
	msgBuffer chan clientToHubMessage

	// Register a client when they are online
	registerClient chan *Client

	// Unregister a client when they disconnect
	unregisterClient chan *Client
}

var errUserOffline = []byte(`{"From":"SERVER","To":"YOU","Content":"The message was not delivered (the recipient is offline)."}`)

func NewHub() *Hub {
	return &Hub{
		onlineClients:    make(map[string]*Client),
		msgBuffer:        make(chan clientToHubMessage),
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
		case packet := <-h.msgBuffer:
			client, ok := h.onlineClients[packet.To]
			if !ok {
				select {
				case packet.From.send <- errUserOffline:
				default:
					log.Println("Client's send channel is full, dropping error message")
				}
				continue
			}

			select {
			case client.send <- packet.Message:
			default:
				close(client.send)
				delete(h.onlineClients, client.ID)
			}
		}
	}
}
