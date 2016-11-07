package main

import (
	"log"
)

func init() {
	theConfig.debugLogf = log.Printf
}
