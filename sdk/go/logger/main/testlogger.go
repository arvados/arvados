// This binary tests the logger package.
// It's not a standard unit test. Instead it writes to the actual log
// and you have to clean up after it.

package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"log"
)

func main() {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %v", err)
	}

	l := logger.NewLogger(logger.LoggerParams{Client: arv,
		EventType: "experimental-logger-testing",
		// No minimum write interval
	})

	{
		properties, _ := l.Edit()
		properties["Ninja"] = "Misha"
	}
	l.Record()
}
