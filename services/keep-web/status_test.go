// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"gopkg.in/check.v1"
)

func (s *UnitSuite) TestStatus(c *check.C) {
	u, _ := url.Parse("http://keep-web.example/status.json")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
	}
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)

	var status map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&status)
	c.Check(err, check.IsNil)
	c.Check(status["Version"], check.Not(check.Equals), "")
}

func (s *IntegrationSuite) TestNoStatusFromVHost(c *check.C) {
	u, _ := url.Parse("http://" + arvadostest.FooCollection + "--keep-web.example/status.json")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Authorization": {"OAuth2 " + arvadostest.ActiveToken},
		},
	}
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
}
