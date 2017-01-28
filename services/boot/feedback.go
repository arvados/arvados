package main

import (
	"context"
	"fmt"
	"log"
)

func feedbackf(ctx context.Context, f string, args ...interface{}) func() {
	msg := fmt.Sprintf(f, args...)
	log.Print("start: ", msg)
	return func() {
		log.Print(" done: ", msg)
	}
}
