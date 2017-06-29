// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package ctxlog

import (
	"context"

	"github.com/Sirupsen/logrus"
)

var (
	loggerCtxKey = new(int)
	rootLogger   = logrus.New()
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

// Context returns a new child context such that FromContext(child)
// returns the given logger.
func Context(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

// FromContext returns the logger suitable for the given context -- the one
// attached by contextWithLogger() if applicable, otherwise the
// top-level logger with no fields/values.
func FromContext(ctx context.Context) *logrus.Entry {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerCtxKey).(*logrus.Entry); ok {
			return logger
		}
	}
	return rootLogger.WithFields(nil)
}

// SetLevel sets the current logging level. See logrus for level
// names.
func SetLevel(level string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Fatal(err)
	}
	rootLogger.Level = lvl
}

// SetFormat sets the current logging format to "json" or "text".
func SetFormat(format string) {
	switch format {
	case "text":
		rootLogger.Formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: rfc3339NanoFixed,
		}
	case "json":
		rootLogger.Formatter = &logrus.JSONFormatter{
			TimestampFormat: rfc3339NanoFixed,
		}
	default:
		logrus.WithField("LogFormat", format).Fatal("unknown log format")
	}
}
