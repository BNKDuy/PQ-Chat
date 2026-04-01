package chat

type MessageType string

const (
	MessageTypeChat                  MessageType = "CHAT"
	MessageTypeSystem                MessageType = "SYSTEM"
	MessageTypeMLKEMInit             MessageType = "MLKEM_INIT"
	MessageTypeMLKEMEncapsulationKey MessageType = "MLKEM_ENCAPSULATION_KEY"
	MessageTypeMLKEMCiphertext       MessageType = "MLKEM_CIPHERTEXT"
	MessageTypeStopEncryptedSession  MessageType = "STOP_ENCRYPTED_SESSION"
)

type ChatMessage struct {
	From    string      `json:"From"`
	To      string      `json:"To"`
	Type    MessageType `json:"Type"`
	Content string      `json:"Content"`
}

type clientToHubMessage struct {
	To      string
	From    *Client
	Message []byte
}

func (m MessageType) IsValid() bool {
	switch m {
	case MessageTypeChat,
		MessageTypeSystem,
		MessageTypeMLKEMInit,
		MessageTypeMLKEMCiphertext,
		MessageTypeStopEncryptedSession,
		MessageTypeMLKEMEncapsulationKey:
		return true
	}
	return false
}
