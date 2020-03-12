// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"io/ioutil"
	"os"
	"strings"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&PamSuite{})

type PamSuite struct {
	cluster  *arvados.Cluster
	ctrl     *pamLoginController
	railsSpy *arvadostest.Proxy
}

func (s *PamSuite) SetUpSuite(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.cluster.Login.PAM = true
	s.cluster.Login.PAMDefaultEmailDomain = "example.com"
	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	s.ctrl = &pamLoginController{
		Cluster:    s.cluster,
		RailsProxy: rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider),
	}
}

func (s *PamSuite) TestLoginFailure(c *check.C) {
	resp, err := s.ctrl.Login(context.Background(), arvados.LoginOptions{
		Username: "bogususername",
		Password: "boguspassword",
		ReturnTo: "https://example.com/foo",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.Token, check.Equals, "")
	c.Check(resp.Message, check.Equals, "Authentication failure")
	c.Check(resp.HTML.String(), check.Equals, "")
}

// This test only runs if the ARVADOS_TEST_PAM_CREDENTIALS_FILE env
// var is set. The credentials file should contain a valid username
// and password, separated by \n.
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

	resp, err := s.ctrl.Login(context.Background(), arvados.LoginOptions{
		Username: u,
		Password: p,
		ReturnTo: "https://example.com/foo",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.Token, check.Matches, `v2/zzzzz-gj3su-.*/.*`)
	c.Check(resp.HTML.String(), check.Equals, "")

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.Email, check.Equals, u+"@"+s.cluster.Login.PAMDefaultEmailDomain)
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string(nil))
}
