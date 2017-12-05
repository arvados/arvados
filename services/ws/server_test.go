// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&serverSuite{})

type serverSuite struct {
	cfg *wsConfig
	srv *server
	wg  sync.WaitGroup
}

func (s *serverSuite) SetUpTest(c *check.C) {
	s.cfg = s.testConfig()
	s.srv = &server{wsConfig: s.cfg}
}

func (*serverSuite) testConfig() *wsConfig {
	cfg := defaultConfig()
	cfg.Client = *(arvados.NewClientFromEnv())
	cfg.Postgres = testDBConfig()
	cfg.Listen = ":"
	cfg.ManagementToken = arvadostest.ManagementToken
	return &cfg
}

// TestBadDB ensures Run() returns an error (instead of panicking or
// deadlocking) if it can't connect to the database server at startup.
func (s *serverSuite) TestBadDB(c *check.C) {
	s.cfg.Postgres["password"] = "1234"

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
	for _, token := range []string{"", "foo", s.cfg.ManagementToken} {
		req, err := http.NewRequest("GET", "http://"+s.srv.listener.Addr().String()+"/_health/ping", nil)
		c.Assert(err, check.IsNil)
		if token != "" {
			req.Header.Add("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		c.Check(err, check.IsNil)
		if token == s.cfg.ManagementToken {
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
	s.cfg.ManagementToken = ""

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
