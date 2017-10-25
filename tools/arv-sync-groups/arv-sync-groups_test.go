// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"os"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestParseFlagsWithPath(c *C) {
	cfg := ConfigParams{}
	os.Args = []string{"cmd", "-path", "/tmp/somefile.csv", "-verbose"}
	err := ParseFlags(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Path, Equals, "/tmp/somefile.csv")
	c.Assert(cfg.Verbose, Equals, true)
}

func (s *TestSuite) TestParseFlagsWithoutPath(c *C) {
	os.Args = []string{"cmd", "-verbose"}
	err := ParseFlags(&ConfigParams{})
	c.Assert(err, NotNil)
}

func (s *TestSuite) TestGetUserID(c *C) {
	u := arvados.User{
		Email:    "testuser@example.com",
		Username: "Testuser",
	}
	email, err := GetUserID(u, "email")
	c.Assert(err, IsNil)
	c.Assert(email, Equals, "testuser@example.com")
	_, err = GetUserID(u, "bogus")
	c.Assert(err, NotNil)
}

func (s *TestSuite) TestGetConfig(c *C) {
	os.Args = []string{"cmd", "-path", "/tmp/somefile.csv"}
	cfg, err := GetConfig()
	c.Assert(err, IsNil)
	c.Assert(cfg.SysUserUUID, NotNil)
	c.Assert(cfg.Client, NotNil)
	c.Assert(cfg.ParentGroupUUID, NotNil)
	c.Assert(cfg.ParentGroupName, Equals, "Externally synchronized groups")
}
