// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"net/url"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&LoginSuite{})

type LoginSuite struct {
	FederationSuite
}

func (s *LoginSuite) TestDeferToLoginCluster(c *check.C) {
	s.addHTTPRemote(c, "zhome", &arvadostest.APIStub{})
	s.cluster.Login.LoginCluster = "zhome"

	returnTo := "https://app.example.com/foo?bar"
	for _, remote := range []string{"", "ccccc"} {
		resp, err := s.fed.Login(context.Background(), arvados.LoginOptions{Remote: remote, ReturnTo: returnTo})
		c.Check(err, check.IsNil)
		c.Logf("remote %q -- RedirectLocation %q", remote, resp.RedirectLocation)
		target, err := url.Parse(resp.RedirectLocation)
		c.Check(err, check.IsNil)
		c.Check(target.Host, check.Equals, s.cluster.RemoteClusters["zhome"].Host)
		c.Check(target.Scheme, check.Equals, "http")
		c.Check(target.Query().Get("return_to"), check.Equals, returnTo)
		c.Check(target.Query().Get("remote"), check.Equals, remote)
		_, remotePresent := target.Query()["remote"]
		c.Check(remotePresent, check.Equals, remote != "")
	}
}
