package main

import (
	"context"

	log "github.com/Sirupsen/logrus"
)

var loggerCtxKey = new(int)

func contextWithLogger(ctx context.Context, logger *log.Entry) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

func logger(ctx context.Context) *log.Entry {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerCtxKey).(*log.Entry); ok {
			return logger
		}
	}
	return log.WithFields(nil)
}
