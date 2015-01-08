// This binary tests the logger package.
// It's not a standard unit test. Instead it writes to the actual log
// and you have to clean up after it.

package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"log"
)

const (
	eventType string = "experimental-logger-testing"
)


func main() {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %v", err)
	}

	l := logger.NewLogger(logger.LoggerParams{Client: arv,
		EventType: eventType,
		// No minimum write interval
	})

	logData := l.Acquire()
	logData["Ninja"] = "Misha"
	logData = l.Release()
}
