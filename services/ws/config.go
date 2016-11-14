package main

import (
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type Config struct {
	Client   arvados.Client
	Postgres pgConfig
	Listen   string
	Debug    bool

	PingTimeout      arvados.Duration
	ClientEventQueue int
	ServerEventQueue int
}

func DefaultConfig() Config {
	return Config{
		Client: arvados.Client{
			APIHost: "localhost:443",
		},
		Postgres: pgConfig{
			"dbname":          "arvados_test",
			"user":            "arvados",
			"password":        "xyzzy",
			"host":            "localhost",
			"connect_timeout": "30",
			"sslmode":         "disable",
		},
		PingTimeout:      arvados.Duration(time.Minute),
		ClientEventQueue: 64,
		ServerEventQueue: 4,
	}
}
