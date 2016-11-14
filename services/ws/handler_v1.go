package main

import (
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type handlerV1 struct {
	Client      arvados.Client
	PingTimeout time.Duration
	QueueSize   int
}

func (h *handlerV1) Handle(ws wsConn, events <-chan *event) {
}
