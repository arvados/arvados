// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
var _ = check.Suite(&Suite{})

func Test(t *testing.T) {
	check.TestingT(t)
}

type Suite struct{}

const (
	goodToken = "supersecret"
	badToken  = "pwn"
)

func (s *Suite) TestPassFailRefuse(c *check.C) {
	h := &Handler{
		Token:  goodToken,
		Prefix: "/_health/",
		Routes: Routes{
			"success": func() error { return nil },
			"miracle": func() error { return errors.New("unimplemented") },
		},
	}

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/_health/ping", goodToken))
	s.checkHealthy(c, resp)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/_health/success", goodToken))
	s.checkHealthy(c, resp)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/_health/miracle", goodToken))
	s.checkUnhealthy(c, resp)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/_health/miracle", badToken))
	c.Check(resp.Code, check.Equals, http.StatusForbidden)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/_health/miracle", ""))
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/_health/theperthcountyconspiracy", ""))
	c.Check(resp.Code, check.Equals, http.StatusNotFound)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/x/miracle", ""))
	c.Check(resp.Code, check.Equals, http.StatusNotFound)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/miracle", ""))
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
}

func (s *Suite) TestPingOverride(c *check.C) {
	var ok bool
	h := &Handler{
		Token: goodToken,
		Routes: Routes{
			"ping": func() error {
				ok = !ok
				if ok {
					return nil
				} else {
					return errors.New("good error")
				}
			},
		},
	}
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/ping", goodToken))
	s.checkHealthy(c, resp)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, s.request("/ping", goodToken))
	s.checkUnhealthy(c, resp)
}

func (s *Suite) TestZeroValueIsDisabled(c *check.C) {
	resp := httptest.NewRecorder()
	(&Handler{}).ServeHTTP(resp, s.request("/ping", goodToken))
	c.Check(resp.Code, check.Equals, http.StatusNotFound)

	resp = httptest.NewRecorder()
	(&Handler{}).ServeHTTP(resp, s.request("/ping", ""))
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
}

func (s *Suite) request(path, token string) *http.Request {
	u, _ := url.Parse("http://foo.local" + path)
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
	}
	if token != "" {
		req.Header = http.Header{
			"Authorization": {"Bearer " + token},
		}
	}
	return req
}

func (s *Suite) checkHealthy(c *check.C, resp *httptest.ResponseRecorder) {
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, `{"health":"OK"}`+"\n")
}

func (s *Suite) checkUnhealthy(c *check.C, resp *httptest.ResponseRecorder) {
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	c.Assert(err, check.IsNil)
	c.Check(result["health"], check.Equals, "ERROR")
	c.Check(result["error"].(string), check.Not(check.Equals), "")
}
