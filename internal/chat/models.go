package chat

type ChatMessage struct {
	From    string `json:"From"`
	To      string `json:"To"`
	Content string `json:"Content"`
}

type clientToHubMessage struct {
	To      string
	From    *Client
	Message []byte
}
