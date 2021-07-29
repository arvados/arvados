// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"gopkg.in/check.v1"
)

type AggregatorSuite struct {
	handler *Aggregator
	req     *http.Request
	resp    *httptest.ResponseRecorder
}

// Gocheck boilerplate
var _ = check.Suite(&AggregatorSuite{})

func (s *AggregatorSuite) TestInterface(c *check.C) {
	var _ http.Handler = &Aggregator{}
}

func (s *AggregatorSuite) SetUpTest(c *check.C) {
	s.handler = &Aggregator{Cluster: &arvados.Cluster{
		ManagementToken: arvadostest.ManagementToken,
	}}
	s.req = httptest.NewRequest("GET", "/_health/all", nil)
	s.req.Header.Set("Authorization", "Bearer "+arvadostest.ManagementToken)
	s.resp = httptest.NewRecorder()
}

func (s *AggregatorSuite) TestNoAuth(c *check.C) {
	s.req.Header.Del("Authorization")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkError(c)
	c.Check(s.resp.Code, check.Equals, http.StatusUnauthorized)
}

func (s *AggregatorSuite) TestBadAuth(c *check.C) {
	s.req.Header.Set("Authorization", "xyzzy")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkError(c)
	c.Check(s.resp.Code, check.Equals, http.StatusUnauthorized)
}

func (s *AggregatorSuite) TestNoServicesConfigured(c *check.C) {
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkUnhealthy(c)
}

func (s *AggregatorSuite) stubServer(handler http.Handler) (*httptest.Server, string) {
	srv := httptest.NewServer(handler)
	var port string
	if parts := strings.Split(srv.URL, ":"); len(parts) < 3 {
		panic(srv.URL)
	} else {
		port = parts[len(parts)-1]
	}
	return srv, ":" + port
}

func (s *AggregatorSuite) TestUnhealthy(c *check.C) {
	srv, listen := s.stubServer(&unhealthyHandler{})
	defer srv.Close()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.Keepstore, "http://localhost"+listen+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkUnhealthy(c)
}

func (s *AggregatorSuite) TestHealthy(c *check.C) {
	srv, listen := s.stubServer(&healthyHandler{})
	defer srv.Close()
	s.setAllServiceURLs(listen)
	s.handler.ServeHTTP(s.resp, s.req)
	resp := s.checkOK(c)
	svc := "keepstore+http://localhost" + listen + "/_health/ping"
	c.Logf("%#v", resp)
	ep := resp.Checks[svc]
	c.Check(ep.Health, check.Equals, "OK")
	c.Check(ep.HTTPStatusCode, check.Equals, 200)
}

func (s *AggregatorSuite) TestHealthyAndUnhealthy(c *check.C) {
	srvH, listenH := s.stubServer(&healthyHandler{})
	defer srvH.Close()
	srvU, listenU := s.stubServer(&unhealthyHandler{})
	defer srvU.Close()
	s.setAllServiceURLs(listenH)
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.Keepstore, "http://localhost"+listenH+"/", "http://127.0.0.1"+listenU+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	resp := s.checkUnhealthy(c)
	ep := resp.Checks["keepstore+http://localhost"+listenH+"/_health/ping"]
	c.Check(ep.Health, check.Equals, "OK")
	c.Check(ep.HTTPStatusCode, check.Equals, 200)
	ep = resp.Checks["keepstore+http://127.0.0.1"+listenU+"/_health/ping"]
	c.Check(ep.Health, check.Equals, "ERROR")
	c.Check(ep.HTTPStatusCode, check.Equals, 200)
	c.Logf("%#v", ep)
}

func (s *AggregatorSuite) TestPingTimeout(c *check.C) {
	s.handler.timeout = arvados.Duration(100 * time.Millisecond)
	srv, listen := s.stubServer(&slowHandler{})
	defer srv.Close()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.Keepstore, "http://localhost"+listen+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	resp := s.checkUnhealthy(c)
	ep := resp.Checks["keepstore+http://localhost"+listen+"/_health/ping"]
	c.Check(ep.Health, check.Equals, "ERROR")
	c.Check(ep.HTTPStatusCode, check.Equals, 0)
	rt, err := ep.ResponseTime.Float64()
	c.Check(err, check.IsNil)
	c.Check(rt > 0.005, check.Equals, true)
}

func (s *AggregatorSuite) checkError(c *check.C) {
	c.Check(s.resp.Code, check.Not(check.Equals), http.StatusOK)
	var resp ClusterHealthResponse
	err := json.Unmarshal(s.resp.Body.Bytes(), &resp)
	c.Check(err, check.IsNil)
	c.Check(resp.Health, check.Not(check.Equals), "OK")
}

func (s *AggregatorSuite) checkUnhealthy(c *check.C) ClusterHealthResponse {
	return s.checkResult(c, "ERROR")
}

func (s *AggregatorSuite) checkOK(c *check.C) ClusterHealthResponse {
	return s.checkResult(c, "OK")
}

func (s *AggregatorSuite) checkResult(c *check.C, health string) ClusterHealthResponse {
	c.Check(s.resp.Code, check.Equals, http.StatusOK)
	var resp ClusterHealthResponse
	c.Log(s.resp.Body.String())
	err := json.Unmarshal(s.resp.Body.Bytes(), &resp)
	c.Check(err, check.IsNil)
	c.Check(resp.Health, check.Equals, health)
	return resp
}

func (s *AggregatorSuite) setAllServiceURLs(listen string) {
	svcs := &s.handler.Cluster.Services
	for _, svc := range []*arvados.Service{
		&svcs.Controller,
		&svcs.DispatchCloud,
		&svcs.DispatchLSF,
		&svcs.Keepbalance,
		&svcs.Keepproxy,
		&svcs.Keepstore,
		&svcs.Health,
		&svcs.RailsAPI,
		&svcs.WebDAV,
		&svcs.Websocket,
		&svcs.Workbench1,
		&svcs.Workbench2,
	} {
		arvadostest.SetServiceURL(svc, "http://localhost"+listen+"/")
	}
}

type unhealthyHandler struct{}

func (*unhealthyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/_health/ping" {
		resp.Write([]byte(`{"health":"ERROR","error":"the bends"}`))
	} else {
		http.Error(resp, "not found", http.StatusNotFound)
	}
}

type healthyHandler struct{}

func (*healthyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/_health/ping" {
		resp.Write([]byte(`{"health":"OK"}`))
	} else {
		http.Error(resp, "not found", http.StatusNotFound)
	}
}

type slowHandler struct{}

func (*slowHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/_health/ping" {
		time.Sleep(3 * time.Second)
		resp.Write([]byte(`{"health":"OK"}`))
	} else {
		http.Error(resp, "not found", http.StatusNotFound)
	}
}
