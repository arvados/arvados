// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&TestUserSuite{})

type TestUserSuite struct {
	cluster  *arvados.Cluster
	ctrl     *testLoginController
	railsSpy *arvadostest.Proxy
	db       *sqlx.DB

	// transaction context
	ctx      context.Context
	rollback func() error
}

func (s *TestUserSuite) SetUpSuite(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.cluster.Login.Test.Enable = true
	s.cluster.Login.Test.Users = map[string]arvados.TestUser{
		"valid": {Email: "valid@example.com", Password: "v@l1d"},
	}
	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	s.ctrl = &testLoginController{
		Cluster: s.cluster,
		Parent:  &Conn{railsProxy: rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)},
	}
	s.db = arvadostest.DB(c, s.cluster)
}

func (s *TestUserSuite) SetUpTest(c *check.C) {
	tx, err := s.db.Beginx()
	c.Assert(err, check.IsNil)
	s.ctx = ctrlctx.NewWithTransaction(context.Background(), tx)
	s.rollback = tx.Rollback
}

func (s *TestUserSuite) TearDownTest(c *check.C) {
	if s.rollback != nil {
		s.rollback()
	}
}

func (s *TestUserSuite) TestLogin(c *check.C) {
	for _, trial := range []struct {
		success  bool
		username string
		password string
	}{
		{false, "foo", "bar"},
		{false, "", ""},
		{false, "valid", ""},
		{false, "", "v@l1d"},
		{true, "valid", "v@l1d"},
		{true, "valid@example.com", "v@l1d"},
	} {
		c.Logf("=== %#v", trial)
		resp, err := s.ctrl.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
			Username: trial.username,
			Password: trial.password,
		})
		if trial.success {
			c.Check(err, check.IsNil)
			c.Check(resp.APIToken, check.Not(check.Equals), "")
			c.Check(resp.UUID, check.Matches, `zzzzz-gj3su-.*`)
			c.Check(resp.Scopes, check.DeepEquals, []string{"all"})

			authinfo := getCallbackAuthInfo(c, s.railsSpy)
			c.Check(authinfo.Email, check.Equals, "valid@example.com")
			c.Check(authinfo.AlternateEmails, check.DeepEquals, []string(nil))
		} else {
			c.Check(err, check.ErrorMatches, `authentication failed.*`)
		}
	}
}

func (s *TestUserSuite) TestLoginForm(c *check.C) {
	resp, err := s.ctrl.Login(s.ctx, arvados.LoginOptions{
		ReturnTo: "https://localhost:12345/example",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*<form method="POST".*`)
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*<input id="return_to" type="hidden" name="return_to" value="https://localhost:12345/example">.*`)
}
