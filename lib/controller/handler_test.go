// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
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
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
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
	ctx     context.Context
	cancel  context.CancelFunc
}

func (s *HandlerSuite) SetUpTest(c *check.C) {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.ctx = ctxlog.Context(s.ctx, ctxlog.New(os.Stderr, "json", "debug"))
	s.cluster = &arvados.Cluster{
		ClusterID:  "zzzzz",
		PostgreSQL: integrationTestCluster().PostgreSQL,
		NodeProfiles: map[string]arvados.NodeProfile{
			"*": {
				Controller: arvados.SystemServiceInstance{Listen: ":"},
				RailsAPI:   arvados.SystemServiceInstance{Listen: os.Getenv("ARVADOS_TEST_API_HOST"), TLS: true, Insecure: true},
			},
		},
	}
	node := s.cluster.NodeProfiles["*"]
	s.handler = newHandler(s.ctx, s.cluster, &node, "")
}

func (s *HandlerSuite) TearDownTest(c *check.C) {
	s.cancel()
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
	c.Check(resp.Code, check.Equals, http.StatusBadGateway)
	var jresp httpserver.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Assert(len(jresp.Errors), check.Equals, 1)
	c.Check(jresp.Errors[0], check.Matches, `.*context deadline exceeded.*`)
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
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
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

func (s *HandlerSuite) TestProxyRedirect(c *check.C) {
	req := httptest.NewRequest("GET", "https://0.0.0.0:1/login?return_to=foo", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusFound)
	c.Check(resp.Header().Get("Location"), check.Matches, `https://0.0.0.0:1/auth/joshid\?return_to=%2Cfoo&?`)
}

func (s *HandlerSuite) TestValidateV1APIToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	user, err := s.handler.(*Handler).validateAPItoken(req, arvadostest.ActiveToken)
	c.Assert(err, check.IsNil)
	c.Check(user.Authorization.UUID, check.Equals, arvadostest.ActiveTokenUUID)
	c.Check(user.Authorization.APIToken, check.Equals, arvadostest.ActiveToken)
	c.Check(user.Authorization.Scopes, check.DeepEquals, []string{"all"})
	c.Check(user.UUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *HandlerSuite) TestValidateV2APIToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	user, err := s.handler.(*Handler).validateAPItoken(req, arvadostest.ActiveTokenV2)
	c.Assert(err, check.IsNil)
	c.Check(user.Authorization.UUID, check.Equals, arvadostest.ActiveTokenUUID)
	c.Check(user.Authorization.APIToken, check.Equals, arvadostest.ActiveToken)
	c.Check(user.Authorization.Scopes, check.DeepEquals, []string{"all"})
	c.Check(user.UUID, check.Equals, arvadostest.ActiveUserUUID)
	c.Check(user.Authorization.TokenV2(), check.Equals, arvadostest.ActiveTokenV2)
}

func (s *HandlerSuite) TestCreateAPIToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	auth, err := s.handler.(*Handler).createAPItoken(req, arvadostest.ActiveUserUUID, nil)
	c.Assert(err, check.IsNil)
	c.Check(auth.Scopes, check.DeepEquals, []string{"all"})

	user, err := s.handler.(*Handler).validateAPItoken(req, auth.TokenV2())
	c.Assert(err, check.IsNil)
	c.Check(user.Authorization.UUID, check.Equals, auth.UUID)
	c.Check(user.Authorization.APIToken, check.Equals, auth.APIToken)
	c.Check(user.Authorization.Scopes, check.DeepEquals, []string{"all"})
	c.Check(user.UUID, check.Equals, arvadostest.ActiveUserUUID)
	c.Check(user.Authorization.TokenV2(), check.Equals, auth.TokenV2())
}
