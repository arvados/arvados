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
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

var enableBetaController14287 bool

// Gocheck boilerplate
func Test(t *testing.T) {
	for _, enableBetaController14287 = range []bool{false, true} {
		check.TestingT(t)
	}
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

		EnableBetaController14287: enableBetaController14287,
	}
	s.cluster.TLS.Insecure = true
	arvadostest.SetServiceURL(&s.cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	arvadostest.SetServiceURL(&s.cluster.Services.Controller, "http://localhost:/")
	s.handler = newHandler(s.ctx, s.cluster, "", prometheus.NewRegistry())
}

func (s *HandlerSuite) TearDownTest(c *check.C) {
	s.cancel()
}

func (s *HandlerSuite) TestConfigExport(c *check.C) {
	s.cluster.ManagementToken = "secret"
	s.cluster.SystemRootToken = "secret"
	s.cluster.Collections.BlobSigning = true
	s.cluster.Collections.BlobSigningTTL = arvados.Duration(23 * time.Second)
	for _, method := range []string{"GET", "OPTIONS"} {
		req := httptest.NewRequest(method, "/arvados/v1/config", nil)
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, `*`)
		c.Check(resp.Header().Get("Access-Control-Allow-Methods"), check.Matches, `.*\bGET\b.*`)
		c.Check(resp.Header().Get("Access-Control-Allow-Headers"), check.Matches, `.+`)
		if method == "OPTIONS" {
			c.Check(resp.Body.String(), check.HasLen, 0)
			continue
		}
		var cluster arvados.Cluster
		c.Log(resp.Body.String())
		err := json.Unmarshal(resp.Body.Bytes(), &cluster)
		c.Check(err, check.IsNil)
		c.Check(cluster.ManagementToken, check.Equals, "")
		c.Check(cluster.SystemRootToken, check.Equals, "")
		c.Check(cluster.Collections.BlobSigning, check.DeepEquals, true)
		c.Check(cluster.Collections.BlobSigningTTL, check.Equals, arvados.Duration(23*time.Second))
	}
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
	s.cluster.API.RequestTimeout = arvados.Duration(time.Nanosecond)
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
	s.cluster.Login.ProviderAppID = "test"
	s.cluster.Login.ProviderAppSecret = "test"
	req := httptest.NewRequest("GET", "https://0.0.0.0:1/login?return_to=foo", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	if !c.Check(resp.Code, check.Equals, http.StatusFound) {
		c.Log(resp.Body.String())
	}
	// Old "proxy entire request" code path returns an absolute
	// URL. New lib/controller/federation code path returns a
	// relative URL.
	c.Check(resp.Header().Get("Location"), check.Matches, `(https://0.0.0.0:1)?/auth/joshid\?return_to=%2Cfoo&?`)
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
