// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package forecaster

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

var _ = check.Suite(&ForecasterSuite{})

type ForecasterSuite struct {
	ctrl     *Controller
	ctx      context.Context
	stub     *arvadostest.APIStub
	rollback func()
}

func (s *ForecasterSuite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.ctx, s.rollback = arvadostest.TransactionContext(c, arvadostest.DB(c, cluster))
	s.stub = &arvadostest.APIStub{}
	s.ctrl = New(cluster, s.stub)
}

func (s *ForecasterSuite) TearDownTest(c *check.C) {
	if s.rollback != nil {
		s.rollback()
		s.rollback = nil
	}
}

func (s *ForecasterSuite) TestGet(c *check.C) {
	resp, err := s.ctrl.CheckpointsGet(s.ctx, arvados.GetOptions{UUID: "alice"})
	c.Check(err, check.IsNil)
	c.Check(resp.UUID, check.Equals, "alice")
