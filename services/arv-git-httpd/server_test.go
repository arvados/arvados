// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GitSuite{})

const (
	spectatorToken = "zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu"
	activeToken    = "3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"
	anonymousToken = "4kg6k6lzmp9kj4cpkcoxie964cmvjahbt4fod9zru44k4jqdmi"
	expiredToken   = "2ym314ysp27sk7h943q6vtc378srb06se3pq6ghurylyf3pdmx"
)

type GitSuite struct {
	IntegrationSuite
}

func (s *GitSuite) TestPathVariants(c *check.C) {
	s.makeArvadosRepo(c)
	for _, repo := range []string{"active/foo.git", "active/foo/.git", "arvados.git", "arvados/.git"} {
		err := s.RunGit(c, spectatorToken, "fetch", repo, "refs/heads/main")
		c.Assert(err, check.Equals, nil)
	}
}

func (s *GitSuite) TestReadonly(c *check.C) {
	err := s.RunGit(c, spectatorToken, "fetch", "active/foo.git", "refs/heads/main")
	c.Assert(err, check.Equals, nil)
	err = s.RunGit(c, spectatorToken, "push", "active/foo.git", "main:newbranchfail")
	c.Assert(err, check.ErrorMatches, `.*HTTP (code = )?403.*`)
	_, err = os.Stat(s.tmpRepoRoot + "/zzzzz-s0uqq-382brsig8rp3666.git/refs/heads/newbranchfail")
	c.Assert(err, check.FitsTypeOf, &os.PathError{})
}

func (s *GitSuite) TestReadwrite(c *check.C) {
	err := s.RunGit(c, activeToken, "fetch", "active/foo.git", "refs/heads/main")
	c.Assert(err, check.Equals, nil)
	err = s.RunGit(c, activeToken, "push", "active/foo.git", "main:newbranch")
	c.Assert(err, check.Equals, nil)
	_, err = os.Stat(s.tmpRepoRoot + "/zzzzz-s0uqq-382brsig8rp3666.git/refs/heads/newbranch")
	c.Assert(err, check.Equals, nil)
}

func (s *GitSuite) TestNonexistent(c *check.C) {
	err := s.RunGit(c, spectatorToken, "fetch", "thisrepodoesnotexist.git", "refs/heads/main")
	c.Assert(err, check.ErrorMatches, `.* not found.*`)
}

func (s *GitSuite) TestMissingGitdirReadableRepository(c *check.C) {
	err := s.RunGit(c, activeToken, "fetch", "active/foo2.git", "refs/heads/main")
	c.Assert(err, check.ErrorMatches, `.* not found.*`)
}

func (s *GitSuite) TestNoPermission(c *check.C) {
	for _, repo := range []string{"active/foo.git", "active/foo/.git"} {
		err := s.RunGit(c, anonymousToken, "fetch", repo, "refs/heads/main")
		c.Assert(err, check.ErrorMatches, `.* not found.*`)
	}
}

func (s *GitSuite) TestExpiredToken(c *check.C) {
	for _, repo := range []string{"active/foo.git", "active/foo/.git"} {
		err := s.RunGit(c, expiredToken, "fetch", repo, "refs/heads/main")
		c.Assert(err, check.ErrorMatches, `.* (500 while accessing|requested URL returned error: 500).*`)
	}
}

func (s *GitSuite) TestInvalidToken(c *check.C) {
	for _, repo := range []string{"active/foo.git", "active/foo/.git"} {
		err := s.RunGit(c, "s3cr3tp@ssw0rd", "fetch", repo, "refs/heads/main")
		c.Assert(err, check.ErrorMatches, `.* requested URL returned error.*`)
	}
}

func (s *GitSuite) TestShortToken(c *check.C) {
	for _, repo := range []string{"active/foo.git", "active/foo/.git"} {
		err := s.RunGit(c, "s3cr3t", "fetch", repo, "refs/heads/main")
		c.Assert(err, check.ErrorMatches, `.* (500 while accessing|requested URL returned error: 500).*`)
	}
}

func (s *GitSuite) TestShortTokenBadReq(c *check.C) {
	for _, repo := range []string{"bogus"} {
		err := s.RunGit(c, "s3cr3t", "fetch", repo, "refs/heads/main")
		c.Assert(err, check.ErrorMatches, `.*not found.*`)
	}
}

// Make a bare arvados repo at {tmpRepoRoot}/arvados.git
func (s *GitSuite) makeArvadosRepo(c *check.C) {
	msg, err := exec.Command("git", "init", "--bare", s.tmpRepoRoot+"/zzzzz-s0uqq-arvadosrepo0123.git").CombinedOutput()
	c.Log(string(msg))
	c.Assert(err, check.Equals, nil)
	msg, err = exec.Command("git", "--git-dir", s.tmpRepoRoot+"/zzzzz-s0uqq-arvadosrepo0123.git", "fetch", "../../.git", "HEAD:main").CombinedOutput()
	c.Log(string(msg))
	c.Assert(err, check.Equals, nil)
}

func (s *GitSuite) TestHealthCheckPing(c *check.C) {
	req, err := http.NewRequest("GET",
		"http://"+s.testServer.Addr+"/_health/ping",
		nil)
	c.Assert(err, check.Equals, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ManagementToken)

	resp := httptest.NewRecorder()
	s.testServer.Handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, 200)
	c.Check(resp.Body.String(), check.Matches, `{"health":"OK"}\n`)
}
