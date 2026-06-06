package main

import (
	"os"

	"github.com/BNKDuy/BroChat/internal/clientapp"
)

func main() {
	host := os.Args[1]
	username := os.Args[2]
	recipient := os.Args[3]

	url := host + "/ws/" + username + "/" + recipient

	app := clientapp.NewChatClient(url, username, recipient)
	app.Start()
}
