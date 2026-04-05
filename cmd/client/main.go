package main

import (
	"log"
	"os"

	"github.com/BNKDuy/BroChat/internal/clientapp"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	username := os.Args[1]
	recipient := os.Args[2]

	var url string
	environment := os.Getenv("ENV")
	host := os.Getenv("Host")

	if environment == "local" {
		url = "ws://" + host + "/ws/" + username + "/" + recipient
	} else {
		url = "wss://" + host + "/ws/" + username + "/" + recipient
	}

	app := clientapp.NewChatClient(url, username, recipient)
	app.Start()
}
