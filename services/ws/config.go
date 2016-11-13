package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type Config struct {
	Client   arvados.Client
	Postgres pgConfig
	Listen   string

	ClientEventQueue int
	ServerEventQueue int
}

func DefaultConfig() Config {
	return Config{
		ClientEventQueue: 64,
	}
}
