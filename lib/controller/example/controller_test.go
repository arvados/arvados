// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package example

import (
	"context"
	"testing"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&ExampleSuite{})

type ExampleSuite struct {
	ctrl     *Controller
	ctx      context.Context
	rollback func()
}

func (s *ExampleSuite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.ctx, s.rollback = arvadostest.TransactionContext(c, arvadostest.DB(c, cluster))
	s.ctrl = New(cluster)
}

func (s *ExampleSuite) TearDownTest(c *check.C) {
	if s.rollback != nil {
		s.rollback()
		s.rollback = nil
	}
}

func (s *ExampleSuite) TestCount(c *check.C) {
	resp, err := s.ctrl.ExampleCount(s.ctx, arvados.ExampleCountOptions{})
	c.Check(err, check.IsNil)
	c.Check(resp.Count, check.Equals, 29)
}

func (s *ExampleSuite) TestGet(c *check.C) {
	resp, err := s.ctrl.ExampleGet(s.ctx, arvados.GetOptions{UUID: "alice"})
	c.Check(err, check.IsNil)
	c.Check(resp.UUID, check.Equals, "alice")
	c.Check(resp.HairStyle, check.Equals, "bob")
}
