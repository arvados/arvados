package main

import (
	"io"
)

type handler interface {
	Handle(io.ReadWriter, <-chan *event)
}
