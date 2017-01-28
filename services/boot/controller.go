package main

import (
	"context"
)

type controller struct{}

func (c *controller) Boot(ctx context.Context) error {
	return Series{
		Concurrent{
			cfg(ctx),
			installCerts,
		},
		Concurrent{
			consul,
		},
	}.Boot(ctx)
}
