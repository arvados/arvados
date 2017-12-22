// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package cmd defines a RunFunc type, representing a process that can
// be invoked from a command line.
package cmd

import (
	"fmt"
	"io"
)

// A RunFunc runs a command with the given args, and returns an exit
// code.
type RunFunc func(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int

// Multi returns a command that looks up its first argument in m, and
// runs the resulting RunFunc with the remaining args.
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
			fmt.Fprintf(stderr, "usage: %s command [args]", prog)
			return 2
		}
		if cmd, ok := m[args[0]]; !ok {
			fmt.Fprintf(stderr, "unrecognized command %q", args[0])
			return 2
		} else {
			return cmd(prog+" "+args[0], args[1:], stdin, stdout, stderr)
		}
	}
}
