// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package githttpd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GitHandlerSuite{})

type GitHandlerSuite struct {
	cluster *arvados.Cluster
}

func (s *GitHandlerSuite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.Equals, nil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.Equals, nil)

	s.cluster.Services.GitHTTP.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Host: "localhost:80"}: {}}
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
	h := newGitHandler(context.Background(), s.cluster)
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
	h := newGitHandler(context.Background(), s.cluster)
	h.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusInternalServerError)
	c.Check(resp.Body.String(), check.Equals, "")
}
