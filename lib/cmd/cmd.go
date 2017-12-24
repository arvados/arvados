// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package cmd defines a RunFunc type, representing a process that can
// be invoked from a command line.
package cmd

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
)

// A RunFunc runs a command with the given args, and returns an exit
// code.
type RunFunc func(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int

// Multi returns a RunFunc that looks up its first argument in m, and
// invokes the resulting RunFunc with the remaining args.
//
// Example:
//
//     os.Exit(Multi(map[string]RunFunc{
//             "foobar": func(prog string, args []string) int {
//                     fmt.Println(args[0])
//                     return 2
//             },
//     })("/usr/bin/multi", []string{"foobar", "baz"}))
//
// ...prints "baz" and exits 2.
func Multi(m map[string]RunFunc) RunFunc {
	return func(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
		if len(args) < 1 {
			fmt.Fprintf(stderr, "usage: %s command [args]\n", prog)
			multiUsage(stderr, m)
			return 2
		}
		if cmd, ok := m[args[0]]; !ok {
			fmt.Fprintf(stderr, "unrecognized command %q\n", args[0])
			multiUsage(stderr, m)
			return 2
		} else {
			return cmd(prog+" "+args[0], args[1:], stdin, stdout, stderr)
		}
	}
}

func multiUsage(stderr io.Writer, m map[string]RunFunc) {
	var subcommands []string
	for sc := range m {
		if strings.HasPrefix(sc, "-") {
			// Some subcommands have alternate versions
			// like "--version" for compatibility. Don't
			// clutter the subcommand summary with those.
			continue
		}
		subcommands = append(subcommands, sc)
	}
	sort.Strings(subcommands)
	fmt.Fprintf(stderr, "\nAvailable commands:\n")
	for _, sc := range subcommands {
		fmt.Fprintf(stderr, "    %s\n", sc)
	}
}

// WithLateSubcommand wraps a RunFunc by skipping over some known
// flags to find a subcommand, and moving that subcommand to the front
// of the args before calling the wrapped RunFunc. For example:
//
//	// Translate [           --format foo subcommand bar]
//	//        to [subcommand --format foo            bar]
//	WithLateSubcommand(fn, []string{"format"}, nil)
func WithLateSubcommand(run RunFunc, argFlags, boolFlags []string) RunFunc {
	return func(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
		flags := flag.NewFlagSet("prog", flag.ContinueOnError)
		for _, arg := range argFlags {
			flags.String(arg, "", "")
		}
		for _, arg := range boolFlags {
			flags.Bool(arg, false, "")
		}
		// Ignore errors. We can't report a useful error
		// message anyway.
		flags.SetOutput(ioutil.Discard)
		flags.Usage = func() {}
		flags.Parse(args)
		if flags.NArg() > 0 {
			// Move the first arg after the recognized
			// flags up to the front.
			flagargs := len(args) - flags.NArg()
			newargs := make([]string, len(args))
			newargs[0] = args[flagargs]
			copy(newargs[1:flagargs+1], args[:flagargs])
			copy(newargs[flagargs+1:], args[flagargs+1:])
			args = newargs
		}
		return run(prog, args, stdin, stdout, stderr)
	}
}
