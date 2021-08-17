// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&mainSuite{})

type mainSuite struct{}

func (s *mainSuite) TestVersionFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	runCommand("keep-balance", []string{"-version"}, nil, &stdout, &stderr)
	c.Check(stderr.String(), check.Equals, "")
	c.Log(stdout.String())
}

func (s *mainSuite) TestHTTPServer(c *check.C) {
	arvadostest.StartKeep(2, true)

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		c.Fatal(err)
	}
	_, p, err := net.SplitHostPort(ln.Addr().String())
	c.Check(err, check.IsNil)
	ln.Close()
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	cluster.Services.Keepbalance.InternalURLs[arvados.URL{Host: "localhost:" + p, Path: "/"}] = arvados.ServiceInstance{}
	cfg.Clusters[cluster.ClusterID] = *cluster
	config, err := yaml.Marshal(cfg)
	c.Assert(err, check.IsNil)

	var stdout bytes.Buffer
	go runCommand("keep-balance", []string{"-config", "-"}, bytes.NewBuffer(config), &stdout, &stdout)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			time.Sleep(time.Second / 10)
			req, err := http.NewRequest(http.MethodGet, "http://:"+p+"/metrics", nil)
			if err != nil {
				c.Fatal(err)
				return
			}
			req.Header.Set("Authorization", "Bearer "+cluster.ManagementToken)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				c.Logf("error %s", err)
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				c.Logf("http status %d", resp.StatusCode)
				continue
			}
			buf, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				c.Logf("read body: %s", err)
				continue
			}
			c.Check(string(buf), check.Matches, `(?ms).*arvados_keepbalance_sweep_seconds_sum.*`)
			return
		}
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Log(stdout.String())
		c.Fatal("timeout")
	}
	c.Log(stdout.String())

	// Check non-metrics URL that gets passed through to us from
	// service.Command
	req, err := http.NewRequest(http.MethodGet, "http://:"+p+"/not-metrics", nil)
	c.Assert(err, check.IsNil)
	resp, err := http.DefaultClient.Do(req)
	c.Check(err, check.IsNil)
	defer resp.Body.Close()
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
}
