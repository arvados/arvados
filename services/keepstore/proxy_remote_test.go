// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&proxyRemoteSuite{})

type proxyRemoteSuite struct {
	cluster *arvados.Cluster
	handler *router

	remoteClusterID      string
	remoteBlobSigningKey []byte
	remoteKeepLocator    string
	remoteKeepData       []byte
	remoteKeepproxy      *httptest.Server
	remoteKeepRequests   int64
	remoteAPI            *httptest.Server
}

func (s *proxyRemoteSuite) remoteKeepproxyHandler(w http.ResponseWriter, r *http.Request) {
	expectToken, err := auth.SaltToken(arvadostest.ActiveTokenV2, s.remoteClusterID)
	if err != nil {
		panic(err)
	}
	atomic.AddInt64(&s.remoteKeepRequests, 1)
	var token string
	if auth := strings.Split(r.Header.Get("Authorization"), " "); len(auth) == 2 && (auth[0] == "OAuth2" || auth[0] == "Bearer") {
		token = auth[1]
	}
	if r.Method == "GET" && r.URL.Path == "/"+s.remoteKeepLocator && token == expectToken {
		w.Write(s.remoteKeepData)
		return
	}
	http.Error(w, "404", 404)
}

func (s *proxyRemoteSuite) remoteAPIHandler(w http.ResponseWriter, r *http.Request) {
	host, port, _ := net.SplitHostPort(strings.Split(s.remoteKeepproxy.URL, "//")[1])
	portnum, _ := strconv.Atoi(port)
	if r.URL.Path == "/arvados/v1/discovery/v1/rest" {
		json.NewEncoder(w).Encode(arvados.DiscoveryDocument{})
		return
	}
	if r.URL.Path == "/arvados/v1/keep_services/accessible" {
		json.NewEncoder(w).Encode(arvados.KeepServiceList{
			Items: []arvados.KeepService{
				{
					UUID:           s.remoteClusterID + "-bi6l4-proxyproxyproxy",
					ServiceType:    "proxy",
					ServiceHost:    host,
					ServicePort:    portnum,
					ServiceSSLFlag: false,
				},
			},
		})
		return
	}
	http.Error(w, "404", 404)
}

func (s *proxyRemoteSuite) SetUpTest(c *check.C) {
	s.remoteClusterID = "z0000"
	s.remoteBlobSigningKey = []byte("3b6df6fb6518afe12922a5bc8e67bf180a358bc8")
	s.remoteKeepproxy = httptest.NewServer(httpserver.LogRequests(http.HandlerFunc(s.remoteKeepproxyHandler)))
	s.remoteAPI = httptest.NewUnstartedServer(http.HandlerFunc(s.remoteAPIHandler))
	s.remoteAPI.StartTLS()
	s.cluster = testCluster(c)
	s.cluster.RemoteClusters = map[string]arvados.RemoteCluster{
		s.remoteClusterID: {
			Host:     strings.Split(s.remoteAPI.URL, "//")[1],
			Proxy:    true,
			Scheme:   "http",
			Insecure: true,
		},
	}
	s.cluster.Volumes = map[string]arvados.Volume{"zzzzz-nyw5e-000000000000000": {Driver: "stub"}}
}

func (s *proxyRemoteSuite) TearDownTest(c *check.C) {
	s.remoteAPI.Close()
	s.remoteKeepproxy.Close()
}

func (s *proxyRemoteSuite) TestProxyRemote(c *check.C) {
	reg := prometheus.NewRegistry()
	router, cancel := testRouter(c, s.cluster, reg)
	defer cancel()
	instrumented := httpserver.Instrument(reg, ctxlog.TestLogger(c), router)
	handler := httpserver.LogRequests(instrumented.ServeAPI(s.cluster.ManagementToken, instrumented))

	data := []byte("foo bar")
	s.remoteKeepData = data
	locator := fmt.Sprintf("%x+%d", md5.Sum(data), len(data))
	s.remoteKeepLocator = keepclient.SignLocator(locator, arvadostest.ActiveTokenV2, time.Now().Add(time.Minute), time.Minute, s.remoteBlobSigningKey)

	path := "/" + strings.Replace(s.remoteKeepLocator, "+A", "+R"+s.remoteClusterID+"-", 1)

	for _, trial := range []struct {
		label            string
		method           string
		token            string
		xKeepSignature   string
		expectRemoteReqs int64
		expectCode       int
		expectSignature  bool
	}{
		{
			label:            "GET only",
			method:           "GET",
			token:            arvadostest.ActiveTokenV2,
			expectRemoteReqs: 1,
			expectCode:       http.StatusOK,
		},
		{
			label:            "obsolete token",
			method:           "GET",
			token:            arvadostest.ActiveToken,
			expectRemoteReqs: 0,
			expectCode:       http.StatusBadRequest,
		},
		{
			label:            "bad token",
			method:           "GET",
			token:            arvadostest.ActiveTokenV2[:len(arvadostest.ActiveTokenV2)-3] + "xxx",
			expectRemoteReqs: 1,
			expectCode:       http.StatusNotFound,
		},
		{
			label:            "HEAD only",
			method:           "HEAD",
			token:            arvadostest.ActiveTokenV2,
			expectRemoteReqs: 1,
			expectCode:       http.StatusOK,
		},
		{
			label:            "HEAD with local signature",
			method:           "HEAD",
			xKeepSignature:   "local, time=" + time.Now().Format(time.RFC3339),
			token:            arvadostest.ActiveTokenV2,
			expectRemoteReqs: 1,
			expectCode:       http.StatusOK,
			expectSignature:  true,
		},
		{
			label:            "GET with local signature",
			method:           "GET",
			xKeepSignature:   "local, time=" + time.Now().Format(time.RFC3339),
			token:            arvadostest.ActiveTokenV2,
			expectRemoteReqs: 1,
			expectCode:       http.StatusOK,
			expectSignature:  true,
		},
	} {
		c.Logf("=== trial: %s", trial.label)

		s.remoteKeepRequests = 0

		var req *http.Request
		var resp *httptest.ResponseRecorder
		req = httptest.NewRequest(trial.method, path, nil)
		req.Header.Set("Authorization", "Bearer "+trial.token)
		if trial.xKeepSignature != "" {
			req.Header.Set("X-Keep-Signature", trial.xKeepSignature)
		}
		resp = httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		c.Check(s.remoteKeepRequests, check.Equals, trial.expectRemoteReqs)
		if !c.Check(resp.Code, check.Equals, trial.expectCode) {
			c.Logf("resp.Code %d came with resp.Body %q", resp.Code, resp.Body.String())
		}
		if resp.Code == http.StatusOK {
			if trial.method == "HEAD" {
				c.Check(resp.Body.String(), check.Equals, "")
				c.Check(resp.Result().ContentLength, check.Equals, int64(len(data)))
			} else {
				c.Check(resp.Body.String(), check.Equals, string(data))
			}
		} else {
			c.Check(resp.Body.String(), check.Not(check.Equals), string(data))
		}

		c.Check(resp.Header().Get("Vary"), check.Matches, `(.*, )?X-Keep-Signature(, .*)?`)

		locHdr := resp.Header().Get("X-Keep-Locator")
		if !trial.expectSignature {
			c.Check(locHdr, check.Equals, "")
			continue
		}

		c.Check(locHdr, check.Not(check.Equals), "")
		c.Check(locHdr, check.Not(check.Matches), `.*\+R.*`)
		c.Check(arvados.VerifySignature(locHdr, trial.token, s.cluster.Collections.BlobSigningTTL.Duration(), []byte(s.cluster.Collections.BlobSigningKey)), check.IsNil)

		// Ensure block can be requested using new signature
		req = httptest.NewRequest("GET", "/"+locHdr, nil)
		req.Header.Set("Authorization", "Bearer "+trial.token)
		resp = httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		c.Check(s.remoteKeepRequests, check.Equals, trial.expectRemoteReqs)
	}
}
