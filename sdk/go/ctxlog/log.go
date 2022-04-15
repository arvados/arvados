// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package ctxlog

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	loggerCtxKey = new(int)
	rootLogger   = logrus.New()
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

// Context returns a new child context such that FromContext(child)
// returns the given logger.
func Context(ctx context.Context, logger logrus.FieldLogger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

// FromContext returns the logger suitable for the given context -- the one
// attached by contextWithLogger() if applicable, otherwise the
// top-level logger with no fields/values.
func FromContext(ctx context.Context) logrus.FieldLogger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerCtxKey).(logrus.FieldLogger); ok {
			return logger
		}
	}
	return rootLogger.WithFields(nil)
}

// New returns a new logger with the indicated format and
// level.
func New(out io.Writer, format, level string) *logrus.Logger {
	logger := logrus.New()
	logger.Out = out
	setFormat(logger, format)
	setLevel(logger, level)
	return logger
}

func TestLogger(c interface{ Log(...interface{}) }) *logrus.Logger {
	logger := logrus.New()
	logger.Out = &logWriter{c.Log}
	setFormat(logger, "text")
	if d := os.Getenv("ARVADOS_DEBUG"); d != "0" && d != "" {
		setLevel(logger, "debug")
	} else {
		setLevel(logger, "info")
	}
	return logger
}

// LogWriter returns an io.Writer that writes to the given log func,
// which is typically (*check.C).Log().
func LogWriter(log func(...interface{})) io.Writer {
	return &logWriter{log}
}

// SetLevel sets the current logging level. See logrus for level
// names.
func SetLevel(level string) {
	setLevel(rootLogger, level)
}

func setLevel(logger *logrus.Logger, level string) {
	if level == "" {
	} else if lvl, err := logrus.ParseLevel(level); err != nil {
		logrus.WithField("Level", level).Fatal("unknown log level")
	} else {
		logger.Level = lvl
	}
}

// SetFormat sets the current logging format to "json" or "text".
func SetFormat(format string) {
	setFormat(rootLogger, format)
}

func setFormat(logger *logrus.Logger, format string) {
	switch format {
	case "text":
		logger.Formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: rfc3339NanoFixed,
		}
	case "plain":
		logger.Formatter = &logrus.TextFormatter{
			DisableColors:    true,
			DisableTimestamp: true,
		}
	case "json", "":
		logger.Formatter = &logrus.JSONFormatter{
			TimestampFormat: rfc3339NanoFixed,
		}
	default:
		logrus.WithField("Format", format).Fatal("unknown log format")
	}
}

// logWriter is an io.Writer that writes by calling a "write log"
// function, typically (*check.C)Log().
type logWriter struct {
	logfunc func(...interface{})
}

func (tl *logWriter) Write(buf []byte) (int, error) {
	tl.logfunc(string(bytes.TrimRight(buf, "\n")))
	return len(buf), nil
}
