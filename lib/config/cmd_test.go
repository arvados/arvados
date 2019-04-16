// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CommandSuite{})

type CommandSuite struct{}

func (s *CommandSuite) TestBadArg(c *check.C) {
	var stderr bytes.Buffer
	code := DumpCommand.RunCommand("arvados dump-config", []string{"-badarg"}, bytes.NewBuffer(nil), bytes.NewBuffer(nil), &stderr)
	c.Check(code, check.Equals, 2)
	c.Check(stderr.String(), check.Matches, `(?ms)usage: .*`)
}

func (s *CommandSuite) TestEmptyInput(c *check.C) {
	var stdout, stderr bytes.Buffer
	code := DumpCommand.RunCommand("arvados dump-config", nil, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(code, check.Equals, 1)
	c.Check(stderr.String(), check.Matches, `config does not define any clusters\n`)
}

func (s *CommandSuite) TestUnknownKey(c *check.C) {
	var stdout, stderr bytes.Buffer
	in := `
Clusters:
 z1234:
  UnknownKey: foobar
  ManagementToken: secret
`
	code := DumpCommand.RunCommand("arvados dump-config", nil, bytes.NewBufferString(in), &stdout, &stderr)
	c.Check(code, check.Equals, 0)
	c.Check(stdout.String(), check.Matches, `(?ms)Clusters:\n  z1234:\n.*`)
	c.Check(stdout.String(), check.Matches, `(?ms).*\n *ManagementToken: secret\n.*`)
	c.Check(stdout.String(), check.Not(check.Matches), `(?ms).*UnknownKey.*`)
}
