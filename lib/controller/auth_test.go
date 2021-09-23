// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
var _ = check.Suite(&AuthSuite{})

type AuthSuite struct {
	log logrus.FieldLogger
	// testServer and testHandler are the controller being tested,
	// "zhome".
	testServer  *httpserver.Server
	testHandler *Handler
	// remoteServer ("zzzzz") forwards requests to the Rails API
	// provided by the integration test environment.
	remoteServer *httpserver.Server
	// remoteMock ("zmock") appends each incoming request to
	// remoteMockRequests, and returns 200 with an empty JSON
	// object.
	remoteMock         *httpserver.Server
	remoteMockRequests []http.Request

	fakeProvider *arvadostest.OIDCProvider
}

func (s *AuthSuite) SetUpTest(c *check.C) {
	s.log = ctxlog.TestLogger(c)

	s.remoteServer = newServerFromIntegrationTestEnv(c)
	c.Assert(s.remoteServer.Start(), check.IsNil)

	s.remoteMock = newServerFromIntegrationTestEnv(c)
	s.remoteMock.Server.Handler = http.HandlerFunc(http.NotFound)
	c.Assert(s.remoteMock.Start(), check.IsNil)

	s.fakeProvider = arvadostest.NewOIDCProvider(c)
	s.fakeProvider.AuthEmail = "active-user@arvados.local"
	s.fakeProvider.AuthEmailVerified = true
	s.fakeProvider.AuthName = "Fake User Name"
	s.fakeProvider.ValidCode = fmt.Sprintf("abcdefgh-%d", time.Now().Unix())
	s.fakeProvider.PeopleAPIResponse = map[string]interface{}{}
	s.fakeProvider.ValidClientID = "test%client$id"
	s.fakeProvider.ValidClientSecret = "test#client/secret"

	cluster := &arvados.Cluster{
		ClusterID:       "zhome",
		PostgreSQL:      integrationTestCluster().PostgreSQL,
		SystemRootToken: arvadostest.SystemRootToken,
	}
	cluster.TLS.Insecure = true
	cluster.API.MaxItemsPerResponse = 1000
	cluster.API.MaxRequestAmplification = 4
	cluster.API.RequestTimeout = arvados.Duration(5 * time.Minute)
	arvadostest.SetServiceURL(&cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	arvadostest.SetServiceURL(&cluster.Services.Controller, "http://localhost/")

	cluster.RemoteClusters = map[string]arvados.RemoteCluster{
		"zzzzz": {
			Host:   s.remoteServer.Addr,
			Proxy:  true,
			Scheme: "http",
		},
		"zmock": {
			Host:   s.remoteMock.Addr,
			Proxy:  true,
			Scheme: "http",
		},
		"*": {
			Scheme: "https",
		},
	}
	cluster.Login.OpenIDConnect.Enable = true
	cluster.Login.OpenIDConnect.Issuer = s.fakeProvider.Issuer.URL
	cluster.Login.OpenIDConnect.ClientID = s.fakeProvider.ValidClientID
	cluster.Login.OpenIDConnect.ClientSecret = s.fakeProvider.ValidClientSecret
	cluster.Login.OpenIDConnect.EmailClaim = "email"
	cluster.Login.OpenIDConnect.EmailVerifiedClaim = "email_verified"
	cluster.Login.OpenIDConnect.AcceptAccessToken = true
	cluster.Login.OpenIDConnect.AcceptAccessTokenScope = ""

	s.testHandler = &Handler{Cluster: cluster}
	s.testServer = newServerFromIntegrationTestEnv(c)
	s.testServer.Server.BaseContext = func(net.Listener) context.Context {
		return ctxlog.Context(context.Background(), s.log)
	}
	s.testServer.Server.Handler = httpserver.AddRequestIDs(httpserver.LogRequests(s.testHandler))
	c.Assert(s.testServer.Start(), check.IsNil)
}

func (s *AuthSuite) TestLocalOIDCAccessToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	req.Header.Set("Authorization", "Bearer "+s.fakeProvider.ValidAccessToken())
	rr := httptest.NewRecorder()
	s.testServer.Server.Handler.ServeHTTP(rr, req)
	resp := rr.Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var u arvados.User
	c.Check(json.NewDecoder(resp.Body).Decode(&u), check.IsNil)
	c.Check(u.UUID, check.Equals, arvadostest.ActiveUserUUID)
	c.Check(u.OwnerUUID, check.Equals, "zzzzz-tpzed-000000000000000")

	// Request again to exercise cache.
	req = httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	req.Header.Set("Authorization", "Bearer "+s.fakeProvider.ValidAccessToken())
	rr = httptest.NewRecorder()
	s.testServer.Server.Handler.ServeHTTP(rr, req)
	resp = rr.Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
}
