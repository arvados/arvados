// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package deduplicationreport

import (
	"io"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

var Command command

type command struct{}

type NoPrefixFormatter struct{}

func (f *NoPrefixFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

// RunCommand implements the subcommand "deduplication-report <collection> <collection> ..."
func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	logger := ctxlog.New(stderr, "text", "info")
	defer func() {
		if err != nil {
			logger.WithError(err).Error("fatal")
		}
	}()

	logger.SetFormatter(new(NoPrefixFormatter))

	exitcode := report(prog, args, logger, stdout, stderr)

	return exitcode
}
