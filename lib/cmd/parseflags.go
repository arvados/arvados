// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"flag"
	"fmt"
	"io"
	"reflect"
)

// Hack to enable checking whether a given FlagSet's Usage method is
// the (private) default one.
var defaultFlagSet = flag.NewFlagSet("none", flag.ContinueOnError)

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
			return false, EXIT_INVALIDARGUMENT
		}
		return true, 0
	case flag.ErrHelp:
		// Use our own default usage func, not the one
		// provided by the flag pkg, if the caller hasn't set
		// one. (We use reflect to determine whether f.Usage
		// is the private defaultUsage func that
		// flag.NewFlagSet uses.)
		if f, ok := f.(*flag.FlagSet); ok && f.Usage != nil && reflect.ValueOf(f.Usage).String() != reflect.ValueOf(defaultFlagSet.Usage).String() {
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
		return false, EXIT_INVALIDARGUMENT
	}
}
