package main

import (
	"io"
	"net/http"
	"time"
)

type handler interface {
	Handle(wsConn, <-chan *event)
}

type wsConn interface {
	io.ReadWriter
	Request() *http.Request
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
