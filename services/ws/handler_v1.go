package main

import (
)

type handlerV1 struct {
	QueueSize int
}

func (h *handlerV1) Handle(ws wsConn, events <-chan *event) {
}
