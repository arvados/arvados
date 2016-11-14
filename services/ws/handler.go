package main

import (
	"io"
	"net/http"
)

type handler interface {
	Handle(wsConn, <-chan *event)
}

type wsConn interface {
	io.ReadWriter
	Request() *http.Request
}
