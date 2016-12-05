package main

import (
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type wsConfig struct {
	Client    arvados.Client
	Postgres  pgConfig
	Listen    string
	LogLevel  string
	LogFormat string

	PingTimeout      arvados.Duration
	ClientEventQueue int
	ServerEventQueue int
}

func defaultConfig() wsConfig {
	return wsConfig{
		Client: arvados.Client{
			APIHost: "localhost:443",
		},
		Postgres: pgConfig{
			"dbname":          "arvados_production",
			"user":            "arvados",
			"password":        "xyzzy",
			"host":            "localhost",
			"connect_timeout": "30",
			"sslmode":         "require",
		},
		LogLevel:         "info",
		LogFormat:        "json",
		PingTimeout:      arvados.Duration(time.Minute),
		ClientEventQueue: 64,
		ServerEventQueue: 4,
	}
}
