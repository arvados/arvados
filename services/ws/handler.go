package main

import (
	"golang.org/x/net/websocket"
)

type handler interface {
	Handle(*websocket.Conn, <-chan *event)
}
