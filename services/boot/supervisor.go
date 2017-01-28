package main

import (
	"context"
	"os"
)

type supervisor interface {
	Running(ctx context.Context) (bool, error)
	Start(ctx context.Context) error
}

func newSupervisor(ctx context.Context, name, cmd string, args ...string) supervisor {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return &systemdUnit{
			name: name,
			cmd:  cmd,
			args: args,
		}
	}
	return &runitService{
		name: name,
		cmd:  cmd,
		args: args,
	}
}

