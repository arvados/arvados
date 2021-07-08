// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package deduplicationreport

import (
	"io"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

var Command command

type command struct{}

// RunCommand implements the subcommand "deduplication-report <collection> <collection> ..."
func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	logger := ctxlog.New(stderr, "text", "info")
	defer func() {
		if err != nil {
			logger.WithError(err).Error("fatal")
		}
	}()

	logger.SetFormatter(cmd.NoPrefixFormatter{})

	exitcode := report(prog, args, logger, stdout, stderr)

	return exitcode
}
