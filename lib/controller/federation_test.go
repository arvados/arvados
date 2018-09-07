// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"encoding/json"
	"io/ioutil"
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
	log *logrus.Logger
	// testServer and testHandler are the controller being tested,
	// "zhome".
	testServer  *httpserver.Server
	testHandler *Handler
	// remoteServer ("zzzzz") forwards requests to the Rails API
	// provided by the integration test environment.
	remoteServer *httpserver.Server
	// remoteMock ("zmock") appends each incoming request to
	// remoteMockRequests, and returns an empty 200 response.
	remoteMock         *httpserver.Server
	remoteMockRequests []http.Request
}

func (s *FederationSuite) SetUpTest(c *check.C) {
	s.log = logrus.New()
	s.log.Formatter = &logrus.JSONFormatter{}
	s.log.Out = &logWriter{c.Log}

	s.remoteServer = newServerFromIntegrationTestEnv(c)
	c.Assert(s.remoteServer.Start(), check.IsNil)

	s.remoteMock = newServerFromIntegrationTestEnv(c)
	s.remoteMock.Server.Handler = http.HandlerFunc(s.remoteMockHandler)
	c.Assert(s.remoteMock.Start(), check.IsNil)

	nodeProfile := arvados.NodeProfile{
		Controller: arvados.SystemServiceInstance{Listen: ":"},
		RailsAPI:   arvados.SystemServiceInstance{Listen: ":1"}, // local reqs will error "connection refused"
	}
	s.testHandler = &Handler{Cluster: &arvados.Cluster{
		ClusterID:  "zhome",
		PostgreSQL: integrationTestCluster().PostgreSQL,
		NodeProfiles: map[string]arvados.NodeProfile{
			"*": nodeProfile,
		},
	}, NodeProfile: &nodeProfile}
	s.testServer = newServerFromIntegrationTestEnv(c)
	s.testServer.Server.Handler = httpserver.AddRequestIDs(httpserver.LogRequests(s.log, s.testHandler))

	s.testHandler.Cluster.RemoteClusters = map[string]arvados.RemoteCluster{
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
	}

	c.Assert(s.testServer.Start(), check.IsNil)

	s.remoteMockRequests = nil
}

func (s *FederationSuite) remoteMockHandler(w http.ResponseWriter, req *http.Request) {
	s.remoteMockRequests = append(s.remoteMockRequests, *req)
}

func (s *FederationSuite) TearDownTest(c *check.C) {
	if s.remoteServer != nil {
		s.remoteServer.Close()
	}
	if s.testServer != nil {
		s.testServer.Close()
	}
}

func (s *FederationSuite) testRequest(req *http.Request) *http.Response {
	resp := httptest.NewRecorder()
	s.testServer.Server.Handler.ServeHTTP(resp, req)
	return resp.Result()
}

func (s *FederationSuite) TestLocalRequest(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zhome-", 1), nil)
	resp := s.testRequest(req)
	s.checkHandledLocally(c, resp)
}

func (s *FederationSuite) checkHandledLocally(c *check.C, resp *http.Response) {
	// Our "home" controller can't handle local requests because
	// it doesn't have its own stub/test Rails API, so we rely on
	// "connection refused" to indicate the controller tried to
	// proxy the request to its local Rails API.
	c.Check(resp.StatusCode, check.Equals, http.StatusBadGateway)
	s.checkJSONErrorMatches(c, resp, `.*connection refused`)
}

func (s *FederationSuite) TestNoAuth(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusUnauthorized)
	s.checkJSONErrorMatches(c, resp, `Not logged in`)
}

func (s *FederationSuite) TestBadAuth(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusUnauthorized)
	s.checkJSONErrorMatches(c, resp, `Not logged in`)
}

func (s *FederationSuite) TestNoAccess(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.SpectatorToken)
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	s.checkJSONErrorMatches(c, resp, `.*not found`)
}

func (s *FederationSuite) TestGetUnknownRemote(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zz404-", 1), nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	s.checkJSONErrorMatches(c, resp, `.*no proxy available for cluster zz404`)
}

func (s *FederationSuite) TestRemoteError(c *check.C) {
	rc := s.testHandler.Cluster.RemoteClusters["zzzzz"]
	rc.Scheme = "https"
	s.testHandler.Cluster.RemoteClusters["zzzzz"] = rc

	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusBadGateway)
	s.checkJSONErrorMatches(c, resp, `.*HTTP response to HTTPS client`)
}

func (s *FederationSuite) TestGetRemoteWorkflow(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var wf arvados.Workflow
	c.Check(json.NewDecoder(resp.Body).Decode(&wf), check.IsNil)
	c.Check(wf.UUID, check.Equals, arvadostest.WorkflowWithDefinitionYAMLUUID)
	c.Check(wf.OwnerUUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *FederationSuite) TestOptionsMethod(c *check.C) {
	req := httptest.NewRequest("OPTIONS", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Origin", "https://example.com")
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	body, err := ioutil.ReadAll(resp.Body)
	c.Check(err, check.IsNil)
	c.Check(string(body), check.Equals, "")
	c.Check(resp.Header.Get("Access-Control-Allow-Origin"), check.Equals, "*")
	for _, hdr := range []string{"Authorization", "Content-Type"} {
		c.Check(resp.Header.Get("Access-Control-Allow-Headers"), check.Matches, ".*"+hdr+".*")
	}
	for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE"} {
		c.Check(resp.Header.Get("Access-Control-Allow-Methods"), check.Matches, ".*"+method+".*")
	}
}

func (s *FederationSuite) TestRemoteWithTokenInQuery(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zmock-", 1)+"?api_token="+arvadostest.ActiveToken, nil)
	s.testRequest(req)
	c.Assert(len(s.remoteMockRequests), check.Equals, 1)
	pr := s.remoteMockRequests[0]
	// Token is salted and moved from query to Authorization header.
	c.Check(pr.URL.String(), check.Not(check.Matches), `.*api_token=.*`)
	c.Check(pr.Header.Get("Authorization"), check.Equals, "Bearer v2/zzzzz-gj3su-077z32aux8dg2s1/7fd31b61f39c0e82a4155592163218272cedacdc")
}

func (s *FederationSuite) TestLocalTokenSalted(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zmock-", 1), nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	s.testRequest(req)
	c.Assert(len(s.remoteMockRequests), check.Equals, 1)
	pr := s.remoteMockRequests[0]
	// The salted token here has a "zzzzz-" UUID instead of a
	// "ztest-" UUID because ztest's local database has the
	// "zzzzz-" test fixtures. The "secret" part is HMAC(sha1,
	// arvadostest.ActiveToken, "zmock") = "7fd3...".
	c.Check(pr.Header.Get("Authorization"), check.Equals, "Bearer v2/zzzzz-gj3su-077z32aux8dg2s1/7fd31b61f39c0e82a4155592163218272cedacdc")
}

func (s *FederationSuite) TestRemoteTokenNotSalted(c *check.C) {
	// remoteToken can be any v1 token that doesn't appear in
	// ztest's local db.
	remoteToken := "abcdef00000000000000000000000000000000000000000000"
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zmock-", 1), nil)
	req.Header.Set("Authorization", "Bearer "+remoteToken)
	s.testRequest(req)
	c.Assert(len(s.remoteMockRequests), check.Equals, 1)
	pr := s.remoteMockRequests[0]
	c.Check(pr.Header.Get("Authorization"), check.Equals, "Bearer "+remoteToken)
}

func (s *FederationSuite) TestWorkflowCRUD(c *check.C) {
	wf := arvados.Workflow{
		Description: "TestCRUD",
	}
	{
		body := &strings.Builder{}
		json.NewEncoder(body).Encode(&wf)
		req := httptest.NewRequest("POST", "/arvados/v1/workflows", strings.NewReader(url.Values{
			"workflow": {body.String()},
		}.Encode()))
		req.Header.Set("Content-type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		rec := httptest.NewRecorder()
		s.remoteServer.Server.Handler.ServeHTTP(rec, req) // direct to remote -- can't proxy a create req because no uuid
		resp := rec.Result()
		s.checkResponseOK(c, resp)
		json.NewDecoder(resp.Body).Decode(&wf)

		defer func() {
			req := httptest.NewRequest("DELETE", "/arvados/v1/workflows/"+wf.UUID, nil)
			req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
			s.remoteServer.Server.Handler.ServeHTTP(httptest.NewRecorder(), req)
		}()
		c.Check(wf.UUID, check.Not(check.Equals), "")

		c.Assert(wf.ModifiedAt, check.NotNil)
		c.Logf("wf.ModifiedAt: %v", wf.ModifiedAt)
		c.Check(time.Since(*wf.ModifiedAt) < time.Minute, check.Equals, true)
	}
	for _, method := range []string{"PATCH", "PUT", "POST"} {
		form := url.Values{
			"workflow": {`{"description": "Updated with ` + method + `"}`},
		}
		if method == "POST" {
			form["_method"] = []string{"PATCH"}
		}
		req := httptest.NewRequest(method, "/arvados/v1/workflows/"+wf.UUID, strings.NewReader(form.Encode()))
		req.Header.Set("Content-type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp := s.testRequest(req)
		s.checkResponseOK(c, resp)
		err := json.NewDecoder(resp.Body).Decode(&wf)
		c.Check(err, check.IsNil)

		c.Check(wf.Description, check.Equals, "Updated with "+method)
	}
	{
		req := httptest.NewRequest("DELETE", "/arvados/v1/workflows/"+wf.UUID, nil)
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp := s.testRequest(req)
		s.checkResponseOK(c, resp)
		err := json.NewDecoder(resp.Body).Decode(&wf)
		c.Check(err, check.IsNil)
	}
	{
		req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+wf.UUID, nil)
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp := s.testRequest(req)
		c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	}
}

func (s *FederationSuite) checkResponseOK(c *check.C, resp *http.Response) {
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		c.Logf("... response body = %q, %v\n", body, err)
	}
}

func (s *FederationSuite) checkJSONErrorMatches(c *check.C, resp *http.Response, re string) {
	var jresp httpserver.ErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&jresp)
	c.Check(err, check.IsNil)
	c.Assert(len(jresp.Errors), check.Equals, 1)
	c.Check(jresp.Errors[0], check.Matches, re)
}

func (s *FederationSuite) TestGetRemoteCollection(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/collections/"+arvadostest.UserAgreementCollection, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var col arvados.Collection
	c.Check(json.NewDecoder(resp.Body).Decode(&col), check.IsNil)
	c.Check(col.UUID, check.Equals, arvadostest.UserAgreementCollection)
	c.Check(col.ManifestText, check.Matches,
		`\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025\+Rzzzzz-[0-9a-f]{40}@[0-9a-f]{8} 0:249025:GNU_General_Public_License,_version_3.pdf
`)

	// Confirm the regular expression identifies other groups of hints correctly
	c.Check(SignedLocatorPattern.FindStringSubmatch(`6a4ff0499484c6c79c95cd8c566bd25f+249025+B1+C2+A05227438989d04712ea9ca1c91b556cef01d5cc7@5ba5405b+D3+E4`),
		check.DeepEquals,
		[]string{"6a4ff0499484c6c79c95cd8c566bd25f+249025+B1+C2+A05227438989d04712ea9ca1c91b556cef01d5cc7@5ba5405b+D3+E4",
			"6a4ff0499484c6c79c95cd8c566bd25f+249025",
			"+B1+C2", "+C2",
			"+A05227438989d04712ea9ca1c91b556cef01d5cc7@5ba5405b",
			"+D3+E4", "+E4"})
}
