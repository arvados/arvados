// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
)

var DumpCommand dumpCommand

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

var CheckCommand checkCommand

type checkCommand struct{}

func (checkCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	defer func() {
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
	}()
	if len(args) != 0 {
		err = fmt.Errorf("usage: %s <config-src.yaml && echo 'no changes needed'", prog)
		return 2
	}
	log := &plainLogger{w: stderr}
	buf, err := ioutil.ReadAll(stdin)
	if err != nil {
		return 1
	}
	withoutDepr, err := load(bytes.NewBuffer(buf), log, false)
	if err != nil {
		return 1
	}
	withDepr, err := load(bytes.NewBuffer(buf), nil, true)
	if err != nil {
		return 1
	}
	cmd := exec.Command("diff", "-u", "--label", "without-deprecated-configs", "--label", "relying-on-deprecated-configs", "/dev/fd/3", "/dev/fd/4")
	for _, obj := range []interface{}{withoutDepr, withDepr} {
		y, _ := yaml.Marshal(obj)
		pr, pw, err := os.Pipe()
		if err != nil {
			return 1
		}
		defer pr.Close()
		go func() {
			io.Copy(pw, bytes.NewBuffer(y))
			pw.Close()
		}()
		cmd.ExtraFiles = append(cmd.ExtraFiles, pr)
	}
	diff, err := cmd.CombinedOutput()
	if bytes.HasPrefix(diff, []byte("--- ")) {
		fmt.Fprintln(stdout, "Your configuration is relying on deprecated entries. Suggest making the following changes.")
		stdout.Write(diff)
		return 1
	} else if len(diff) > 0 {
		fmt.Fprintf(stderr, "Unexpected diff output:\n%s", diff)
		return 1
	} else if err != nil {
		return 1
	}
	if log.used {
		return 1
	}
	return 0
}

type plainLogger struct {
	w    io.Writer
	used bool
}

func (pl *plainLogger) Warnf(format string, args ...interface{}) {
	pl.used = true
	fmt.Fprintf(pl.w, format+"\n", args...)
}
