// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"net/url"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

func (s *FederationSuite) TestDeferToLoginCluster(c *check.C) {
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
		c.Check(target.Query().Get("remote"), check.Equals, remote)
		c.Check(target.Query().Get("return_to"), check.Equals, returnTo)
	}
}
