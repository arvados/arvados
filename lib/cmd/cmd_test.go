// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"git.curoverse.com/arvados.git/lib/cmdtest"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&CmdSuite{})

type CmdSuite struct{}

var testCmd = Multi(map[string]RunFunc{
	"echo": func(prog string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
		fmt.Fprintln(stdout, strings.Join(args, " "))
		return 0
	},
})

func (s *CmdSuite) TestHello(c *check.C) {
	defer cmdtest.LeakCheck(c)()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exited := testCmd("prog", []string{"echo", "hello", "world"}, bytes.NewReader(nil), stdout, stderr)
	c.Check(exited, check.Equals, 0)
	c.Check(stdout.String(), check.Equals, "hello world\n")
	c.Check(stderr.String(), check.Equals, "")
}

func (s *CmdSuite) TestUsage(c *check.C) {
	defer cmdtest.LeakCheck(c)()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exited := testCmd("prog", []string{"nosuchcommand", "hi"}, bytes.NewReader(nil), stdout, stderr)
	c.Check(exited, check.Equals, 2)
	c.Check(stdout.String(), check.Equals, "")
	c.Check(stderr.String(), check.Matches, `(?ms)^unrecognized command "nosuchcommand"\n.*echo.*\n`)
}

func (s *CmdSuite) TestWithLateSubcommand(c *check.C) {
	defer cmdtest.LeakCheck(c)()
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	run := WithLateSubcommand(testCmd, []string{"format", "f"}, []string{"n"})
	exited := run("prog", []string{"--format=yaml", "-n", "-format", "beep", "echo", "hi"}, bytes.NewReader(nil), stdout, stderr)
	c.Check(exited, check.Equals, 0)
	c.Check(stdout.String(), check.Equals, "--format=yaml -n -format beep hi\n")
	c.Check(stderr.String(), check.Equals, "")
}
