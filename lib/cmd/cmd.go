// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package cmd helps define reusable functions that can be exposed as
// [subcommands of] command line programs.
package cmd

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type Handler interface {
	RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int
}

type HandlerFunc func(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int

func (f HandlerFunc) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return f(prog, args, stdin, stdout, stderr)
}

type Version string

func (v Version) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	prog = regexp.MustCompile(` -*version$`).ReplaceAllLiteralString(prog, "")
	fmt.Fprintf(stdout, "%s %s (%s)\n", prog, v, runtime.Version())
	return 0
}

// Multi is a Handler that looks up its first argument in a map, and
// invokes the resulting Handler with the remaining args.
//
// Example:
//
//     os.Exit(Multi(map[string]Handler{
//             "foobar": HandlerFunc(func(prog string, args []string) int {
//                     fmt.Println(args[0])
//                     return 2
//             }),
//     })("/usr/bin/multi", []string{"foobar", "baz"}))
//
// ...prints "baz" and exits 2.
type Multi map[string]Handler

func (m Multi) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintf(stderr, "usage: %s command [args]\n", prog)
		m.Usage(stderr)
		return 2
	}
	_, basename := filepath.Split(prog)
	if strings.HasPrefix(basename, "arvados-") {
		basename = basename[8:]
	} else if strings.HasPrefix(basename, "crunch-") {
		basename = basename[7:]
	}
	if cmd, ok := m[basename]; ok {
		return cmd.RunCommand(prog, args, stdin, stdout, stderr)
	} else if cmd, ok = m[args[0]]; ok {
		return cmd.RunCommand(prog+" "+args[0], args[1:], stdin, stdout, stderr)
	} else {
		fmt.Fprintf(stderr, "%s: unrecognized command %q\n", prog, args[0])
		m.Usage(stderr)
		return 2
	}
}

func (m Multi) Usage(stderr io.Writer) {
	fmt.Fprintf(stderr, "\nAvailable commands:\n")
	m.listSubcommands(stderr, "")
}

func (m Multi) listSubcommands(out io.Writer, prefix string) {
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
	for _, sc := range subcommands {
		switch cmd := m[sc].(type) {
		case Multi:
			cmd.listSubcommands(out, prefix+sc+" ")
		default:
			fmt.Fprintf(out, "    %s%s\n", prefix, sc)
		}
	}
}

type FlagSet interface {
	Init(string, flag.ErrorHandling)
	Args() []string
	NArg() int
	Parse([]string) error
	SetOutput(io.Writer)
	PrintDefaults()
}

// SubcommandToFront silently parses args using flagset, and returns a
// copy of args with the first non-flag argument moved to the
// front. If parsing fails or consumes all of args, args is returned
// unchanged.
//
// SubcommandToFront invokes methods on flagset that have side
// effects, including Parse. In typical usage, flagset will not used
// for anything else after being passed to SubcommandToFront.
func SubcommandToFront(args []string, flagset FlagSet) []string {
	flagset.Init("", flag.ContinueOnError)
	flagset.SetOutput(ioutil.Discard)
	if err := flagset.Parse(args); err != nil || flagset.NArg() == 0 {
		// No subcommand found.
		return args
	}
	// Move subcommand to the front.
	flagargs := len(args) - flagset.NArg()
	newargs := make([]string, len(args))
	newargs[0] = args[flagargs]
	copy(newargs[1:flagargs+1], args[:flagargs])
	copy(newargs[flagargs+1:], args[flagargs+1:])
	return newargs
}
