package main

import (
	"context"
)

type controller struct{}

func (c *controller) Boot(ctx context.Context) error {
	return Concurrent{
		consul,
	}.Boot(ctx)
}
