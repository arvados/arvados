package main

import (
	"io"
)

type handlerV1 struct {
	QueueSize int
}

func (h *handlerV1) Handle(ws io.ReadWriter, events <-chan *event) {
}
