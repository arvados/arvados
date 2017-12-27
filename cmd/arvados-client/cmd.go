// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"

	"git.curoverse.com/arvados.git/lib/cli"
	"git.curoverse.com/arvados.git/lib/cmd"
	"rsc.io/getopt"
)

var version = "dev"

var Run = cmd.Multi(map[string]cmd.RunFunc{
	"get":       cli.Get,
	"-e":        cmdVersion,
	"version":   cmdVersion,
	"-version":  cmdVersion,
	"--version": cmdVersion,
})

func cmdVersion(prog string, args []string, _ io.Reader, stdout, _ io.Writer) int {
	prog = regexp.MustCompile(` -*version$`).ReplaceAllLiteralString(prog, "")
	fmt.Fprintf(stdout, "%s %s (%s)\n", prog, version, runtime.Version())
	return 0
}

func fixLegacyArgs(args []string) []string {
	flags := getopt.NewFlagSet("", flag.ContinueOnError)
	flags.Bool("dry-run", false, "dry run")
	flags.Alias("n", "dry-run")
	flags.String("format", "json", "output format")
	flags.Alias("f", "format")
	flags.Bool("short", false, "short")
	flags.Alias("s", "short")
	flags.Bool("verbose", false, "verbose")
	flags.Alias("v", "verbose")
	return cmd.SubcommandToFront(args, flags)
}

func main() {
	os.Exit(Run(os.Args[0], fixLegacyArgs(os.Args[1:]), os.Stdin, os.Stdout, os.Stderr))
}
