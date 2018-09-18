// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

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

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ProxyRemoteSuite{})

type ProxyRemoteSuite struct {
	cluster *arvados.Cluster
	vm      VolumeManager
	rtr     http.Handler

	remoteClusterID      string
	remoteBlobSigningKey []byte
	remoteKeepLocator    string
	remoteKeepData       []byte
	remoteKeepproxy      *httptest.Server
	remoteKeepRequests   int64
	remoteAPI            *httptest.Server
}

func (s *ProxyRemoteSuite) remoteKeepproxyHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *ProxyRemoteSuite) remoteAPIHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *ProxyRemoteSuite) SetUpTest(c *check.C) {
	s.remoteClusterID = "z0000"
	s.remoteBlobSigningKey = []byte("3b6df6fb6518afe12922a5bc8e67bf180a358bc8")
	s.remoteKeepproxy = httptest.NewServer(http.HandlerFunc(s.remoteKeepproxyHandler))
	s.remoteAPI = httptest.NewUnstartedServer(http.HandlerFunc(s.remoteAPIHandler))
	s.remoteAPI.StartTLS()
	s.cluster = arvados.IntegrationTestCluster()
	s.cluster.RemoteClusters = map[string]arvados.RemoteCluster{
		s.remoteClusterID: arvados.RemoteCluster{
			Host:     strings.Split(s.remoteAPI.URL, "//")[1],
			Proxy:    true,
			Scheme:   "http",
			Insecure: true,
		},
	}
	s.vm = MakeTestVolumeManager(2)
	KeepVM = s.vm
	theConfig = DefaultConfig()
	theConfig.systemAuthToken = arvadostest.DataManagerToken
	theConfig.Start()
	s.rtr = MakeRESTRouter(s.cluster)
}

func (s *ProxyRemoteSuite) TearDownTest(c *check.C) {
	s.vm.Close()
	KeepVM = nil
	theConfig = DefaultConfig()
	theConfig.Start()
	s.remoteAPI.Close()
	s.remoteKeepproxy.Close()
}

func (s *ProxyRemoteSuite) TestProxyRemote(c *check.C) {
	data := []byte("foo bar")
	s.remoteKeepData = data
	locator := fmt.Sprintf("%x+%d", md5.Sum(data), len(data))
	s.remoteKeepLocator = keepclient.SignLocator(locator, arvadostest.ActiveTokenV2, time.Now().Add(time.Minute), time.Minute, s.remoteBlobSigningKey)

	path := "/" + strings.Replace(s.remoteKeepLocator, "+A", "+R"+s.remoteClusterID+"-", 1)

	var req *http.Request
	var resp *httptest.ResponseRecorder
	tryWithToken := func(token string) {
		req = httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp = httptest.NewRecorder()
		s.rtr.ServeHTTP(resp, req)
	}

	// Happy path
	tryWithToken(arvadostest.ActiveTokenV2)
	c.Check(s.remoteKeepRequests, check.Equals, int64(1))
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, string(data))

	// Obsolete token
	tryWithToken(arvadostest.ActiveToken)
	c.Check(s.remoteKeepRequests, check.Equals, int64(1))
	c.Check(resp.Code, check.Equals, http.StatusBadRequest)
	c.Check(resp.Body.String(), check.Not(check.Equals), string(data))

	// Bad token
	tryWithToken(arvadostest.ActiveTokenV2[:len(arvadostest.ActiveTokenV2)-3] + "xxx")
	c.Check(s.remoteKeepRequests, check.Equals, int64(2))
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	c.Check(resp.Body.String(), check.Not(check.Equals), string(data))
}
