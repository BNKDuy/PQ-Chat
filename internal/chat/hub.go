package chat

import (
	"log"
	"sort"
	"strings"
)

type Session struct {
	ID string
	P1 string
	P2 string
}

type Hub struct {
	// Online Client
	onlineClients map[string]*Client

	// Handle active Connection
	activeSessions map[string]*Session

	// buffer for current msg
	msgBuffer chan clientToHubMessage

	// Register a client when they are online
	registerClient chan *Client

	// Unregister a client when they disconnect
	unregisterClient chan *Client
}

var errUserMoreThanOneConnection = []byte(`{"From":"SERVER","To":"YOU","Type":"SYSTEM","Content":"You can only have one connection at a time"}`)
var errUserOffline = []byte(`{"From":"SERVER","To":"YOU","Type":"SYSTEM","Content":"The message was not delivered (the recipient is offline)."}`)
var errDeliveryFailed = []byte(`{"From":"SERVER","To":"YOU","Type":"SYSTEM","Content":"The message was not delivered (the recipient cannot recieve any message now). Please try again later."}`)
var msgStartMLKEM = []byte(`{"From":"SERVER","To":"YOU","Type":"MLKEM_INIT","Content":"Start ML-KEM"}`)
var msgStopEncryptedSession = []byte(`{"From":"SERVER","To":"YOU","Type":"STOP_ENCRYPTED_SESSION","Content":"Stop encrypted session."}`)

func NewHub() *Hub {
	return &Hub{
		onlineClients:    make(map[string]*Client),
		activeSessions:   make(map[string]*Session),
		msgBuffer:        make(chan clientToHubMessage),
		registerClient:   make(chan *Client),
		unregisterClient: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.registerClient:
			if oldConnection, ok := h.onlineClients[client.ID]; ok {
				close(oldConnection.send)
			}
			h.onlineClients[client.ID] = client
			recipient := client.Recipient

			// If other is online, start MKLEM
			if peer, ok := h.onlineClients[recipient]; ok && peer.Recipient == client.ID {
				h.sendServerMessageToClient(recipient, msgStartMLKEM)
			}
		case client := <-h.unregisterClient:
			if currentCLient, ok := h.onlineClients[client.ID]; ok {
				if client == currentCLient {
					close(client.send)
					delete(h.onlineClients, client.ID)
					h.sendServerMessageToClient(client.Recipient, msgStopEncryptedSession)
				}
			}
		case packet := <-h.msgBuffer:
			client, ok := h.onlineClients[packet.To]
			if !ok {
				select {
				case packet.From.send <- errUserOffline:
				default:
					log.Println("Client's send channel is full, dropping error message")
					close(packet.From.send)
					delete(h.onlineClients, packet.From.ID)
				}
				continue
			}

			select {
			case client.send <- packet.Message:
			default:
				close(client.send)
				delete(h.onlineClients, client.ID)
				select {
				case packet.From.send <- errDeliveryFailed:
				default:
					log.Println("Sender's send channel is full, dropping error message")
					close(packet.From.send)
					delete(h.onlineClients, packet.From.ID)
				}
			}
		}
	}
}

func getSessionID(user1, user2 string) string {
	users := []string{user1, user2}
	sort.Strings(users)
	return strings.Join(users, "::")
}

func (h *Hub) sendServerMessageToClient(username string, message []byte) {
	client, ok := h.onlineClients[username]
	if !ok {
		return
	}
	select {
	case client.send <- message:
	default:
		close(client.send)
		delete(h.onlineClients, username)
	}
}
