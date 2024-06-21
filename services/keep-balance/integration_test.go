// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&integrationSuite{})

type integrationSuite struct {
	config     *arvados.Cluster
	db         *sqlx.DB
	client     *arvados.Client
	keepClient *keepclient.KeepClient
}

func (s *integrationSuite) SetUpSuite(c *check.C) {
	if testing.Short() {
		c.Skip("-short")
	}
	arvadostest.ResetEnv()
	arvadostest.StartKeep(4, true)

	arv, err := arvadosclient.MakeArvadosClient()
	arv.ApiToken = arvadostest.SystemRootToken
	c.Assert(err, check.IsNil)

	s.keepClient, err = keepclient.MakeKeepClient(arv)
	c.Assert(err, check.IsNil)
	s.keepClient.DiskCacheSize = keepclient.DiskCacheDisabled
	s.putReplicas(c, "foo", 4)
	s.putReplicas(c, "bar", 1)
}

func (s *integrationSuite) putReplicas(c *check.C, data string, replicas int) {
	s.keepClient.Want_replicas = replicas
	_, _, err := s.keepClient.PutB([]byte(data))
	c.Assert(err, check.IsNil)
}

func (s *integrationSuite) TearDownSuite(c *check.C) {
	if testing.Short() {
		c.Skip("-short")
	}
	arvadostest.StopKeep(4)
}

func (s *integrationSuite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.Equals, nil)
	s.config, err = cfg.GetCluster("")
	c.Assert(err, check.Equals, nil)
	s.db, err = sqlx.Open("postgres", s.config.PostgreSQL.Connection.String())
	c.Assert(err, check.IsNil)
	s.config.Collections.BalancePeriod = arvados.Duration(time.Second)

	s.client = &arvados.Client{
		APIHost:   os.Getenv("ARVADOS_API_HOST"),
		AuthToken: arvadostest.SystemRootToken,
		Insecure:  true,
	}
}

func (s *integrationSuite) TestBalanceAPIFixtures(c *check.C) {
	var logBuf bytes.Buffer
	for iter := 0; iter < 20; iter++ {
		logBuf.Reset()
		logger := logrus.New()
		logger.Out = io.MultiWriter(&logBuf, ctxlog.LogWriter(c.Log))
		opts := RunOptions{
			CommitConfirmedFields: true,
			Logger:                logger,
		}

		bal := &Balancer{
			DB:      s.db,
			Logger:  logger,
			Metrics: newMetrics(prometheus.NewRegistry()),
		}
		nextOpts, err := bal.Run(context.Background(), s.client, s.config, opts)
		c.Check(err, check.IsNil)
		c.Check(nextOpts.SafeRendezvousState, check.Not(check.Equals), "")
		if iter == 0 {
			c.Check(logBuf.String(), check.Matches, `(?ms).*ChangeSet{Pulls:1.*`)
			c.Check(logBuf.String(), check.Not(check.Matches), `(?ms).*ChangeSet{.*Trashes:[^0]}*`)
		} else if !strings.Contains(logBuf.String(), "ChangeSet{Pulls:1") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	c.Check(logBuf.String(), check.Not(check.Matches), `(?ms).*0 replicas (0 blocks, 0 bytes) underreplicated.*`)

	for _, trial := range []struct {
		uuid    string
		repl    int
		classes []string
	}{
		{arvadostest.EmptyCollectionUUID, 0, []string{}},
		{arvadostest.FooCollection, 2, []string{"default"}},                                // "foo" blk
		{arvadostest.StorageClassesDesiredDefaultConfirmedDefault, 2, []string{"default"}}, // "bar" blk
		{arvadostest.StorageClassesDesiredArchiveConfirmedDefault, 0, []string{}},          // "bar" blk
	} {
		c.Logf("%#v", trial)
		var coll arvados.Collection
		s.client.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+trial.uuid, nil, nil)
		if c.Check(coll.ReplicationConfirmed, check.NotNil) {
			c.Check(*coll.ReplicationConfirmed, check.Equals, trial.repl)
		}
		c.Check(coll.StorageClassesConfirmed, check.DeepEquals, trial.classes)
	}
}
