// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
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
	ldr := config.NewLoader(bytes.NewBufferString(`Clusters: {zzzzz: {}}`), ctxlog.TestLogger(c))
	ldr.Path = "-"
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	cluster.ManagementToken = arvadostest.ManagementToken
	cluster.SystemRootToken = arvadostest.SystemRootToken
	cluster.Collections.BlobSigningKey = arvadostest.BlobSigningKey
	cluster.Volumes["z"] = arvados.Volume{StorageClasses: map[string]bool{"default": true}}
	cluster.Containers.LocalKeepBlobBuffersPerVCPU = 0
	s.handler = &Aggregator{Cluster: cluster}
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

// If an InternalURL host is 0.0.0.0, localhost, 127/8, or ::1 and
// nothing is listening there, don't fail the health check -- instead,
// assume the relevant component just isn't installed/enabled on this
// node, but does work when contacted through ExternalURL.
func (s *AggregatorSuite) TestUnreachableLoopbackPort(c *check.C) {
	srvH, listenH := s.stubServer(&healthyHandler{})
	defer srvH.Close()
	s.setAllServiceURLs(listenH)
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.Keepproxy, "http://localhost:9/")
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.Workbench1, "http://0.0.0.0:9/")
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.Keepbalance, "http://127.0.0.127:9/")
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.WebDAV, "http://[::1]:9/")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkOK(c)

	// If a non-loopback address is unreachable, that's still a
	// fail.
	s.resp = httptest.NewRecorder()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.WebDAV, "http://172.31.255.254:9/")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkUnhealthy(c)
}

func (s *AggregatorSuite) TestIsLocalHost(c *check.C) {
	c.Check(isLocalHost("Localhost"), check.Equals, true)
	c.Check(isLocalHost("localhost"), check.Equals, true)
	c.Check(isLocalHost("127.0.0.1"), check.Equals, true)
	c.Check(isLocalHost("127.0.0.127"), check.Equals, true)
	c.Check(isLocalHost("127.1.2.7"), check.Equals, true)
	c.Check(isLocalHost("0.0.0.0"), check.Equals, true)
	c.Check(isLocalHost("::1"), check.Equals, true)
	c.Check(isLocalHost("1.2.3.4"), check.Equals, false)
	c.Check(isLocalHost("1::1"), check.Equals, false)
	c.Check(isLocalHost("example.com"), check.Equals, false)
	c.Check(isLocalHost("127.0.0"), check.Equals, false)
	c.Check(isLocalHost(""), check.Equals, false)
}

func (s *AggregatorSuite) TestConfigMismatch(c *check.C) {
	// time1/hash1: current config
	time1 := time.Now().Add(time.Second - time.Minute - time.Hour)
	hash1 := fmt.Sprintf("%x", sha256.Sum256([]byte(`Clusters: {zzzzz: {SystemRootToken: xyzzy}}`)))
	// time2/hash2: old config
	time2 := time1.Add(-time.Hour)
	hash2 := fmt.Sprintf("%x", sha256.Sum256([]byte(`Clusters: {zzzzz: {SystemRootToken: old-token}}`)))

	// srv1: current file
	handler1 := healthyHandler{configHash: hash1, configTime: time1}
	srv1, listen1 := s.stubServer(&handler1)
	defer srv1.Close()
	// srv2: old file, current content
	handler2 := healthyHandler{configHash: hash1, configTime: time2}
	srv2, listen2 := s.stubServer(&handler2)
	defer srv2.Close()
	// srv3: old file, old content
	handler3 := healthyHandler{configHash: hash2, configTime: time2}
	srv3, listen3 := s.stubServer(&handler3)
	defer srv3.Close()
	// srv4: no metrics handler
	handler4 := healthyHandler{}
	srv4, listen4 := s.stubServer(&handler4)
	defer srv4.Close()

	s.setAllServiceURLs(listen1)

	// listen2 => old timestamp, same content => no problem
	s.resp = httptest.NewRecorder()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.DispatchCloud,
		"http://localhost"+listen2+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	resp := s.checkOK(c)

	// listen4 => no metrics on some services => no problem
	s.resp = httptest.NewRecorder()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.WebDAV,
		"http://localhost"+listen4+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	resp = s.checkOK(c)

	// listen3 => old timestamp, old content => report discrepancy
	s.resp = httptest.NewRecorder()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.Keepstore,
		"http://localhost"+listen1+"/",
		"http://localhost"+listen3+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	resp = s.checkUnhealthy(c)
	if c.Check(len(resp.Errors) > 0, check.Equals, true) {
		c.Check(resp.Errors[0], check.Matches, `outdated config: \Qkeepstore+http://localhost`+listen3+`\E: config file \(sha256 .*\) does not match latest version with timestamp .*`)
	}

	// no services report config time (migrating to current version) => no problem
	s.resp = httptest.NewRecorder()
	s.setAllServiceURLs(listen4)
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkOK(c)
}

func (s *AggregatorSuite) TestClockSkew(c *check.C) {
	// srv1: report real wall clock time
	handler1 := healthyHandler{}
	srv1, listen1 := s.stubServer(&handler1)
	defer srv1.Close()
	// srv2: report near-future time
	handler2 := healthyHandler{headerDate: time.Now().Add(3 * time.Second)}
	srv2, listen2 := s.stubServer(&handler2)
	defer srv2.Close()
	// srv3: report far-future time
	handler3 := healthyHandler{headerDate: time.Now().Add(3*time.Minute + 3*time.Second)}
	srv3, listen3 := s.stubServer(&handler3)
	defer srv3.Close()

	s.setAllServiceURLs(listen1)

	// near-future time => OK
	s.resp = httptest.NewRecorder()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.DispatchCloud,
		"http://localhost"+listen2+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkOK(c)

	// far-future time => error
	s.resp = httptest.NewRecorder()
	arvadostest.SetServiceURL(&s.handler.Cluster.Services.WebDAV,
		"http://localhost"+listen3+"/")
	s.handler.ServeHTTP(s.resp, s.req)
	resp := s.checkUnhealthy(c)
	if c.Check(len(resp.Errors) > 0, check.Equals, true) {
		c.Check(resp.Errors[0], check.Matches, `clock skew detected: maximum timestamp spread is 3m.* \(exceeds warning threshold of 1m\)`)
	}
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

func (s *AggregatorSuite) TestCheckCommand(c *check.C) {
	srv, listen := s.stubServer(&healthyHandler{})
	defer srv.Close()
	s.setAllServiceURLs(listen)
	tmpdir := c.MkDir()
	confdata, err := yaml.Marshal(arvados.Config{Clusters: map[string]arvados.Cluster{s.handler.Cluster.ClusterID: *s.handler.Cluster}})
	c.Assert(err, check.IsNil)
	confdata = regexp.MustCompile(`Source(Timestamp|SHA256): [^\n]+\n`).ReplaceAll(confdata, []byte{})
	err = ioutil.WriteFile(tmpdir+"/config.yml", confdata, 0777)
	c.Assert(err, check.IsNil)

	var stdout, stderr bytes.Buffer

	exitcode := CheckCommand.RunCommand("check", []string{"-config=" + tmpdir + "/config.yml"}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 0)
	c.Check(stderr.String(), check.Equals, "")
	c.Check(stdout.String(), check.Equals, "")

	stdout.Reset()
	stderr.Reset()
	exitcode = CheckCommand.RunCommand("check", []string{"-config=" + tmpdir + "/config.yml", "-yaml"}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 0)
	c.Check(stderr.String(), check.Equals, "")
	c.Check(stdout.String(), check.Matches, `(?ms).*(\n|^)health: OK\n.*`)
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
		&svcs.DispatchSLURM,
		&svcs.GitHTTP,
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

type healthyHandler struct {
	configHash string
	configTime time.Time
	headerDate time.Time
}

func (h *healthyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if !h.headerDate.IsZero() {
		resp.Header().Set("Date", h.headerDate.Format(time.RFC1123))
	}
	authOK := req.Header.Get("Authorization") == "Bearer "+arvadostest.ManagementToken
	if req.URL.Path == "/_health/ping" {
		if !authOK {
			http.Error(resp, "unauthorized", http.StatusUnauthorized)
			return
		}
		resp.Write([]byte(`{"health":"OK"}`))
	} else if req.URL.Path == "/metrics" {
		if !authOK {
			http.Error(resp, "unauthorized", http.StatusUnauthorized)
			return
		}
		t := h.configTime
		if t.IsZero() {
			t = time.Now()
		}
		fmt.Fprintf(resp, `# HELP arvados_config_load_timestamp_seconds Time when config file was loaded.
# TYPE arvados_config_load_timestamp_seconds gauge
arvados_config_load_timestamp_seconds{sha256="%s"} %g
# HELP arvados_config_source_timestamp_seconds Timestamp of config file when it was loaded.
# TYPE arvados_config_source_timestamp_seconds gauge
arvados_config_source_timestamp_seconds{sha256="%s"} %g
`,
			h.configHash, float64(time.Now().UnixNano())/1e9,
			h.configHash, float64(t.UnixNano())/1e9)
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
