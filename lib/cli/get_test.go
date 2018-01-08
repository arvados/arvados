// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"regexp"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&GetSuite{})

type GetSuite struct{}

func (s *GetSuite) TestGetCollectionJSON(c *check.C) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	exited := Get.RunCommand("arvados-client get", []string{arvadostest.FooCollection}, bytes.NewReader(nil), stdout, stderr)
	c.Check(stdout.String(), check.Matches, `(?ms){.*"uuid": "`+arvadostest.FooCollection+`".*}\n`)
	c.Check(stdout.String(), check.Matches, `(?ms){.*"portable_data_hash": "`+regexp.QuoteMeta(arvadostest.FooCollectionPDH)+`".*}\n`)
	c.Check(stderr.String(), check.Equals, "")
	c.Check(exited, check.Equals, 0)
}
