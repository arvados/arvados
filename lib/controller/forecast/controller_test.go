// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package forecast

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"

	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&ForecastSuite{})

type ForecastSuite struct {
	ctrl     *Controller
	ctx      context.Context
	stub     *arvadostest.APIStub
	rollback func()
}

func integrationTestCluster() *arvados.Cluster {
	cfg, err := arvados.GetConfig(filepath.Join(os.Getenv("WORKSPACE"), "tmp", "arvados.yml"))
	if err != nil {
		panic(err)
	}
	cc, err := cfg.GetCluster("zzzzz")
	if err != nil {
		panic(err)
	}
	return cc
}

func (s *ForecastSuite) SetUpTest(c *check.C) {
	s.ctx = context.Background()
	// default user that has access to the fixtures
	s.ctx = auth.NewContext(s.ctx, &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})
	s.ctx = ctxlog.Context(s.ctx, ctxlog.New(os.Stderr, "json", "debug"))
	cluster := &arvados.Cluster{
		ClusterID:  "zzzzz",
		PostgreSQL: integrationTestCluster().PostgreSQL,
	}
	cluster.API.RequestTimeout = arvados.Duration(5 * time.Minute)
	cluster.TLS.Insecure = true
	arvadostest.SetServiceURL(&cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	arvadostest.SetServiceURL(&cluster.Services.Controller, "http://localhost:/")

	s.ctrl = New(cluster, rpc.NewConn(
		cluster.ClusterID,
		&url.URL{Scheme: "https", Host: os.Getenv("ARVADOS_TEST_API_HOST")},
		true, rpc.PassthroughTokenProvider))

}

func (s *ForecastSuite) TearDownTest(c *check.C) {
	if s.rollback != nil {
		s.rollback()
		s.rollback = nil
	}
}

func (s *ForecastSuite) TestDatapoints(c *check.C) {
	// Just as basic test to make sure the datapoints endpoint is there and giving out some data
	// CompletedDiagnosticsContainerRequest1UUID (zzzzz-xvhdp-p1i7h1gy5z1ft4p) is hasher_root from services/api/test/fixtures/container_requests.yml
	// this container request start all the
	resp, err := s.ctrl.ForecastDatapoints(s.ctx, arvados.GetOptions{UUID: arvadostest.CompletedDiagnosticsContainerRequest1UUID})
	c.Check(err, check.IsNil)
	c.Check(len(resp.Datapoints), check.Equals, 3)
}

// A great way to update golden files if needed:   go test -update
var update = flag.Bool("update", false, "Update golden files")

func (s *ForecastSuite) TestDatapointsValues(c *check.C) {
	cases := []struct {
		Name       string
		Checkpoint string
	}{
		{"hasher1_data", "hasher1"},
		{"hasher2_data", "hasher2"},
		{"hasher3_data", "hasher3"},
	}
	resp, err := s.ctrl.ForecastDatapoints(s.ctx, arvados.GetOptions{UUID: arvadostest.CompletedDiagnosticsContainerRequest1UUID})
	c.Check(err, check.IsNil)

	for _, tc := range cases {
		var actual arvados.Datapoint
		for _, d := range resp.Datapoints {
			if d.Checkpoint == tc.Checkpoint {
				actual = d
			}
		}
		c.Check(actual, check.NotNil)

		actualBytes, err := json.Marshal(actual)
		c.Check(err, check.IsNil)

		golden := filepath.Join("test-fixtures", tc.Name+".golden")

		if *update {
			err = ioutil.WriteFile(golden, actualBytes, 0644)
			c.Check(err, check.IsNil)
		}

		expected, err := ioutil.ReadFile(golden)
		c.Check(err, check.IsNil)
		c.Check(string(actualBytes), check.Equals, string(expected))
	}
}
