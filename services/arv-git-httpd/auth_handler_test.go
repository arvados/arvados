// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&AuthHandlerSuite{})

type AuthHandlerSuite struct {
	cluster *arvados.Cluster
}

func (s *AuthHandlerSuite) SetUpTest(c *check.C) {
	arvadostest.ResetEnv()
	repoRoot, err := filepath.Abs("../api/tmp/git/test")
	c.Assert(err, check.IsNil)

	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.Equals, nil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.Equals, nil)

	s.cluster.Services.GitHTTP.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Host: "localhost:0"}: {}}
	s.cluster.TLS.Insecure = true
	s.cluster.Git.GitCommand = "/usr/bin/git"
	s.cluster.Git.Repositories = repoRoot
}

func (s *AuthHandlerSuite) TestPermission(c *check.C) {
	h := &authHandler{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v", r.URL)
		io.WriteString(w, r.URL.Path)
	}), cluster: s.cluster}
	baseURL, err := url.Parse("http://git.example/")
	c.Assert(err, check.IsNil)
	for _, trial := range []struct {
		label   string
		token   string
		pathIn  string
		pathOut string
		status  int
	}{
		{
			label:   "read repo by name",
			token:   arvadostest.ActiveToken,
			pathIn:  arvadostest.Repository2Name + ".git/git-upload-pack",
			pathOut: arvadostest.Repository2UUID + ".git/git-upload-pack",
		},
		{
			label:   "read repo by uuid",
			token:   arvadostest.ActiveToken,
			pathIn:  arvadostest.Repository2UUID + ".git/git-upload-pack",
			pathOut: arvadostest.Repository2UUID + ".git/git-upload-pack",
		},
		{
			label:   "write repo by name",
			token:   arvadostest.ActiveToken,
			pathIn:  arvadostest.Repository2Name + ".git/git-receive-pack",
			pathOut: arvadostest.Repository2UUID + ".git/git-receive-pack",
		},
		{
			label:   "write repo by uuid",
			token:   arvadostest.ActiveToken,
			pathIn:  arvadostest.Repository2UUID + ".git/git-receive-pack",
			pathOut: arvadostest.Repository2UUID + ".git/git-receive-pack",
		},
		{
			label:  "uuid not found",
			token:  arvadostest.ActiveToken,
			pathIn: strings.Replace(arvadostest.Repository2UUID, "6", "z", -1) + ".git/git-upload-pack",
			status: http.StatusNotFound,
		},
		{
			label:  "name not found",
			token:  arvadostest.ActiveToken,
			pathIn: "nonexistent-bogus.git/git-upload-pack",
			status: http.StatusNotFound,
		},
		{
			label:   "read read-only repo",
			token:   arvadostest.SpectatorToken,
			pathIn:  arvadostest.FooRepoName + ".git/git-upload-pack",
			pathOut: arvadostest.FooRepoUUID + "/.git/git-upload-pack",
		},
		{
			label:  "write read-only repo",
			token:  arvadostest.SpectatorToken,
			pathIn: arvadostest.FooRepoName + ".git/git-receive-pack",
			status: http.StatusForbidden,
		},
	} {
		c.Logf("trial label: %q", trial.label)
		u, err := baseURL.Parse(trial.pathIn)
		c.Assert(err, check.IsNil)
		resp := httptest.NewRecorder()
		req := &http.Request{
			Method: "POST",
			URL:    u,
			Header: http.Header{
				"Authorization": {"Bearer " + trial.token}}}
		h.ServeHTTP(resp, req)
		if trial.status == 0 {
			trial.status = http.StatusOK
		}
		c.Check(resp.Code, check.Equals, trial.status)
		if trial.status < 400 {
			if trial.pathOut != "" && !strings.HasPrefix(trial.pathOut, "/") {
				trial.pathOut = "/" + trial.pathOut
			}
			c.Check(resp.Body.String(), check.Equals, trial.pathOut)
		}
	}
}

func (s *AuthHandlerSuite) TestCORS(c *check.C) {
	h := &authHandler{cluster: s.cluster}

	// CORS preflight
	resp := httptest.NewRecorder()
	req := &http.Request{
		Method: "OPTIONS",
		Header: http.Header{
			"Origin":                        {"*"},
			"Access-Control-Request-Method": {"GET"},
		},
	}
	h.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Header().Get("Access-Control-Allow-Methods"), check.Equals, "GET, POST")
	c.Check(resp.Header().Get("Access-Control-Allow-Headers"), check.Equals, "Authorization, Content-Type")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")
	c.Check(resp.Body.String(), check.Equals, "")

	// CORS actual request. Bogus token and path ensure
	// authHandler responds 4xx without calling our wrapped (nil)
	// handler.
	u, err := url.Parse("git.zzzzz.arvadosapi.com/test")
	c.Assert(err, check.Equals, nil)
	resp = httptest.NewRecorder()
	req = &http.Request{
		Method: "GET",
		URL:    u,
		Header: http.Header{
			"Origin":        {"*"},
			"Authorization": {"OAuth2 foobar"},
		},
	}
	h.ServeHTTP(resp, req)
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")
}
