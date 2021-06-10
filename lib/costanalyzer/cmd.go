// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package costanalyzer

import (
	"io"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

var Command command

type command struct{}

// RunCommand implements the subcommand "costanalyzer <collection> <collection> ..."
func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	logger := ctxlog.New(stderr, "text", "info")
	logger.SetFormatter(cmd.NoPrefixFormatter{})
	defer func() {
		if err != nil {
			logger.Error("\n" + err.Error())
		}
	}()

	loader := config.NewLoader(stdin, logger)
	loader.SkipLegacy = true

	exitcode, err := costanalyzer(prog, args, loader, logger, stdout, stderr)

	return exitcode
}
