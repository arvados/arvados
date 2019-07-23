// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/config"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&serverSuite{})

type serverSuite struct {
	cluster *arvados.Cluster
	srv     *server
	wg      sync.WaitGroup
}

func (s *serverSuite) SetUpTest(c *check.C) {
	var err error
	s.cluster, err = s.testConfig()
	c.Assert(err, check.IsNil)
	s.srv = &server{cluster: s.cluster}
}

func (*serverSuite) testConfig() (*arvados.Cluster, error) {
	ldr := config.NewLoader(nil, nil)
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

// TestBadDB ensures Run() returns an error (instead of panicking or
// deadlocking) if it can't connect to the database server at startup.
func (s *serverSuite) TestBadDB(c *check.C) {
	s.cluster.PostgreSQL.Connection["password"] = "1234"

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := s.srv.Run()
		c.Check(err, check.NotNil)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		s.srv.WaitReady()
		wg.Done()
	}()

	done := make(chan bool)
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		c.Fatal("timeout")
	}
}

func (s *serverSuite) TestHealth(c *check.C) {
	go s.srv.Run()
	defer s.srv.Close()
	s.srv.WaitReady()
	for _, token := range []string{"", "foo", s.cluster.ManagementToken} {
		req, err := http.NewRequest("GET", "http://"+s.srv.listener.Addr().String()+"/_health/ping", nil)
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

func (s *serverSuite) TestStatus(c *check.C) {
	go s.srv.Run()
	defer s.srv.Close()
	s.srv.WaitReady()
	req, err := http.NewRequest("GET", "http://"+s.srv.listener.Addr().String()+"/status.json", nil)
	c.Assert(err, check.IsNil)
	resp, err := http.DefaultClient.Do(req)
	c.Check(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	c.Check(err, check.IsNil)
	c.Check(status["Version"], check.Not(check.Equals), "")
}

func (s *serverSuite) TestHealthDisabled(c *check.C) {
	s.cluster.ManagementToken = ""

	go s.srv.Run()
	defer s.srv.Close()
	s.srv.WaitReady()

	for _, token := range []string{"", "foo", arvadostest.ManagementToken} {
		req, err := http.NewRequest("GET", "http://"+s.srv.listener.Addr().String()+"/_health/ping", nil)
		c.Assert(err, check.IsNil)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		c.Check(err, check.IsNil)
		c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	}
}

func (s *serverSuite) TestLoadLegacyConfig(c *check.C) {
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
		log.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)

	}
	cluster := configure(logger(nil), []string{"arvados-ws", "-config", tmpfile.Name()})
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
	c.Check(cluster.Services.Websocket.InternalURLs, check.DeepEquals, map[arvados.URL]arvados.ServiceInstance{
		arvados.URL{Host: ":8765"}: arvados.ServiceInstance{}})
	c.Check(cluster.SystemLogs.LogLevel, check.Equals, "debug")
	c.Check(cluster.SystemLogs.Format, check.Equals, "text")
	c.Check(cluster.API.SendTimeout, check.Equals, arvados.Duration(61*time.Second))
	c.Check(cluster.API.WebsocketClientEventQueue, check.Equals, 62)
	c.Check(cluster.API.WebsocketServerEventQueue, check.Equals, 5)
	c.Check(cluster.ManagementToken, check.Equals, "qqqqq")
}
