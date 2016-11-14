package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type Config struct {
	Client   arvados.Client
	Postgres pgConfig
	Listen   string
	Debug    bool

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
		ClientEventQueue: 64,
	}
}
