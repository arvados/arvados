// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"

	"git.arvados.org/arvados.git/lib/cmdtest"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&CmdSuite{})

type CmdSuite struct{}

var testCmd = Multi(map[string]Handler{
	"echo": HandlerFunc(func(prog string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		fmt.Fprintln(stdout, strings.Join(args, " "))
		return 0
	}),
})

func (s *CmdSuite) TestHello(c *check.C) {
	defer cmdtest.LeakCheck(c)()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exited := testCmd.RunCommand("prog", []string{"echo", "hello", "world"}, bytes.NewReader(nil), stdout, stderr)
	c.Check(exited, check.Equals, 0)
	c.Check(stdout.String(), check.Equals, "hello world\n")
	c.Check(stderr.String(), check.Equals, "")
}

func (s *CmdSuite) TestHelloViaProg(c *check.C) {
	defer cmdtest.LeakCheck(c)()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exited := testCmd.RunCommand("/usr/local/bin/echo", []string{"hello", "world"}, bytes.NewReader(nil), stdout, stderr)
	c.Check(exited, check.Equals, 0)
	c.Check(stdout.String(), check.Equals, "hello world\n")
	c.Check(stderr.String(), check.Equals, "")
}

func (s *CmdSuite) TestUsage(c *check.C) {
	defer cmdtest.LeakCheck(c)()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exited := testCmd.RunCommand("prog", []string{"nosuchcommand", "hi"}, bytes.NewReader(nil), stdout, stderr)
	c.Check(exited, check.Equals, 2)
	c.Check(stdout.String(), check.Equals, "")
	c.Check(stderr.String(), check.Matches, `(?ms)^prog: unrecognized command "nosuchcommand"\n.*echo.*\n`)
}

func (s *CmdSuite) TestSubcommandToFront(c *check.C) {
	defer cmdtest.LeakCheck(c)()
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.String("format", "json", "")
	flags.Bool("n", false, "")
	args := SubcommandToFront([]string{"--format=yaml", "-n", "-format", "beep", "echo", "hi"}, flags)
	c.Check(args, check.DeepEquals, []string{"echo", "--format=yaml", "-n", "-format", "beep", "hi"})
}
