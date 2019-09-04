// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"

	"git.curoverse.com/arvados.git/lib/config"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GitHandlerSuite{})

type GitHandlerSuite struct {
	cluster *arvados.Cluster
}

func (s *GitHandlerSuite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, nil).Load()
	c.Assert(err, check.Equals, nil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.Equals, nil)

	s.cluster.Services.GitHTTP.InternalURLs = map[arvados.URL]arvados.ServiceInstance{arvados.URL{Host: "localhost:80"}: arvados.ServiceInstance{}}
	s.cluster.Git.GitoliteHome = "/test/ghh"
	s.cluster.Git.Repositories = "/"
}

func (s *GitHandlerSuite) TestEnvVars(c *check.C) {
	u, err := url.Parse("git.zzzzz.arvadosapi.com/test")
	c.Check(err, check.Equals, nil)
	resp := httptest.NewRecorder()
	req := &http.Request{
		Method:     "GET",
		URL:        u,
		RemoteAddr: "[::1]:12345",
	}
	h := newGitHandler(s.cluster)
	h.(*gitHandler).Path = "/bin/sh"
	h.(*gitHandler).Args = []string{"-c", "printf 'Content-Type: text/plain\r\n\r\n'; env"}

	h.ServeHTTP(resp, req)

	c.Check(resp.Code, check.Equals, http.StatusOK)
	body := resp.Body.String()
	c.Check(body, check.Matches, `(?ms).*^PATH=.*:/test/ghh/bin$.*`)
	c.Check(body, check.Matches, `(?ms).*^GITOLITE_HTTP_HOME=/test/ghh$.*`)
	c.Check(body, check.Matches, `(?ms).*^GL_BYPASS_ACCESS_CHECKS=1$.*`)
	c.Check(body, check.Matches, `(?ms).*^REMOTE_HOST=::1$.*`)
	c.Check(body, check.Matches, `(?ms).*^REMOTE_PORT=12345$.*`)
	c.Check(body, check.Matches, `(?ms).*^SERVER_ADDR=`+regexp.QuoteMeta("localhost:80")+`$.*`)
}

func (s *GitHandlerSuite) TestCGIErrorOnSplitHostPortError(c *check.C) {
	u, err := url.Parse("git.zzzzz.arvadosapi.com/test")
	c.Check(err, check.Equals, nil)
	resp := httptest.NewRecorder()
	req := &http.Request{
		Method:     "GET",
		URL:        u,
		RemoteAddr: "test.bad.address.missing.port",
	}
	h := newGitHandler(s.cluster)
	h.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusInternalServerError)
	c.Check(resp.Body.String(), check.Equals, "")
}
