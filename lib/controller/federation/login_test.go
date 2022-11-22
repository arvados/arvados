// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"net/url"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
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

func (s *LoginSuite) TestLogout(c *check.C) {
	otherOrigin := arvados.URL{Scheme: "https", Host: "app.example.com", Path: "/"}
	otherURL := "https://app.example.com/foo"
	s.cluster.Services.Workbench1.ExternalURL = arvados.URL{Scheme: "https", Host: "workbench1.example.com"}
	s.cluster.Services.Workbench2.ExternalURL = arvados.URL{Scheme: "https", Host: "workbench2.example.com"}
	s.cluster.Login.TrustedClients = map[arvados.URL]struct{}{otherOrigin: {}}
	s.addHTTPRemote(c, "zhome", &arvadostest.APIStub{})
	s.cluster.Login.LoginCluster = "zhome"
	// s.fed is already set by SetUpTest, but we need to
	// reinitialize with the above config changes.
	s.fed = New(s.cluster, nil)

	for _, trial := range []struct {
		token    string
		returnTo string
		target   string
	}{
		{token: "", returnTo: "", target: s.cluster.Services.Workbench2.ExternalURL.String()},
		{token: "", returnTo: otherURL, target: otherURL},
		{token: "zzzzzzzzzzzzzzzzzzzzz", returnTo: otherURL, target: otherURL},
		{token: "v2/zzzzz-aaaaa-aaaaaaaaaaaaaaa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", returnTo: otherURL, target: otherURL},
		{token: "v2/zhome-aaaaa-aaaaaaaaaaaaaaa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", returnTo: otherURL, target: "http://" + s.cluster.RemoteClusters["zhome"].Host + "/logout?" + url.Values{"return_to": {otherURL}}.Encode()},
	} {
		c.Logf("trial %#v", trial)
		ctx := s.ctx
		if trial.token != "" {
			ctx = auth.NewContext(ctx, &auth.Credentials{Tokens: []string{trial.token}})
		}
		resp, err := s.fed.Logout(ctx, arvados.LogoutOptions{ReturnTo: trial.returnTo})
		c.Assert(err, check.IsNil)
		c.Logf("  RedirectLocation %q", resp.RedirectLocation)
		target, err := url.Parse(resp.RedirectLocation)
		c.Check(err, check.IsNil)
		c.Check(target.String(), check.Equals, trial.target)
	}
}
