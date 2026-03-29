package main

import (
	"net/http"

	"github.com/BNKDuy/BroChat/internal/chat"
)

func main() {
	hub := chat.NewHub()
	go hub.Run()
	http.HandleFunc("/ws/{username}", func(w http.ResponseWriter, r *http.Request) {
		chat.ServeWs(hub, w, r)
	})
	http.ListenAndServe(":8080", nil)
}
