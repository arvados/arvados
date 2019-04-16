// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"fmt"
	"io"

	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
)

var DumpCommand cmd.Handler = dumpCommand{}

type dumpCommand struct{}

func (dumpCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	defer func() {
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
	}()
	if len(args) != 0 {
		err = fmt.Errorf("usage: %s <config-src.yaml >config-min.yaml", prog)
		return 2
	}
	log := ctxlog.New(stderr, "text", "info")
	cfg, err := Load(stdin, log)
	if err != nil {
		return 1
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return 1
	}
	_, err = stdout.Write(out)
	if err != nil {
		return 1
	}
	return 0
}
