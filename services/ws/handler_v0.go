package main

import (
	"io"
)

type handlerV0 struct {
	QueueSize int
}

func (h *handlerV0) Handle(ws io.ReadWriter, events <-chan *event) {
}
