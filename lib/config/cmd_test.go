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
	code := DumpCommand.RunCommand("arvados config-dump", []string{"-badarg"}, bytes.NewBuffer(nil), bytes.NewBuffer(nil), &stderr)
	c.Check(code, check.Equals, 2)
	c.Check(stderr.String(), check.Matches, `(?ms)usage: .*`)
}

func (s *CommandSuite) TestEmptyInput(c *check.C) {
	var stdout, stderr bytes.Buffer
	code := DumpCommand.RunCommand("arvados config-dump", nil, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(code, check.Equals, 1)
	c.Check(stderr.String(), check.Matches, `config does not define any clusters\n`)
}

func (s *CommandSuite) TestCheckNoDeprecatedKeys(c *check.C) {
	var stdout, stderr bytes.Buffer
	in := `
Clusters:
 z1234:
  API:
    MaxItemsPerResponse: 1234
`
	code := CheckCommand.RunCommand("arvados config-check", nil, bytes.NewBufferString(in), &stdout, &stderr)
	c.Check(code, check.Equals, 0)
	c.Check(stdout.String(), check.Equals, "")
	c.Check(stderr.String(), check.Equals, "")
}

func (s *CommandSuite) TestCheckDeprecatedKeys(c *check.C) {
	var stdout, stderr bytes.Buffer
	in := `
Clusters:
 z1234:
  RequestLimits:
    MaxItemsPerResponse: 1234
`
	code := CheckCommand.RunCommand("arvados config-check", nil, bytes.NewBufferString(in), &stdout, &stderr)
	c.Check(code, check.Equals, 1)
	c.Check(stdout.String(), check.Matches, `(?ms).*API:\n\- +.*MaxItemsPerResponse: 1000\n\+ +MaxItemsPerResponse: 1234\n.*`)
}

func (s *CommandSuite) TestCheckUnknownKey(c *check.C) {
	var stdout, stderr bytes.Buffer
	in := `
Clusters:
 z1234:
  Bogus1: foo
  BogusSection:
    Bogus2: foo
  API:
    Bogus3:
     Bogus4: true
  PostgreSQL:
    ConnectionPool:
      {Bogus5: true}
`
	code := CheckCommand.RunCommand("arvados config-check", nil, bytes.NewBufferString(in), &stdout, &stderr)
	c.Log(stderr.String())
	c.Check(code, check.Equals, 1)
	c.Check(stderr.String(), check.Matches, `(?ms).*deprecated or unknown config entry: Clusters.z1234.Bogus1\n.*`)
	c.Check(stderr.String(), check.Matches, `(?ms).*deprecated or unknown config entry: Clusters.z1234.BogusSection\n.*`)
	c.Check(stderr.String(), check.Matches, `(?ms).*deprecated or unknown config entry: Clusters.z1234.API.Bogus3\n.*`)
	c.Check(stderr.String(), check.Matches, `(?ms).*unexpected object in config entry: Clusters.z1234.PostgreSQL.ConnectionPool\n.*`)
}

func (s *CommandSuite) TestDumpUnknownKey(c *check.C) {
	var stdout, stderr bytes.Buffer
	in := `
Clusters:
 z1234:
  UnknownKey: foobar
  ManagementToken: secret
`
	code := DumpCommand.RunCommand("arvados config-dump", nil, bytes.NewBufferString(in), &stdout, &stderr)
	c.Check(code, check.Equals, 0)
	c.Check(stderr.String(), check.Matches, `(?ms).*deprecated or unknown config entry: Clusters.z1234.UnknownKey.*`)
	c.Check(stdout.String(), check.Matches, `(?ms)Clusters:\n  z1234:\n.*`)
	c.Check(stdout.String(), check.Matches, `(?ms).*\n *ManagementToken: secret\n.*`)
	c.Check(stdout.String(), check.Not(check.Matches), `(?ms).*UnknownKey.*`)
}
