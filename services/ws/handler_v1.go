package main

import (
	"golang.org/x/net/websocket"
)

type handlerV1 struct {
	QueueSize int
}

func (h *handlerV1) Handle(ws *websocket.Conn, events <-chan *event) {
}
