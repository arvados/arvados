// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&serviceSuite{})

type serviceSuite struct {
	handler service.Handler
	srv     *httptest.Server
	cluster *arvados.Cluster
	wg      sync.WaitGroup
}

func (s *serviceSuite) SetUpTest(c *check.C) {
	var err error
	s.cluster, err = s.testConfig(c)
	c.Assert(err, check.IsNil)
}

func (s *serviceSuite) start() {
	s.handler = newHandler(context.Background(), s.cluster, "", prometheus.NewRegistry())
	s.srv = httptest.NewServer(s.handler)
}

func (s *serviceSuite) TearDownTest(c *check.C) {
	if s.srv != nil {
		s.srv.Close()
	}
}

func (*serviceSuite) testConfig(c *check.C) (*arvados.Cluster, error) {
	ldr := config.NewLoader(nil, ctxlog.TestLogger(c))
	cfg, err := ldr.Load()
	if err != nil {
		return nil, err
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return nil, err
	}
	client := arvados.NewClientFromEnv()
	cluster.Services.Controller.ExternalURL.Host = client.APIHost
	cluster.SystemRootToken = client.AuthToken
	cluster.TLS.Insecure = client.Insecure
	cluster.PostgreSQL.Connection = testDBConfig()
	cluster.Services.Websocket.InternalURLs = map[arvados.URL]arvados.ServiceInstance{arvados.URL{Host: ":"}: arvados.ServiceInstance{}}
	cluster.ManagementToken = arvadostest.ManagementToken
	return cluster, nil
}

// TestBadDB ensures the server returns an error (instead of panicking
// or deadlocking) if it can't connect to the database server at
// startup.
func (s *serviceSuite) TestBadDB(c *check.C) {
	s.cluster.PostgreSQL.Connection["password"] = "1234"
	s.start()
	resp, err := http.Get(s.srv.URL)
	c.Check(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusInternalServerError)
	c.Check(s.handler.CheckHealth(), check.ErrorMatches, "database not connected")
	c.Check(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusInternalServerError)
}

func (s *serviceSuite) TestHealth(c *check.C) {
	s.start()
	for _, token := range []string{"", "foo", s.cluster.ManagementToken} {
		req, err := http.NewRequest("GET", s.srv.URL+"/_health/ping", nil)
		c.Assert(err, check.IsNil)
		if token != "" {
			req.Header.Add("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		c.Check(err, check.IsNil)
		if token == s.cluster.ManagementToken {
			c.Check(resp.StatusCode, check.Equals, http.StatusOK)
			buf, err := ioutil.ReadAll(resp.Body)
			c.Check(err, check.IsNil)
			c.Check(string(buf), check.Equals, `{"health":"OK"}`+"\n")
		} else {
			c.Check(resp.StatusCode, check.Not(check.Equals), http.StatusOK)
		}
	}
}

func (s *serviceSuite) TestStatus(c *check.C) {
	s.start()
	req, err := http.NewRequest("GET", s.srv.URL+"/status.json", nil)
	c.Assert(err, check.IsNil)
	resp, err := http.DefaultClient.Do(req)
	c.Check(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	c.Check(err, check.IsNil)
	c.Check(status["Version"], check.Not(check.Equals), "")
}

func (s *serviceSuite) TestHealthDisabled(c *check.C) {
	s.cluster.ManagementToken = ""
	s.start()
	for _, token := range []string{"", "foo", arvadostest.ManagementToken} {
		req, err := http.NewRequest("GET", s.srv.URL+"/_health/ping", nil)
		c.Assert(err, check.IsNil)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		c.Check(err, check.IsNil)
		c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	}
}

func (s *serviceSuite) TestLoadLegacyConfig(c *check.C) {
	content := []byte(`
Client:
  APIHost: example.com
  AuthToken: abcdefg
Postgres:
  "dbname": "arvados_production"
  "user": "arvados"
  "password": "xyzzy"
  "host": "localhost"
  "connect_timeout": "30"
  "sslmode": "require"
  "fallback_application_name": "arvados-ws"
PostgresPool: 63
Listen: ":8765"
LogLevel: "debug"
LogFormat: "text"
PingTimeout: 61s
ClientEventQueue: 62
ServerEventQueue:  5
ManagementToken: qqqqq
`)
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		c.Error(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		c.Error(err)
	}
	if err := tmpfile.Close(); err != nil {
		c.Error(err)

	}
	ldr := config.NewLoader(&bytes.Buffer{}, logrus.New())
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	ldr.SetupFlags(flagset)
	flagset.Parse(ldr.MungeLegacyConfigArgs(ctxlog.TestLogger(c), []string{"-config", tmpfile.Name()}, "-legacy-ws-config"))
	cfg, err := ldr.Load()
	c.Check(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Check(err, check.IsNil)
	c.Check(cluster, check.NotNil)

	c.Check(cluster.Services.Controller.ExternalURL, check.Equals, arvados.URL{Scheme: "https", Host: "example.com"})
	c.Check(cluster.SystemRootToken, check.Equals, "abcdefg")

	c.Check(cluster.PostgreSQL.Connection, check.DeepEquals, arvados.PostgreSQLConnection{
		"connect_timeout":           "30",
		"dbname":                    "arvados_production",
		"fallback_application_name": "arvados-ws",
		"host":                      "localhost",
		"password":                  "xyzzy",
		"sslmode":                   "require",
		"user":                      "arvados"})
	c.Check(cluster.PostgreSQL.ConnectionPool, check.Equals, 63)
	c.Check(cluster.Services.Websocket.InternalURLs[arvados.URL{Host: ":8765"}], check.NotNil)
	c.Check(cluster.SystemLogs.LogLevel, check.Equals, "debug")
	c.Check(cluster.SystemLogs.Format, check.Equals, "text")
	c.Check(cluster.API.SendTimeout, check.Equals, arvados.Duration(61*time.Second))
	c.Check(cluster.API.WebsocketClientEventQueue, check.Equals, 62)
	c.Check(cluster.API.WebsocketServerEventQueue, check.Equals, 5)
	c.Check(cluster.ManagementToken, check.Equals, "qqqqq")
}
