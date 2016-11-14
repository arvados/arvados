package main

import (
	"time"
)

type handlerV1 struct {
	PingTimeout time.Duration
	QueueSize   int
}

func (h *handlerV1) Handle(ws wsConn, events <-chan *event) {
}
