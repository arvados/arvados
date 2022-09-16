// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&PamSuite{})

type PamSuite struct {
	cluster  *arvados.Cluster
	ctrl     *pamLoginController
	railsSpy *arvadostest.Proxy
	db       *sqlx.DB
	ctx      context.Context
	rollback func() error
}

func (s *PamSuite) SetUpSuite(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.cluster.Login.PAM.Enable = true
	s.cluster.Login.PAM.DefaultEmailDomain = "example.com"
	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	s.ctrl = &pamLoginController{
		Cluster: s.cluster,
		Parent:  &Conn{railsProxy: rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)},
	}
	s.db = arvadostest.DB(c, s.cluster)
}

func (s *PamSuite) SetUpTest(c *check.C) {
	tx, err := s.db.Beginx()
	c.Assert(err, check.IsNil)
	s.ctx = ctrlctx.NewWithTransaction(context.Background(), tx)
	s.rollback = tx.Rollback
}

func (s *PamSuite) TearDownTest(c *check.C) {
	if s.rollback != nil {
		s.rollback()
	}
}

func (s *PamSuite) TestLoginFailure(c *check.C) {
	resp, err := s.ctrl.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: "bogususername",
		Password: "boguspassword",
	})
	c.Check(err, check.ErrorMatches, `PAM: Authentication failure \(with username "bogususername" and password\)`)
	hs, ok := err.(interface{ HTTPStatus() int })
	if c.Check(ok, check.Equals, true) {
		c.Check(hs.HTTPStatus(), check.Equals, http.StatusUnauthorized)
	}
	c.Check(resp.APIToken, check.Equals, "")
}

// This test only runs if the ARVADOS_TEST_PAM_CREDENTIALS_FILE env
// var is set. The credentials file should contain a valid username
// and password, separated by \n.
//
// Depending on the host config, this test succeeds only if the test
// credentials are for the same account being used to run tests.
func (s *PamSuite) TestLoginSuccess(c *check.C) {
	testCredsFile := os.Getenv("ARVADOS_TEST_PAM_CREDENTIALS_FILE")
	if testCredsFile == "" {
		c.Skip("no test credentials file given in ARVADOS_TEST_PAM_CREDENTIALS_FILE")
		return
	}
	buf, err := ioutil.ReadFile(testCredsFile)
	c.Assert(err, check.IsNil)
	lines := strings.Split(string(buf), "\n")
	c.Assert(len(lines), check.Equals, 2, check.Commentf("credentials file %s should contain \"username\\npassword\"", testCredsFile))
	u, p := lines[0], lines[1]

	resp, err := s.ctrl.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: u,
		Password: p,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.APIToken, check.Not(check.Equals), "")
	c.Check(resp.UUID, check.Matches, `zzzzz-gj3su-.*`)
	c.Check(resp.Scopes, check.DeepEquals, []string{"all"})

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.Email, check.Equals, u+"@"+s.cluster.Login.PAM.DefaultEmailDomain)
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string(nil))
}
