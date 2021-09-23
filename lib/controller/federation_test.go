// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
var _ = check.Suite(&FederationSuite{})

type FederationSuite struct {
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
}

func (s *FederationSuite) SetUpTest(c *check.C) {
	s.log = ctxlog.TestLogger(c)

	s.remoteServer = newServerFromIntegrationTestEnv(c)
	c.Assert(s.remoteServer.Start(), check.IsNil)

	s.remoteMock = newServerFromIntegrationTestEnv(c)
	s.remoteMock.Server.Handler = http.HandlerFunc(s.remoteMockHandler)
	c.Assert(s.remoteMock.Start(), check.IsNil)

	cluster := &arvados.Cluster{
		ClusterID:  "zhome",
		PostgreSQL: integrationTestCluster().PostgreSQL,
	}
	cluster.TLS.Insecure = true
	cluster.API.MaxItemsPerResponse = 1000
	cluster.API.MaxRequestAmplification = 4
	cluster.API.RequestTimeout = arvados.Duration(5 * time.Minute)
	cluster.Collections.BlobSigning = true
	cluster.Collections.BlobSigningKey = arvadostest.BlobSigningKey
	cluster.Collections.BlobSigningTTL = arvados.Duration(time.Hour * 24 * 14)
	arvadostest.SetServiceURL(&cluster.Services.RailsAPI, "http://localhost:1/")
	arvadostest.SetServiceURL(&cluster.Services.Controller, "http://localhost:/")
	s.testHandler = &Handler{Cluster: cluster}
	s.testServer = newServerFromIntegrationTestEnv(c)
	s.testServer.Server.BaseContext = func(net.Listener) context.Context {
		return ctxlog.Context(context.Background(), s.log)
	}
	s.testServer.Server.Handler = httpserver.AddRequestIDs(httpserver.LogRequests(s.testHandler))

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

	c.Assert(s.testServer.Start(), check.IsNil)

	s.remoteMockRequests = nil
}

func (s *FederationSuite) remoteMockHandler(w http.ResponseWriter, req *http.Request) {
	b := &bytes.Buffer{}
	io.Copy(b, req.Body)
	req.Body.Close()
	req.Body = ioutil.NopCloser(b)
	s.remoteMockRequests = append(s.remoteMockRequests, *req)
	// Repond 200 with a valid JSON object
	fmt.Fprint(w, "{}")
}

func (s *FederationSuite) TearDownTest(c *check.C) {
	if s.remoteServer != nil {
		s.remoteServer.Close()
	}
	if s.testServer != nil {
		s.testServer.Close()
	}
}

func (s *FederationSuite) testRequest(req *http.Request) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	s.testServer.Server.Handler.ServeHTTP(resp, req)
	return resp
}

func (s *FederationSuite) TestLocalRequest(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zhome-", 1), nil)
	resp := s.testRequest(req).Result()
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
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusUnauthorized)
	s.checkJSONErrorMatches(c, resp, `Not logged in.*`)
}

func (s *FederationSuite) TestBadAuth(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusUnauthorized)
	s.checkJSONErrorMatches(c, resp, `Not logged in.*`)
}

func (s *FederationSuite) TestNoAccess(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.SpectatorToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	s.checkJSONErrorMatches(c, resp, `.*not found.*`)
}

func (s *FederationSuite) TestGetUnknownRemote(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zz404-", 1), nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	s.checkJSONErrorMatches(c, resp, `.*no proxy available for cluster zz404`)
}

func (s *FederationSuite) TestRemoteError(c *check.C) {
	rc := s.testHandler.Cluster.RemoteClusters["zzzzz"]
	rc.Scheme = "https"
	s.testHandler.Cluster.RemoteClusters["zzzzz"] = rc

	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusBadGateway)
	s.checkJSONErrorMatches(c, resp, `.*HTTP response to HTTPS client`)
}

func (s *FederationSuite) TestGetRemoteWorkflow(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var wf arvados.Workflow
	c.Check(json.NewDecoder(resp.Body).Decode(&wf), check.IsNil)
	c.Check(wf.UUID, check.Equals, arvadostest.WorkflowWithDefinitionYAMLUUID)
	c.Check(wf.OwnerUUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *FederationSuite) TestOptionsMethod(c *check.C) {
	req := httptest.NewRequest("OPTIONS", "/arvados/v1/workflows/"+arvadostest.WorkflowWithDefinitionYAMLUUID, nil)
	req.Header.Set("Origin", "https://example.com")
	resp := s.testRequest(req).Result()
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
	s.testRequest(req).Result()
	c.Assert(s.remoteMockRequests, check.HasLen, 1)
	pr := s.remoteMockRequests[0]
	// Token is salted and moved from query to Authorization header.
	c.Check(pr.URL.String(), check.Not(check.Matches), `.*api_token=.*`)
	c.Check(pr.Header.Get("Authorization"), check.Equals, "Bearer v2/zzzzz-gj3su-077z32aux8dg2s1/7fd31b61f39c0e82a4155592163218272cedacdc")
}

func (s *FederationSuite) TestLocalTokenSalted(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	for _, path := range []string{
		// During the transition to the strongly typed
		// controller implementation (#14287), workflows and
		// collections test different code paths.
		"/arvados/v1/workflows/" + strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zmock-", 1),
		"/arvados/v1/collections/" + strings.Replace(arvadostest.UserAgreementCollection, "zzzzz-", "zmock-", 1),
	} {
		c.Log("testing path ", path)
		s.remoteMockRequests = nil
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		s.testRequest(req).Result()
		c.Assert(s.remoteMockRequests, check.HasLen, 1)
		pr := s.remoteMockRequests[0]
		// The salted token here has a "zzzzz-" UUID instead of a
		// "ztest-" UUID because ztest's local database has the
		// "zzzzz-" test fixtures. The "secret" part is HMAC(sha1,
		// arvadostest.ActiveToken, "zmock") = "7fd3...".
		c.Check(pr.Header.Get("Authorization"), check.Equals, "Bearer v2/zzzzz-gj3su-077z32aux8dg2s1/7fd31b61f39c0e82a4155592163218272cedacdc")
	}
}

func (s *FederationSuite) TestRemoteTokenNotSalted(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	// remoteToken can be any v1 token that doesn't appear in
	// ztest's local db.
	remoteToken := "abcdef00000000000000000000000000000000000000000000"

	for _, path := range []string{
		// During the transition to the strongly typed
		// controller implementation (#14287), workflows and
		// collections test different code paths.
		"/arvados/v1/workflows/" + strings.Replace(arvadostest.WorkflowWithDefinitionYAMLUUID, "zzzzz-", "zmock-", 1),
		"/arvados/v1/collections/" + strings.Replace(arvadostest.UserAgreementCollection, "zzzzz-", "zmock-", 1),
	} {
		c.Log("testing path ", path)
		s.remoteMockRequests = nil
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+remoteToken)
		s.testRequest(req).Result()
		c.Assert(s.remoteMockRequests, check.HasLen, 1)
		pr := s.remoteMockRequests[0]
		c.Check(pr.Header.Get("Authorization"), check.Equals, "Bearer "+remoteToken)
	}
}

func (s *FederationSuite) TestWorkflowCRUD(c *check.C) {
	var wf arvados.Workflow
	{
		req := httptest.NewRequest("POST", "/arvados/v1/workflows", strings.NewReader(url.Values{
			"workflow": {`{"description": "TestCRUD"}`},
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
		resp := s.testRequest(req).Result()
		s.checkResponseOK(c, resp)
		err := json.NewDecoder(resp.Body).Decode(&wf)
		c.Check(err, check.IsNil)

		c.Check(wf.Description, check.Equals, "Updated with "+method)
	}
	{
		req := httptest.NewRequest("DELETE", "/arvados/v1/workflows/"+wf.UUID, nil)
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp := s.testRequest(req).Result()
		s.checkResponseOK(c, resp)
		err := json.NewDecoder(resp.Body).Decode(&wf)
		c.Check(err, check.IsNil)
	}
	{
		req := httptest.NewRequest("GET", "/arvados/v1/workflows/"+wf.UUID, nil)
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp := s.testRequest(req).Result()
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
	c.Assert(jresp.Errors, check.HasLen, 1)
	c.Check(jresp.Errors[0], check.Matches, re)
}

func (s *FederationSuite) localServiceHandler(c *check.C, h http.Handler) *httpserver.Server {
	srv := &httpserver.Server{
		Server: http.Server{
			Handler: h,
		},
	}
	c.Assert(srv.Start(), check.IsNil)
	arvadostest.SetServiceURL(&s.testHandler.Cluster.Services.RailsAPI, "http://"+srv.Addr)
	return srv
}

func (s *FederationSuite) localServiceReturns404(c *check.C) *httpserver.Server {
	return s.localServiceHandler(c, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/arvados/v1/api_client_authorizations/current" {
			if req.Header.Get("Authorization") == "Bearer "+arvadostest.ActiveToken {
				json.NewEncoder(w).Encode(arvados.APIClientAuthorization{UUID: arvadostest.ActiveTokenUUID, APIToken: arvadostest.ActiveToken, Scopes: []string{"all"}})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		} else if req.URL.Path == "/arvados/v1/users/current" {
			if req.Header.Get("Authorization") == "Bearer "+arvadostest.ActiveToken {
				json.NewEncoder(w).Encode(arvados.User{UUID: arvadostest.ActiveUserUUID})
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		} else {
			w.WriteHeader(404)
		}
	}))
}

func (s *FederationSuite) TestGetLocalCollection(c *check.C) {
	s.testHandler.Cluster.ClusterID = "zzzzz"
	arvadostest.SetServiceURL(&s.testHandler.Cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))

	// HTTP GET

	req := httptest.NewRequest("GET", "/arvados/v1/collections/"+arvadostest.UserAgreementCollection, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()

	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var col arvados.Collection
	c.Check(json.NewDecoder(resp.Body).Decode(&col), check.IsNil)
	c.Check(col.UUID, check.Equals, arvadostest.UserAgreementCollection)
	c.Check(col.ManifestText, check.Matches,
		`\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025\+A[0-9a-f]{40}@[0-9a-f]{8} 0:249025:GNU_General_Public_License,_version_3.pdf
`)

	// HTTP POST with _method=GET as a form parameter

	req = httptest.NewRequest("POST", "/arvados/v1/collections/"+arvadostest.UserAgreementCollection, bytes.NewBufferString((url.Values{
		"_method": {"GET"},
	}).Encode()))
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp = s.testRequest(req).Result()

	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	col = arvados.Collection{}
	c.Check(json.NewDecoder(resp.Body).Decode(&col), check.IsNil)
	c.Check(col.UUID, check.Equals, arvadostest.UserAgreementCollection)
	c.Check(col.ManifestText, check.Matches,
		`\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025\+A[0-9a-f]{40}@[0-9a-f]{8} 0:249025:GNU_General_Public_License,_version_3.pdf
`)
}

func (s *FederationSuite) TestGetRemoteCollection(c *check.C) {
	defer s.localServiceReturns404(c).Close()

	req := httptest.NewRequest("GET", "/arvados/v1/collections/"+arvadostest.UserAgreementCollection, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var col arvados.Collection
	c.Check(json.NewDecoder(resp.Body).Decode(&col), check.IsNil)
	c.Check(col.UUID, check.Equals, arvadostest.UserAgreementCollection)
	c.Check(col.ManifestText, check.Matches,
		`\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025\+Rzzzzz-[0-9a-f]{40}@[0-9a-f]{8} 0:249025:GNU_General_Public_License,_version_3.pdf
`)
}

func (s *FederationSuite) TestGetRemoteCollectionError(c *check.C) {
	defer s.localServiceReturns404(c).Close()

	req := httptest.NewRequest("GET", "/arvados/v1/collections/zzzzz-4zz18-fakefakefakefak", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
}

func (s *FederationSuite) TestSignedLocatorPattern(c *check.C) {
	// Confirm the regular expression identifies other groups of hints correctly
	c.Check(keepclient.SignedLocatorRe.FindStringSubmatch(`6a4ff0499484c6c79c95cd8c566bd25f+249025+B1+C2+A05227438989d04712ea9ca1c91b556cef01d5cc7@5ba5405b+D3+E4`),
		check.DeepEquals,
		[]string{"6a4ff0499484c6c79c95cd8c566bd25f+249025+B1+C2+A05227438989d04712ea9ca1c91b556cef01d5cc7@5ba5405b+D3+E4",
			"6a4ff0499484c6c79c95cd8c566bd25f",
			"+249025",
			"+B1+C2", "+C2",
			"+A05227438989d04712ea9ca1c91b556cef01d5cc7@5ba5405b",
			"05227438989d04712ea9ca1c91b556cef01d5cc7", "5ba5405b",
			"+D3+E4", "+E4"})
}

func (s *FederationSuite) TestGetLocalCollectionByPDH(c *check.C) {
	arvadostest.SetServiceURL(&s.testHandler.Cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))

	req := httptest.NewRequest("GET", "/arvados/v1/collections/"+arvadostest.UserAgreementPDH, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()

	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var col arvados.Collection
	c.Check(json.NewDecoder(resp.Body).Decode(&col), check.IsNil)
	c.Check(col.PortableDataHash, check.Equals, arvadostest.UserAgreementPDH)
	c.Check(col.ManifestText, check.Matches,
		`\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025\+A[0-9a-f]{40}@[0-9a-f]{8} 0:249025:GNU_General_Public_License,_version_3.pdf
`)
}

func (s *FederationSuite) TestGetRemoteCollectionByPDH(c *check.C) {
	defer s.localServiceReturns404(c).Close()

	req := httptest.NewRequest("GET", "/arvados/v1/collections/"+arvadostest.UserAgreementPDH, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()

	c.Check(resp.StatusCode, check.Equals, http.StatusOK)

	var col arvados.Collection
	c.Check(json.NewDecoder(resp.Body).Decode(&col), check.IsNil)
	c.Check(col.PortableDataHash, check.Equals, arvadostest.UserAgreementPDH)
	c.Check(col.ManifestText, check.Matches,
		`\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025\+Rzzzzz-[0-9a-f]{40}@[0-9a-f]{8} 0:249025:GNU_General_Public_License,_version_3.pdf
`)
}

func (s *FederationSuite) TestGetCollectionByPDHError(c *check.C) {
	defer s.localServiceReturns404(c).Close()

	// zmock's normal response (200 with an empty body) would
	// change the outcome from 404 to 502
	delete(s.testHandler.Cluster.RemoteClusters, "zmock")

	req := httptest.NewRequest("GET", "/arvados/v1/collections/99999999999999999999999999999999+99", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)

	resp := s.testRequest(req).Result()
	defer resp.Body.Close()

	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
}

func (s *FederationSuite) TestGetCollectionByPDHErrorBadHash(c *check.C) {
	defer s.localServiceReturns404(c).Close()

	// zmock's normal response (200 with an empty body) would
	// change the outcome
	delete(s.testHandler.Cluster.RemoteClusters, "zmock")

	srv2 := &httpserver.Server{
		Server: http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(200)
				// Return a collection where the hash
				// of the manifest text doesn't match
				// PDH that was requested.
				var col arvados.Collection
				col.PortableDataHash = "99999999999999999999999999999999+99"
				col.ManifestText = `. 6a4ff0499484c6c79c95cd8c566bd25f\+249025 0:249025:GNU_General_Public_License,_version_3.pdf
`
				enc := json.NewEncoder(w)
				enc.Encode(col)
			}),
		},
	}

	c.Assert(srv2.Start(), check.IsNil)
	defer srv2.Close()

	// Direct zzzzz to service that returns a 200 result with a bogus manifest_text
	s.testHandler.Cluster.RemoteClusters["zzzzz"] = arvados.RemoteCluster{
		Host:   srv2.Addr,
		Proxy:  true,
		Scheme: "http",
	}

	req := httptest.NewRequest("GET", "/arvados/v1/collections/99999999999999999999999999999999+99", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)

	resp := s.testRequest(req).Result()
	defer resp.Body.Close()

	c.Check(resp.StatusCode, check.Equals, http.StatusBadGateway)
}

func (s *FederationSuite) TestSaltedTokenGetCollectionByPDH(c *check.C) {
	arvadostest.SetServiceURL(&s.testHandler.Cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))

	req := httptest.NewRequest("GET", "/arvados/v1/collections/"+arvadostest.UserAgreementPDH, nil)
	req.Header.Set("Authorization", "Bearer v2/zzzzz-gj3su-077z32aux8dg2s1/282d7d172b6cfdce364c5ed12ddf7417b2d00065")
	resp := s.testRequest(req).Result()

	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var col arvados.Collection
	c.Check(json.NewDecoder(resp.Body).Decode(&col), check.IsNil)
	c.Check(col.PortableDataHash, check.Equals, arvadostest.UserAgreementPDH)
	c.Check(col.ManifestText, check.Matches,
		`\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025\+A[0-9a-f]{40}@[0-9a-f]{8} 0:249025:GNU_General_Public_License,_version_3.pdf
`)
}

func (s *FederationSuite) TestSaltedTokenGetCollectionByPDHError(c *check.C) {
	arvadostest.SetServiceURL(&s.testHandler.Cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))

	// zmock's normal response (200 with an empty body) would
	// change the outcome
	delete(s.testHandler.Cluster.RemoteClusters, "zmock")

	req := httptest.NewRequest("GET", "/arvados/v1/collections/99999999999999999999999999999999+99", nil)
	req.Header.Set("Authorization", "Bearer v2/zzzzz-gj3su-077z32aux8dg2s1/282d7d172b6cfdce364c5ed12ddf7417b2d00065")
	resp := s.testRequest(req).Result()

	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
}

func (s *FederationSuite) TestGetRemoteContainerRequest(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	req := httptest.NewRequest("GET", "/arvados/v1/container_requests/"+arvadostest.QueuedContainerRequestUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var cr arvados.ContainerRequest
	c.Check(json.NewDecoder(resp.Body).Decode(&cr), check.IsNil)
	c.Check(cr.UUID, check.Equals, arvadostest.QueuedContainerRequestUUID)
	c.Check(cr.Priority, check.Equals, 1)
}

func (s *FederationSuite) TestUpdateRemoteContainerRequest(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	setPri := func(pri int) {
		req := httptest.NewRequest("PATCH", "/arvados/v1/container_requests/"+arvadostest.QueuedContainerRequestUUID,
			strings.NewReader(fmt.Sprintf(`{"container_request": {"priority": %d}}`, pri)))
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		req.Header.Set("Content-type", "application/json")
		resp := s.testRequest(req).Result()
		c.Check(resp.StatusCode, check.Equals, http.StatusOK)
		var cr arvados.ContainerRequest
		c.Check(json.NewDecoder(resp.Body).Decode(&cr), check.IsNil)
		c.Check(cr.UUID, check.Equals, arvadostest.QueuedContainerRequestUUID)
		c.Check(cr.Priority, check.Equals, pri)
	}
	setPri(696)
	setPri(1) // Reset fixture so side effect doesn't break other tests.
}

func (s *FederationSuite) TestCreateContainerRequestBadToken(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	// pass cluster_id via query parameter, this allows arvados-controller
	// to avoid parsing the body
	req := httptest.NewRequest("POST", "/arvados/v1/container_requests?cluster_id=zzzzz",
		strings.NewReader(`{"container_request":{}}`))
	req.Header.Set("Authorization", "Bearer abcdefg")
	req.Header.Set("Content-type", "application/json")
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusForbidden)
	var e map[string][]string
	c.Check(json.NewDecoder(resp.Body).Decode(&e), check.IsNil)
	c.Check(e["errors"], check.DeepEquals, []string{"invalid API token"})
}

func (s *FederationSuite) TestCreateRemoteContainerRequest(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	// pass cluster_id via query parameter, this allows arvados-controller
	// to avoid parsing the body
	req := httptest.NewRequest("POST", "/arvados/v1/container_requests?cluster_id=zzzzz",
		strings.NewReader(`{
  "container_request": {
    "name": "hello world",
    "state": "Uncommitted",
    "output_path": "/",
    "container_image": "123",
    "command": ["abc"]
  }
}
`))
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	req.Header.Set("Content-type", "application/json")
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var cr arvados.ContainerRequest
	c.Check(json.NewDecoder(resp.Body).Decode(&cr), check.IsNil)
	c.Check(cr.Name, check.Equals, "hello world")
	c.Check(strings.HasPrefix(cr.UUID, "zzzzz-"), check.Equals, true)
}

// getCRfromMockRequest returns a ContainerRequest with the content of the
// request sent to the remote mock. This function takes into account the
// Content-Type and acts accordingly.
func (s *FederationSuite) getCRfromMockRequest(c *check.C) arvados.ContainerRequest {

	// Body can be a json formated or something like:
	//  cluster_id=zmock&container_request=%7B%22command%22%3A%5B%22abc%22%5D%2C%22container_image%22%3A%22123%22%2C%22...7D
	// or:
	//  "{\"container_request\":{\"command\":[\"abc\"],\"container_image\":\"12...Uncommitted\"}}"

	var cr arvados.ContainerRequest
	data, err := ioutil.ReadAll(s.remoteMockRequests[0].Body)
	c.Check(err, check.IsNil)

	if s.remoteMockRequests[0].Header.Get("Content-Type") == "application/json" {
		// legacy code path sends a JSON request body
		var answerCR struct {
			ContainerRequest arvados.ContainerRequest `json:"container_request"`
		}
		c.Check(json.Unmarshal(data, &answerCR), check.IsNil)
		cr = answerCR.ContainerRequest
	} else if s.remoteMockRequests[0].Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		// new code path sends a form-encoded request body with a JSON-encoded parameter value
		decodedValue, err := url.ParseQuery(string(data))
		c.Check(err, check.IsNil)
		decodedValueCR := decodedValue.Get("container_request")
		c.Check(json.Unmarshal([]byte(decodedValueCR), &cr), check.IsNil)
	} else {
		// mock needs to have Content-Type that we can parse.
		c.Fail()
	}

	return cr
}

func (s *FederationSuite) TestCreateRemoteContainerRequestCheckRuntimeToken(c *check.C) {
	// Send request to zmock and check that outgoing request has
	// runtime_token set with a new random v2 token.

	defer s.localServiceReturns404(c).Close()
	req := httptest.NewRequest("POST", "/arvados/v1/container_requests?cluster_id=zmock",
		strings.NewReader(`{
	  "container_request": {
	    "name": "hello world",
	    "state": "Uncommitted",
	    "output_path": "/",
	    "container_image": "123",
	    "command": ["abc"]
	  }
	}
	`))
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveTokenV2)
	req.Header.Set("Content-type", "application/json")

	// We replace zhome with zzzzz values (RailsAPI, ClusterID, SystemRootToken)
	// SystemRoot token is needed because we check the
	// https://[RailsAPI]/arvados/v1/api_client_authorizations/current
	// https://[RailsAPI]/arvados/v1/users/current and
	// https://[RailsAPI]/auth/controller/callback
	arvadostest.SetServiceURL(&s.testHandler.Cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	s.testHandler.Cluster.ClusterID = "zzzzz"
	s.testHandler.Cluster.SystemRootToken = arvadostest.SystemRootToken
	s.testHandler.Cluster.API.MaxTokenLifetime = arvados.Duration(time.Hour)

	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)

	cr := s.getCRfromMockRequest(c)

	// Runtime token must match zzzzz cluster
	c.Check(cr.RuntimeToken, check.Matches, "v2/zzzzz-gj3su-.*")

	// RuntimeToken must be different than the Original Token we originally did the request with.
	c.Check(cr.RuntimeToken, check.Not(check.Equals), arvadostest.ActiveTokenV2)

	// Runtime token should not have an expiration based on API.MaxTokenLifetime
	req2 := httptest.NewRequest("GET", "/arvados/v1/api_client_authorizations/current", nil)
	req2.Header.Set("Authorization", "Bearer "+cr.RuntimeToken)
	req2.Header.Set("Content-type", "application/json")
	resp = s.testRequest(req2).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var aca arvados.APIClientAuthorization
	c.Check(json.NewDecoder(resp.Body).Decode(&aca), check.IsNil)
	c.Check(aca.ExpiresAt, check.NotNil) // Time.Now()+BlobSigningTTL
	t, _ := time.Parse(time.RFC3339Nano, aca.ExpiresAt)
	c.Check(t.After(time.Now().Add(s.testHandler.Cluster.API.MaxTokenLifetime.Duration())), check.Equals, true)
	c.Check(t.Before(time.Now().Add(s.testHandler.Cluster.Collections.BlobSigningTTL.Duration())), check.Equals, true)
}

func (s *FederationSuite) TestCreateRemoteContainerRequestCheckSetRuntimeToken(c *check.C) {
	// Send request to zmock and check that outgoing request has
	// runtime_token set with the explicitly provided token.

	defer s.localServiceReturns404(c).Close()
	// pass cluster_id via query parameter, this allows arvados-controller
	// to avoid parsing the body
	req := httptest.NewRequest("POST", "/arvados/v1/container_requests?cluster_id=zmock",
		strings.NewReader(`{
	  "container_request": {
	    "name": "hello world",
	    "state": "Uncommitted",
	    "output_path": "/",
	    "container_image": "123",
	    "command": ["abc"],
	    "runtime_token": "xyz"
	  }
	}
	`))
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	req.Header.Set("Content-type", "application/json")
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)

	cr := s.getCRfromMockRequest(c)

	// After mocking around now making sure the runtime_token we sent is still there.
	c.Check(cr.RuntimeToken, check.Equals, "xyz")
}

func (s *FederationSuite) TestCreateRemoteContainerRequestError(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	// pass cluster_id via query parameter, this allows arvados-controller
	// to avoid parsing the body
	req := httptest.NewRequest("POST", "/arvados/v1/container_requests?cluster_id=zz404",
		strings.NewReader(`{
  "container_request": {
    "name": "hello world",
    "state": "Uncommitted",
    "output_path": "/",
    "container_image": "123",
    "command": ["abc"]
  }
}
`))
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	req.Header.Set("Content-type", "application/json")
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
}

func (s *FederationSuite) TestGetRemoteContainer(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	req := httptest.NewRequest("GET", "/arvados/v1/containers/"+arvadostest.QueuedContainerUUID, nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var cn arvados.Container
	c.Check(json.NewDecoder(resp.Body).Decode(&cn), check.IsNil)
	c.Check(cn.UUID, check.Equals, arvadostest.QueuedContainerUUID)
}

func (s *FederationSuite) TestListRemoteContainer(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	req := httptest.NewRequest("GET", "/arvados/v1/containers?count=none&filters="+
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v"]]]`, arvadostest.QueuedContainerUUID)), nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var cn arvados.ContainerList
	c.Check(json.NewDecoder(resp.Body).Decode(&cn), check.IsNil)
	c.Assert(cn.Items, check.HasLen, 1)
	c.Check(cn.Items[0].UUID, check.Equals, arvadostest.QueuedContainerUUID)
}

func (s *FederationSuite) TestListMultiRemoteContainers(c *check.C) {
	defer s.localServiceHandler(c, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		bd, _ := ioutil.ReadAll(req.Body)
		c.Check(string(bd), check.Equals, `_method=GET&count=none&filters=%5B%5B%22uuid%22%2C+%22in%22%2C+%5B%22zhome-xvhdp-cr5queuedcontnr%22%5D%5D%5D&select=%5B%22uuid%22%2C+%22command%22%5D`)
		w.WriteHeader(200)
		w.Write([]byte(`{"kind": "arvados#containerList", "items": [{"uuid": "zhome-xvhdp-cr5queuedcontnr", "command": ["abc"]}]}`))
	})).Close()
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s&select=%s",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID)),
		url.QueryEscape(`["uuid", "command"]`)),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var cn arvados.ContainerList
	c.Check(json.NewDecoder(resp.Body).Decode(&cn), check.IsNil)
	c.Check(cn.Items, check.HasLen, 2)
	mp := make(map[string]arvados.Container)
	for _, cr := range cn.Items {
		mp[cr.UUID] = cr
	}
	c.Check(mp[arvadostest.QueuedContainerUUID].Command, check.DeepEquals, []string{"echo", "hello"})
	c.Check(mp[arvadostest.QueuedContainerUUID].ContainerImage, check.Equals, "")
	c.Check(mp["zhome-xvhdp-cr5queuedcontnr"].Command, check.DeepEquals, []string{"abc"})
	c.Check(mp["zhome-xvhdp-cr5queuedcontnr"].ContainerImage, check.Equals, "")
}

func (s *FederationSuite) TestListMultiRemoteContainerError(c *check.C) {
	defer s.localServiceReturns404(c).Close()
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s&select=%s",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID)),
		url.QueryEscape(`["uuid", "command"]`)),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusBadGateway)
	s.checkJSONErrorMatches(c, resp, `error fetching from zhome \(404 Not Found\): EOF`)
}

func (s *FederationSuite) TestListMultiRemoteContainersPaged(c *check.C) {

	callCount := 0
	defer s.localServiceHandler(c, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		bd, _ := ioutil.ReadAll(req.Body)
		if callCount == 0 {
			c.Check(string(bd), check.Equals, `_method=GET&count=none&filters=%5B%5B%22uuid%22%2C+%22in%22%2C+%5B%22zhome-xvhdp-cr5queuedcontnr%22%2C%22zhome-xvhdp-cr6queuedcontnr%22%5D%5D%5D`)
			w.WriteHeader(200)
			w.Write([]byte(`{"kind": "arvados#containerList", "items": [{"uuid": "zhome-xvhdp-cr5queuedcontnr", "command": ["abc"]}]}`))
		} else if callCount == 1 {
			c.Check(string(bd), check.Equals, `_method=GET&count=none&filters=%5B%5B%22uuid%22%2C+%22in%22%2C+%5B%22zhome-xvhdp-cr6queuedcontnr%22%5D%5D%5D`)
			w.WriteHeader(200)
			w.Write([]byte(`{"kind": "arvados#containerList", "items": [{"uuid": "zhome-xvhdp-cr6queuedcontnr", "command": ["efg"]}]}`))
		}
		callCount++
	})).Close()
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr", "zhome-xvhdp-cr6queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID))),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	c.Check(callCount, check.Equals, 2)
	var cn arvados.ContainerList
	c.Check(json.NewDecoder(resp.Body).Decode(&cn), check.IsNil)
	c.Check(cn.Items, check.HasLen, 3)
	mp := make(map[string]arvados.Container)
	for _, cr := range cn.Items {
		mp[cr.UUID] = cr
	}
	c.Check(mp[arvadostest.QueuedContainerUUID].Command, check.DeepEquals, []string{"echo", "hello"})
	c.Check(mp["zhome-xvhdp-cr5queuedcontnr"].Command, check.DeepEquals, []string{"abc"})
	c.Check(mp["zhome-xvhdp-cr6queuedcontnr"].Command, check.DeepEquals, []string{"efg"})
}

func (s *FederationSuite) TestListMultiRemoteContainersMissing(c *check.C) {

	callCount := 0
	defer s.localServiceHandler(c, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		bd, _ := ioutil.ReadAll(req.Body)
		if callCount == 0 {
			c.Check(string(bd), check.Equals, `_method=GET&count=none&filters=%5B%5B%22uuid%22%2C+%22in%22%2C+%5B%22zhome-xvhdp-cr5queuedcontnr%22%2C%22zhome-xvhdp-cr6queuedcontnr%22%5D%5D%5D`)
			w.WriteHeader(200)
			w.Write([]byte(`{"kind": "arvados#containerList", "items": [{"uuid": "zhome-xvhdp-cr6queuedcontnr", "command": ["efg"]}]}`))
		} else if callCount == 1 {
			c.Check(string(bd), check.Equals, `_method=GET&count=none&filters=%5B%5B%22uuid%22%2C+%22in%22%2C+%5B%22zhome-xvhdp-cr5queuedcontnr%22%5D%5D%5D`)
			w.WriteHeader(200)
			w.Write([]byte(`{"kind": "arvados#containerList", "items": []}`))
		}
		callCount++
	})).Close()
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr", "zhome-xvhdp-cr6queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID))),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	c.Check(callCount, check.Equals, 2)
	var cn arvados.ContainerList
	c.Check(json.NewDecoder(resp.Body).Decode(&cn), check.IsNil)
	c.Check(cn.Items, check.HasLen, 2)
	mp := make(map[string]arvados.Container)
	for _, cr := range cn.Items {
		mp[cr.UUID] = cr
	}
	c.Check(mp[arvadostest.QueuedContainerUUID].Command, check.DeepEquals, []string{"echo", "hello"})
	c.Check(mp["zhome-xvhdp-cr6queuedcontnr"].Command, check.DeepEquals, []string{"efg"})
}

func (s *FederationSuite) TestListMultiRemoteContainerPageSizeError(c *check.C) {
	s.testHandler.Cluster.API.MaxItemsPerResponse = 1
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID))),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusBadRequest)
	s.checkJSONErrorMatches(c, resp, `Federated multi-object request for 2 objects which is more than max page size 1.`)
}

func (s *FederationSuite) TestListMultiRemoteContainerLimitError(c *check.C) {
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s&limit=1",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID))),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusBadRequest)
	s.checkJSONErrorMatches(c, resp, `Federated multi-object may not provide 'limit', 'offset' or 'order'.`)
}

func (s *FederationSuite) TestListMultiRemoteContainerOffsetError(c *check.C) {
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s&offset=1",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID))),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusBadRequest)
	s.checkJSONErrorMatches(c, resp, `Federated multi-object may not provide 'limit', 'offset' or 'order'.`)
}

func (s *FederationSuite) TestListMultiRemoteContainerOrderError(c *check.C) {
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s&order=uuid",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID))),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusBadRequest)
	s.checkJSONErrorMatches(c, resp, `Federated multi-object may not provide 'limit', 'offset' or 'order'.`)
}

func (s *FederationSuite) TestListMultiRemoteContainerSelectError(c *check.C) {
	req := httptest.NewRequest("GET", fmt.Sprintf("/arvados/v1/containers?count=none&filters=%s&select=%s",
		url.QueryEscape(fmt.Sprintf(`[["uuid", "in", ["%v", "zhome-xvhdp-cr5queuedcontnr"]]]`,
			arvadostest.QueuedContainerUUID)),
		url.QueryEscape(`["command"]`)),
		nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := s.testRequest(req).Result()
	c.Check(resp.StatusCode, check.Equals, http.StatusBadRequest)
	s.checkJSONErrorMatches(c, resp, `Federated multi-object request must include 'uuid' in 'select'`)
}
