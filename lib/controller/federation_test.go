// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"github.com/Sirupsen/logrus"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
var _ = check.Suite(&FederationSuite{})

type FederationSuite struct {
	log          *logrus.Logger
	localServer  *httpserver.Server
	remoteServer *httpserver.Server
	handler      *Handler
}

func (s *FederationSuite) SetUpTest(c *check.C) {
	s.log = logrus.New()
	s.log.Formatter = &logrus.JSONFormatter{}
	s.log.Out = &logWriter{c.Log}

	s.remoteServer = newServerFromIntegrationTestEnv(c)
	c.Assert(s.remoteServer.Start(), check.IsNil)

	nodeProfile := arvados.NodeProfile{
		Controller: arvados.SystemServiceInstance{Listen: ":"},
		RailsAPI:   arvados.SystemServiceInstance{Listen: ":1"}, // local reqs will error "connection refused"
	}
	s.handler = &Handler{Cluster: &arvados.Cluster{
		ClusterID: "zhome",
		NodeProfiles: map[string]arvados.NodeProfile{
			"*": nodeProfile,
		},
	}, NodeProfile: &nodeProfile}
	s.localServer = newServerFromIntegrationTestEnv(c)
	s.localServer.Server.Handler = httpserver.AddRequestIDs(httpserver.LogRequests(s.log, s.handler))
	s.handler.Cluster.RemoteClusters = map[string]arvados.RemoteCluster{
		"zzzzz": {
			Host:   s.remoteServer.Addr,
			Proxy:  true,
			Scheme: "http",
		},
	}
	c.Assert(s.localServer.Start(), check.IsNil)
}

func (s *FederationSuite) TearDownTest(c *check.C) {
	if s.remoteServer != nil {
		s.remoteServer.Close()
	}
	if s.localServer != nil {
		s.localServer.Close()
	}
}

func (s *FederationSuite) TestLocalRequest(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zhome-", 1), nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	s.checkHandledLocally(c, resp)
}

func (s *FederationSuite) checkHandledLocally(c *check.C, resp *httptest.ResponseRecorder) {
	// Our "home" controller can't handle local requests because
	// it doesn't have its own stub/test Rails API, so we rely on
	// "connection refused" to indicate the controller tried to
	// proxy the request to its local Rails API.
	c.Check(resp.Code, check.Equals, http.StatusInternalServerError)
	s.checkJSONErrorMatches(c, resp, `.*connection refused`)
}

func (s *FederationSuite) TestNoAuth(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
	s.checkJSONErrorMatches(c, resp, `Not logged in`)
}

func (s *FederationSuite) TestBadAuth(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
	s.checkJSONErrorMatches(c, resp, `Not logged in`)
}

func (s *FederationSuite) TestNoAccess(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.SpectatorToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	s.checkJSONErrorMatches(c, resp, `.*not found`)
}

func (s *FederationSuite) TestGetUnknownRemote(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zz404-", 1), nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	s.checkJSONErrorMatches(c, resp, `.*no proxy available for cluster zz404`)
}

func (s *FederationSuite) TestRemoteError(c *check.C) {
	rc := s.handler.Cluster.RemoteClusters["zzzzz"]
	rc.Scheme = "https"
	s.handler.Cluster.RemoteClusters["zzzzz"] = rc

	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusInternalServerError)
	s.checkJSONErrorMatches(c, resp, `.*HTTP response to HTTPS client`)
}

func (s *FederationSuite) TestGetRemoteWorkflow(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var wf arvados.Workflow
	c.Check(json.Unmarshal(resp.Body.Bytes(), &wf), check.IsNil)
	c.Check(wf.UUID, check.Equals, arvadostest.WorkflowWithDefinitionYAMLUUID)
	c.Check(wf.OwnerUUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *FederationSuite) TestUpdateRemoteWorkflow(c *check.C) {
	updateDescription := func(descr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PATCH", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, strings.NewReader(url.Values{
			"workflow": {`{"description":"` + descr + `"}`},
		}.Encode()))
		req.Header.Set("Content-type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		s.checkResponseOK(c, resp)
		return resp
	}

	// Update description twice so running this test twice in a
	// row still causes ModifiedAt to change
	updateDescription("updated once by TestUpdateRemoteWorkflow")
	resp := updateDescription("updated twice by TestUpdateRemoteWorkflow")

	var wf arvados.Workflow
	c.Check(json.Unmarshal(resp.Body.Bytes(), &wf), check.IsNil)
	c.Check(wf.UUID, check.Equals, arvadostest.WorkflowWithDefinitionYAMLUUID)
	c.Assert(wf.ModifiedAt, check.NotNil)
	c.Logf("%s", *wf.ModifiedAt)
	c.Check(time.Since(*wf.ModifiedAt) < time.Minute, check.Equals, true)
}

func (s *FederationSuite) checkResponseOK(c *check.C, resp *httptest.ResponseRecorder) {
	c.Check(resp.Code, check.Equals, http.StatusOK)
	if resp.Code != http.StatusOK {
		c.Logf("... response body = %s\n", resp.Body.String())
	}
}

func (s *FederationSuite) checkJSONErrorMatches(c *check.C, resp *httptest.ResponseRecorder, re string) {
	var jresp httpserver.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Assert(len(jresp.Errors), check.Equals, 1)
	c.Check(jresp.Errors[0], check.Matches, re)
}
