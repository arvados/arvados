// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&HandlerSuite{})

type HandlerSuite struct {
	cluster *arvados.Cluster
	handler http.Handler
}

func (s *HandlerSuite) SetUpTest(c *check.C) {
	s.cluster = &arvados.Cluster{
		ClusterID: "zzzzz",
		NodeProfiles: map[string]arvados.NodeProfile{
			"*": {
				Controller: arvados.SystemServiceInstance{Listen: ":"},
				RailsAPI:   arvados.SystemServiceInstance{Listen: os.Getenv("ARVADOS_TEST_API_HOST"), TLS: true},
			},
		},
	}
	node := s.cluster.NodeProfiles["*"]
	s.handler = newHandler(s.cluster, &node)
}

func (s *HandlerSuite) TestProxyDiscoveryDoc(c *check.C) {
	req := httptest.NewRequest("GET", "/discovery/v1/apis/arvados/v1/rest", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var dd arvados.DiscoveryDocument
	err := json.Unmarshal(resp.Body.Bytes(), &dd)
	c.Check(err, check.IsNil)
	c.Check(dd.BlobSignatureTTL, check.Not(check.Equals), int64(0))
	c.Check(dd.BlobSignatureTTL > 0, check.Equals, true)
	c.Check(len(dd.Resources), check.Not(check.Equals), 0)
	c.Check(len(dd.Schemas), check.Not(check.Equals), 0)
}

func (s *HandlerSuite) TestRequestTimeout(c *check.C) {
	s.cluster.HTTPRequestTimeout = arvados.Duration(time.Nanosecond)
	req := httptest.NewRequest("GET", "/discovery/v1/apis/arvados/v1/rest", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusInternalServerError)
	var jresp httpserver.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Assert(len(jresp.Errors), check.Equals, 1)
	c.Check(jresp.Errors[0], check.Matches, `.*context deadline exceeded`)
}

func (s *HandlerSuite) TestProxyWithoutToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
	jresp := map[string]interface{}{}
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Check(jresp["errors"], check.FitsTypeOf, []interface{}{})
}

func (s *HandlerSuite) TestProxyWithToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var u arvados.User
	err := json.Unmarshal(resp.Body.Bytes(), &u)
	c.Check(err, check.IsNil)
	c.Check(u.UUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *HandlerSuite) TestProxyWithTokenInRequestBody(c *check.C) {
	req := httptest.NewRequest("POST", "/arvados/v1/users/current", strings.NewReader(url.Values{
		"_method":   {"GET"},
		"api_token": {arvadostest.ActiveToken},
	}.Encode()))
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var u arvados.User
	err := json.Unmarshal(resp.Body.Bytes(), &u)
	c.Check(err, check.IsNil)
	c.Check(u.UUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *HandlerSuite) TestProxyNotFound(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/xyzzy", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	jresp := map[string]interface{}{}
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Check(jresp["errors"], check.FitsTypeOf, []interface{}{})
}
