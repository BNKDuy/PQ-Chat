package chat

type ChatMessage struct {
	From    string `json:"From"`
	To      string `json:"To"`
	Content []byte `json:"Content"`
}
