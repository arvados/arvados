// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&LogoutSuite{})
var emptyURL = &url.URL{}

type LogoutStub struct {
	arvadostest.APIStub
	redirectLocation *url.URL
}

func (as *LogoutStub) CheckCalls(c *check.C, returnURL *url.URL) bool {
	actual := as.APIStub.Calls(as.APIStub.Logout)
	allOK := c.Check(actual, check.Not(check.HasLen), 0,
		check.Commentf("Logout stub never called"))
	expected := returnURL.String()
	for _, call := range actual {
		opts, ok := call.Options.(arvados.LogoutOptions)
		allOK = c.Check(ok, check.Equals, true,
			check.Commentf("call options were not LogoutOptions")) &&
			c.Check(opts.ReturnTo, check.Equals, expected) &&
			allOK
	}
	return allOK
}

func (as *LogoutStub) Logout(ctx context.Context, options arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	as.APIStub.Logout(ctx, options)
	loc := as.redirectLocation.String()
	if loc == "" {
		loc = options.ReturnTo
	}
	return arvados.LogoutResponse{
		RedirectLocation: loc,
	}, as.Error
}

type LogoutSuite struct {
	FederationSuite
}

func (s *LogoutSuite) badReturnURL(path string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "example.net",
		Path:   path,
	}
}

func (s *LogoutSuite) goodReturnURL(path string) *url.URL {
	u, _ := url.Parse(s.cluster.Services.Workbench2.ExternalURL.String())
	u.Path = path
	return u
}

func (s *LogoutSuite) setupFederation(loginCluster string) {
	if loginCluster == "" {
		s.cluster.Login.Test.Enable = true
	} else {
		s.cluster.Login.LoginCluster = loginCluster
	}
	dbconn := ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}
	s.fed = New(s.ctx, s.cluster, nil, dbconn.GetDB)
}

func (s *LogoutSuite) setupStub(c *check.C, id string, stubURL *url.URL, stubErr error) *LogoutStub {
	loc, err := url.Parse(stubURL.String())
	c.Check(err, check.IsNil)
	stub := LogoutStub{redirectLocation: loc}
	stub.Error = stubErr
	if id == s.cluster.ClusterID {
		s.fed.local = &stub
	} else {
		s.addDirectRemote(c, id, &stub)
	}
	return &stub
}

func (s *LogoutSuite) v2Token(clusterID string) string {
	return fmt.Sprintf("v2/%s-gj3su-12345abcde67890/abcdefghijklmnopqrstuvwxy", clusterID)
}

func (s *LogoutSuite) TestLocalLogoutOK(c *check.C) {
	s.setupFederation("")
	resp, err := s.fed.Logout(s.ctx, arvados.LogoutOptions{})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, s.cluster.Services.Workbench2.ExternalURL.String())
}

func (s *LogoutSuite) TestLocalLogoutRedirect(c *check.C) {
	s.setupFederation("")
	expURL := s.cluster.Services.Workbench1.ExternalURL
	opts := arvados.LogoutOptions{ReturnTo: expURL.String()}
	resp, err := s.fed.Logout(s.ctx, opts)
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, expURL.String())
}

func (s *LogoutSuite) TestLocalLogoutBadRequestError(c *check.C) {
	s.setupFederation("")
	returnTo := s.badReturnURL("TestLocalLogoutBadRequestError")
	opts := arvados.LogoutOptions{ReturnTo: returnTo.String()}
	_, err := s.fed.Logout(s.ctx, opts)
	c.Check(err, check.NotNil)
}

func (s *LogoutSuite) TestRemoteLogoutRedirect(c *check.C) {
	s.setupFederation("zhome")
	redirect := url.URL{Scheme: "https", Host: "example.com"}
	loginStub := s.setupStub(c, "zhome", &redirect, nil)
	returnTo := s.goodReturnURL("TestRemoteLogoutRedirect")
	resp, err := s.fed.Logout(s.ctx, arvados.LogoutOptions{ReturnTo: returnTo.String()})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, redirect.String())
	loginStub.CheckCalls(c, returnTo)
}

func (s *LogoutSuite) TestRemoteLogoutError(c *check.C) {
	s.setupFederation("zhome")
	expErr := errors.New("TestRemoteLogoutError expErr")
	loginStub := s.setupStub(c, "zhome", emptyURL, expErr)
	returnTo := s.goodReturnURL("TestRemoteLogoutError")
	_, err := s.fed.Logout(s.ctx, arvados.LogoutOptions{ReturnTo: returnTo.String()})
	c.Check(err, check.Equals, expErr)
	loginStub.CheckCalls(c, returnTo)
}

func (s *LogoutSuite) TestRemoteLogoutLocalRedirect(c *check.C) {
	s.setupFederation("zhome")
	loginStub := s.setupStub(c, "zhome", emptyURL, nil)
	redirect := url.URL{Scheme: "https", Host: "example.com"}
	localStub := s.setupStub(c, "aaaaa", &redirect, nil)
	resp, err := s.fed.Logout(s.ctx, arvados.LogoutOptions{})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, redirect.String())
	// emptyURL to match the empty LogoutOptions
	loginStub.CheckCalls(c, emptyURL)
	localStub.CheckCalls(c, emptyURL)
}

func (s *LogoutSuite) TestRemoteLogoutLocalError(c *check.C) {
	s.setupFederation("zhome")
	expErr := errors.New("TestRemoteLogoutLocalError expErr")
	loginStub := s.setupStub(c, "zhome", emptyURL, nil)
	localStub := s.setupStub(c, "aaaaa", emptyURL, expErr)
	_, err := s.fed.Logout(s.ctx, arvados.LogoutOptions{})
	c.Check(err, check.Equals, expErr)
	loginStub.CheckCalls(c, emptyURL)
	localStub.CheckCalls(c, emptyURL)
}

func (s *LogoutSuite) TestV2TokenRedirect(c *check.C) {
	s.setupFederation("")
	redirect := url.URL{Scheme: "https", Host: "example.com"}
	returnTo := s.goodReturnURL("TestV2TokenRedirect")
	localErr := errors.New("TestV2TokenRedirect error")
	tokenStub := s.setupStub(c, "zzzzz", &redirect, nil)
	s.setupStub(c, "aaaaa", emptyURL, localErr)
	tokens := []string{s.v2Token("zzzzz")}
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: tokens})
	resp, err := s.fed.Logout(ctx, arvados.LogoutOptions{ReturnTo: returnTo.String()})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, redirect.String())
	tokenStub.CheckCalls(c, returnTo)
}

func (s *LogoutSuite) TestV2TokenError(c *check.C) {
	s.setupFederation("")
	returnTo := s.goodReturnURL("TestV2TokenError")
	tokenErr := errors.New("TestV2TokenError error")
	tokenStub := s.setupStub(c, "zzzzz", emptyURL, tokenErr)
	s.setupStub(c, "aaaaa", emptyURL, nil)
	tokens := []string{s.v2Token("zzzzz")}
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: tokens})
	_, err := s.fed.Logout(ctx, arvados.LogoutOptions{ReturnTo: returnTo.String()})
	c.Check(err, check.Equals, tokenErr)
	tokenStub.CheckCalls(c, returnTo)
}

func (s *LogoutSuite) TestV2TokenLocalRedirect(c *check.C) {
	s.setupFederation("")
	redirect := url.URL{Scheme: "https", Host: "example.com"}
	tokenStub := s.setupStub(c, "zzzzz", emptyURL, nil)
	localStub := s.setupStub(c, "aaaaa", &redirect, nil)
	tokens := []string{s.v2Token("zzzzz")}
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: tokens})
	resp, err := s.fed.Logout(ctx, arvados.LogoutOptions{})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, redirect.String())
	tokenStub.CheckCalls(c, emptyURL)
	localStub.CheckCalls(c, emptyURL)
}

func (s *LogoutSuite) TestV2TokenLocalError(c *check.C) {
	s.setupFederation("")
	tokenErr := errors.New("TestV2TokenLocalError error")
	tokenStub := s.setupStub(c, "zzzzz", emptyURL, nil)
	localStub := s.setupStub(c, "aaaaa", emptyURL, tokenErr)
	tokens := []string{s.v2Token("zzzzz")}
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: tokens})
	_, err := s.fed.Logout(ctx, arvados.LogoutOptions{})
	c.Check(err, check.Equals, tokenErr)
	tokenStub.CheckCalls(c, emptyURL)
	localStub.CheckCalls(c, emptyURL)
}

func (s *LogoutSuite) TestV2LocalTokenRedirect(c *check.C) {
	s.setupFederation("")
	redirect := url.URL{Scheme: "https", Host: "example.com"}
	returnTo := s.goodReturnURL("TestV2LocalTokenRedirect")
	localStub := s.setupStub(c, "aaaaa", &redirect, nil)
	tokens := []string{s.v2Token("aaaaa")}
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: tokens})
	resp, err := s.fed.Logout(ctx, arvados.LogoutOptions{ReturnTo: returnTo.String()})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, redirect.String())
	localStub.CheckCalls(c, returnTo)
}

func (s *LogoutSuite) TestV2LocalTokenError(c *check.C) {
	s.setupFederation("")
	returnTo := s.goodReturnURL("TestV2LocalTokenError")
	tokenErr := errors.New("TestV2LocalTokenError error")
	localStub := s.setupStub(c, "aaaaa", emptyURL, tokenErr)
	tokens := []string{s.v2Token("aaaaa")}
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: tokens})
	_, err := s.fed.Logout(ctx, arvados.LogoutOptions{ReturnTo: returnTo.String()})
	c.Check(err, check.Equals, tokenErr)
	localStub.CheckCalls(c, returnTo)
}
