// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&railsRestartSuite{})

type railsRestartSuite struct{}

// This tests RailsAPI, not controller -- but tests RailsAPI's
// integration with passenger, so it needs to run against the
// run-tests.sh environment where RailsAPI runs under passenger, not
// in the Rails test environment.
func (s *railsRestartSuite) TestConfigReload(c *check.C) {
	hc := http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}

	confdata, err := os.ReadFile(os.Getenv("ARVADOS_CONFIG"))
	c.Assert(err, check.IsNil)
	oldhash := fmt.Sprintf("%x", sha256.Sum256(confdata))
	c.Logf("oldhash %s", oldhash)

	ldr := config.NewLoader(&bytes.Buffer{}, ctxlog.TestLogger(c))
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cc, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	var metricsURL string
	for u := range cc.Services.RailsAPI.InternalURLs {
		u := url.URL(u)
		mu, err := u.Parse("/metrics")
		c.Assert(err, check.IsNil)
		metricsURL = mu.String()
	}

	req, err := http.NewRequest(http.MethodGet, metricsURL, nil)
	c.Assert(err, check.IsNil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ManagementToken)

	resp, err := hc.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)
	c.Check(string(body), check.Matches, `(?ms).*`+oldhash+`.*`)

	f, err := os.OpenFile(os.Getenv("ARVADOS_CONFIG"), os.O_WRONLY|os.O_APPEND, 0)
	c.Assert(err, check.IsNil)
	_, err = f.Write([]byte{'\n'})
	c.Assert(err, check.IsNil)
	err = f.Close()
	c.Assert(err, check.IsNil)
	newhash := fmt.Sprintf("%x", sha256.Sum256(append(confdata, '\n')))
	c.Logf("newhash %s", newhash)

	// Wait for RailsAPI's 1 Hz reload_config thread to poll and
	// hit restart.txt
	pollstart := time.Now()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(time.Second) {
		resp, err = hc.Do(req)
		c.Assert(err, check.IsNil)
		c.Check(resp.StatusCode, check.Equals, http.StatusOK)
		body, err = ioutil.ReadAll(resp.Body)
		c.Assert(err, check.IsNil)
		if strings.Contains(string(body), newhash) {
			break
		}
	}
	c.Logf("waited %s for rails to restart", time.Now().Sub(pollstart))
	c.Check(string(body), check.Matches, `(?ms).*`+newhash+`.*`)
}
