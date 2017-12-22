// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"

	"git.curoverse.com/arvados.git/lib/cli"
	"git.curoverse.com/arvados.git/lib/cmd"
)

var version = "dev"

var Run = cmd.Multi(map[string]cmd.RunFunc{
	"get":       cli.Get,
	"version":   cmdVersion,
	"-version":  cmdVersion,
	"--version": cmdVersion,
})

func cmdVersion(prog string, args []string, _ io.Reader, stdout, _ io.Writer) int {
	prog = regexp.MustCompile(` -*version$`).ReplaceAllLiteralString(prog, "")
	fmt.Fprintf(stdout, "%s %s (%s)\n", prog, version, runtime.Version())
	return 0
}

func main() {
	os.Exit(Run(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
