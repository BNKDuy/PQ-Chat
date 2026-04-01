package main

import (
	"net/http"
	"os"

	"github.com/BNKDuy/BroChat/internal/chat"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}
	hub := chat.NewHub()
	go hub.Run()
	http.HandleFunc("/ws/{username}/{recipient}", func(w http.ResponseWriter, r *http.Request) {
		chat.ServeWs(hub, w, r)
	})

	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hi"))
	})

	http.ListenAndServe(":"+port, nil)
}
