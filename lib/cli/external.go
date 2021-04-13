// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"syscall"

	"git.arvados.org/arvados.git/lib/cmd"
)

var (
	Create = rubyArvCmd{"create"}
	Edit   = rubyArvCmd{"edit"}

	Copy = externalCmd{"arv-copy"}
	Tag  = externalCmd{"arv-tag"}
	Ws   = externalCmd{"arv-ws"}

	Keep = cmd.Multi(map[string]cmd.Handler{
		"get":       externalCmd{"arv-get"},
		"put":       externalCmd{"arv-put"},
		"ls":        externalCmd{"arv-ls"},
		"normalize": externalCmd{"arv-normalize"},
		"docker":    externalCmd{"arv-keepdocker"},
	})
	// user, group, container, specimen, etc.
	APICall = apiCallCmd{}
)

// When using the ruby "arv" command, flags must come before the
// subcommand: "arv --format=yaml get foo" works, but "arv get
// --format=yaml foo" does not work.
func legacyFlagsToFront(subcommand string, argsin []string) (argsout []string) {
	flags, _ := LegacyFlagSet()
	flags.SetOutput(ioutil.Discard)
	flags.Parse(argsin)
	narg := flags.NArg()
	argsout = append(argsout, argsin[:len(argsin)-narg]...)
	argsout = append(argsout, subcommand)
	argsout = append(argsout, argsin[len(argsin)-narg:]...)
	return
}

type apiCallCmd struct{}

func (cmd apiCallCmd) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	split := strings.Split(prog, " ")
	if len(split) < 2 {
		fmt.Fprintf(stderr, "internal error: no api model in %q\n", prog)
		return 2
	}
	model := split[len(split)-1]
	return externalCmd{"arv"}.RunCommand("arv", legacyFlagsToFront(model, args), stdin, stdout, stderr)
}

type rubyArvCmd struct {
	subcommand string
}

func (rc rubyArvCmd) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return externalCmd{"arv"}.RunCommand("arv", legacyFlagsToFront(rc.subcommand, args), stdin, stdout, stderr)
}

type externalCmd struct {
	prog string
}

func (ec externalCmd) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cmd := exec.Command(ec.prog, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	switch err := err.(type) {
	case nil:
		return 0
	case *exec.ExitError:
		status := err.Sys().(syscall.WaitStatus)
		if status.Exited() {
			return status.ExitStatus()
		}
		fmt.Fprintf(stderr, "%s failed: %s\n", ec.prog, err)
		return 1
	case *exec.Error:
		fmt.Fprintln(stderr, err)
		if ec.prog == "arv" {
			fmt.Fprint(stderr, rubyInstallHints)
		} else if strings.HasPrefix(ec.prog, "arv-") {
			fmt.Fprint(stderr, pythonInstallHints)
		}
		return 1
	default:
		fmt.Fprintf(stderr, "error running %s: %s\n", ec.prog, err)
		return 1
	}
}

var (
	rubyInstallHints = `
Note: This subcommand uses the arvados-cli Ruby gem. If that is not
installed, try "gem install arvados-cli", or see
https://doc.arvados.org/install for more details.

`
	pythonInstallHints = `
Note: This subcommand uses the "arvados" Python module. If that is
not installed, try:
* "pip install arvados" (either as root or in a virtualenv), or
* "sudo apt-get install python3-arvados-python-client", or
* see https://doc.arvados.org/install for more details.

`
)
