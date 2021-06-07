// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package costanalyzer

import (
	"io"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

var Command = &command{}

type command struct {
	uuids      arrayFlags
	resultsDir string
	cache      bool
	begin      time.Time
	end        time.Time
}

type NoPrefixFormatter struct{}

func (f *NoPrefixFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message), nil
}

// RunCommand implements the subcommand "costanalyzer <collection> <collection> ..."
func (c *command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	logger := ctxlog.New(stderr, "text", "info")
	defer func() {
		if err != nil {
			logger.Error("\n" + err.Error() + "\n")
		}
	}()

	logger.SetFormatter(new(NoPrefixFormatter))

	exitcode, err := c.costAnalyzer(prog, args, logger, stdout, stderr)

	return exitcode
}
