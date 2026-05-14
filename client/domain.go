package main

import "time"

type Key [256]byte
type ExchangeKey struct {
	keyID          string
	key            Key
	expirationDate time.Time
}
type MessageKey struct {
	keyID      string
	messageSeq int
	key        Key
}

type Message struct {
	KeyID   string
	SeqID   string
	Content []byte
}

type MessageKeyChain struct {
	head *MessageKeyChainNode
}

type MessageKeyChainNode struct {
	head uint32
}
