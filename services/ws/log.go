package main

import (
	"context"

	"github.com/Sirupsen/logrus"
)

var (
	loggerCtxKey = new(int)
	rootLogger   = logrus.New()
)

func contextWithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

func logger(ctx context.Context) *logrus.Entry {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerCtxKey).(*logrus.Entry); ok {
			return logger
		}
	}
	return rootLogger.WithFields(nil)
}
