// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	"git.arvados.org/arvados.git/lib/cmd"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&ClientSuite{})

type ClientSuite struct{}

func (s *ClientSuite) TestBadCommand(c *check.C) {
	exited := handler.RunCommand("arvados-client", []string{"no such command"}, bytes.NewReader(nil), ioutil.Discard, ioutil.Discard)
	c.Check(exited, check.Equals, cmd.EX_USAGE)
}

func (s *ClientSuite) TestBadSubcommandArgs(c *check.C) {
	exited := handler.RunCommand("arvados-client", []string{"get"}, bytes.NewReader(nil), ioutil.Discard, ioutil.Discard)
	c.Check(exited, check.Equals, cmd.EX_USAGE)
}

func (s *ClientSuite) TestVersion(c *check.C) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exited := handler.RunCommand("arvados-client", []string{"version"}, bytes.NewReader(nil), stdout, stderr)
	c.Check(exited, check.Equals, 0)
	c.Check(stdout.String(), check.Matches, `arvados-client dev \(go[0-9\.]+\)\n`)
	c.Check(stderr.String(), check.Equals, "")
}
