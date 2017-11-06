// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&AuthHandlerSuite{})

type AuthHandlerSuite struct{}

func (s *AuthHandlerSuite) TestCORS(c *check.C) {
	h := &authHandler{}

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
