package main

import (
	"context"
	"time"
)

func waitCheck(ctx context.Context, timeout time.Duration, check func(ctx context.Context) error) error {
	ctx, _ = context.WithTimeout(ctx, timeout)
	var err error
	for err = check(ctx); err != nil && ctx.Err() == nil; err = check(ctx) {
		time.Sleep(time.Second)
	}
	return err
}
