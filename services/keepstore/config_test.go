package main

import (
	log "github.com/Sirupsen/logrus"
)

func init() {
	theConfig.debugLogf = log.Printf
}
