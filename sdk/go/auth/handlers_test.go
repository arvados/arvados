// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&HandlersSuite{})

type HandlersSuite struct {
	served         int
	gotCredentials *Credentials
}

func (s *HandlersSuite) SetUpTest(c *check.C) {
	s.served = 0
	s.gotCredentials = nil
}

func (s *HandlersSuite) TestLoadToken(c *check.C) {
	handler := LoadToken(s)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/foo/bar?api_token=xyzzy", nil))
	c.Check(s.gotCredentials.Tokens, check.DeepEquals, []string{"xyzzy"})
}

// Ignore leading and trailing spaces, newlines, etc. in case a user
// has added them inadvertently during copy/paste.
func (s *HandlersSuite) TestTrimSpaceInQuery(c *check.C) {
	handler := LoadToken(s)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/foo/bar?api_token=%20xyzzy%0a", nil))
	c.Check(s.gotCredentials.Tokens, check.DeepEquals, []string{"xyzzy"})
}
func (s *HandlersSuite) TestTrimSpaceInPostForm(c *check.C) {
	handler := LoadToken(s)
	req := httptest.NewRequest("POST", "/foo/bar", strings.NewReader(url.Values{"api_token": []string{"\nxyzzy\n"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	c.Check(s.gotCredentials.Tokens, check.DeepEquals, []string{"xyzzy"})
}
func (s *HandlersSuite) TestTrimSpaceInCookie(c *check.C) {
	handler := LoadToken(s)
	req := httptest.NewRequest("GET", "/foo/bar", nil)
	req.AddCookie(&http.Cookie{Name: "arvados_api_token", Value: EncodeTokenCookie([]byte("\vxyzzy\n"))})
	handler.ServeHTTP(httptest.NewRecorder(), req)
	c.Check(s.gotCredentials.Tokens, check.DeepEquals, []string{"xyzzy"})
}
func (s *HandlersSuite) TestTrimSpaceInBasicAuth(c *check.C) {
	handler := LoadToken(s)
	req := httptest.NewRequest("GET", "/foo/bar", nil)
	req.SetBasicAuth("username", "\txyzzy\n")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	c.Check(s.gotCredentials.Tokens, check.DeepEquals, []string{"xyzzy"})
}

func (s *HandlersSuite) TestRequireLiteralTokenEmpty(c *check.C) {
	handler := RequireLiteralToken("", s)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/foo/bar?api_token=abcdef", nil))
	c.Check(s.served, check.Equals, 1)
	c.Check(w.Code, check.Equals, http.StatusOK)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/foo/bar", nil))
	c.Check(s.served, check.Equals, 2)
	c.Check(w.Code, check.Equals, http.StatusOK)
}

func (s *HandlersSuite) TestRequireLiteralToken(c *check.C) {
	handler := RequireLiteralToken("xyzzy", s)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/foo/bar?api_token=abcdef", nil))
	c.Check(s.served, check.Equals, 0)
	c.Check(w.Code, check.Equals, http.StatusForbidden)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/foo/bar", nil))
	c.Check(s.served, check.Equals, 0)
	c.Check(w.Code, check.Equals, http.StatusUnauthorized)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/foo/bar?api_token=xyzzy", nil))
	c.Check(s.served, check.Equals, 1)
	c.Check(w.Code, check.Equals, http.StatusOK)
	c.Assert(s.gotCredentials, check.NotNil)
	c.Assert(s.gotCredentials.Tokens, check.HasLen, 1)
	c.Check(s.gotCredentials.Tokens[0], check.Equals, "xyzzy")
}

func (s *HandlersSuite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.served++
	s.gotCredentials = CredentialsFromRequest(r)
	s.gotCredentials.LoadTokensFromHTTPRequestBody(r)
}
