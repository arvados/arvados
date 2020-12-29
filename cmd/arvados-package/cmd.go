// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"os"

	"git.arvados.org/arvados.git/lib/cmd"
)

var (
	handler = cmd.Multi(map[string]cmd.Handler{
		"version":   cmd.Version,
		"-version":  cmd.Version,
		"--version": cmd.Version,

		"build": build{},
	})
)

func main() {
	os.Exit(handler.RunCommand(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
