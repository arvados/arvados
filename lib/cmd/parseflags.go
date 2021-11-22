// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"flag"
	"fmt"
	"io"
)

// ParseFlags calls f.Parse(args) and prints appropriate error/help
// messages to stderr.
//
// The positional argument is "" if no positional arguments are
// accepted, otherwise a string to print with the usage message,
// "Usage: {prog} [options] {positional}".
//
// The first return value, ok, is true if the program should continue
// running normally, or false if it should exit now.
//
// If ok is false, the second return value is an appropriate exit
// code: 0 if "-help" was given, 2 if there was a usage error.
func ParseFlags(f FlagSet, prog string, args []string, positional string, stderr io.Writer) (ok bool, exitCode int) {
	f.Init(prog, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	err := f.Parse(args)
	switch err {
	case nil:
		if f.NArg() > 0 && positional == "" {
			fmt.Fprintf(stderr, "unrecognized command line arguments: %v (try -help)\n", f.Args())
			return false, 2
		}
		return true, 0
	case flag.ErrHelp:
		if f, ok := f.(*flag.FlagSet); ok && f.Usage != nil {
			f.SetOutput(stderr)
			f.Usage()
		} else {
			fmt.Fprintf(stderr, "Usage: %s [options] %s\n", prog, positional)
			f.SetOutput(stderr)
			f.PrintDefaults()
		}
		return false, 0
	default:
		fmt.Fprintf(stderr, "error parsing command line arguments: %s (try -help)\n", err)
		return false, 2
	}
}
