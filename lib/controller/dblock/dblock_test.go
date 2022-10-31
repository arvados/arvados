// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dblock

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&suite{})

type suite struct {
	cluster *arvados.Cluster
	db      *sqlx.DB
	getdb   func(context.Context) (*sqlx.DB, error)
}

var testLocker = &DBLocker{key: 999}

func (s *suite) SetUpSuite(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.db = arvadostest.DB(c, s.cluster)
	s.getdb = func(context.Context) (*sqlx.DB, error) { return s.db, nil }
}

func (s *suite) TestLock(c *check.C) {
	retryDelay = 10 * time.Millisecond

	var logbuf bytes.Buffer
	logger := ctxlog.New(&logbuf, "text", "debug")
	logger.Level = logrus.DebugLevel
	ctx := ctxlog.Context(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	testLocker.Lock(ctx, s.getdb)
	testLocker.Check()

	lock2 := make(chan bool)
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		testLocker2 := &DBLocker{key: 999}
		testLocker2.Lock(ctx, s.getdb)
		close(lock2)
		testLocker2.Check()
		testLocker2.Unlock()
	}()

	// Second lock should wait for first to Unlock
	select {
	case <-time.After(time.Second / 10):
		c.Check(logbuf.String(), check.Matches, `(?ms).*level=info.*DBClient="[^"]+:\d+".*ID=999.*`)
	case <-lock2:
		c.Log("double-lock")
		c.Fail()
	}

	testLocker.Check()
	testLocker.Unlock()

	// Now the second lock should succeed within retryDelay
	select {
	case <-time.After(retryDelay * 2):
		c.Log("timed out")
		c.Fail()
	case <-lock2:
	}
	c.Logf("%s", logbuf.String())
}
