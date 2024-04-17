// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"sort"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&eventSourceSuite{})

type eventSourceSuite struct{}

func testDBConfig() arvados.PostgreSQLConnection {
	cfg, err := arvados.GetConfig(arvados.DefaultConfigFile)
	if err != nil {
		panic(err)
	}
	cc, err := cfg.GetCluster("zzzzz")
	if err != nil {
		panic(err)
	}
	return cc.PostgreSQL.Connection
}

func testDB() *sql.DB {
	db, err := sql.Open("postgres", testDBConfig().String())
	if err != nil {
		panic(err)
	}
	return db
}

func (*eventSourceSuite) TestEventSource(c *check.C) {
	var logfixtures map[string]struct {
		ID int
	}
	yamldata, err := ioutil.ReadFile("../api/test/fixtures/logs.yml")
	c.Assert(err, check.IsNil)
	c.Assert(yaml.Unmarshal(yamldata, &logfixtures), check.IsNil)
	var logIDs []int
	for _, logfixture := range logfixtures {
		logIDs = append(logIDs, logfixture.ID)
	}
	sort.Ints(logIDs)

	cfg := testDBConfig()
	db := testDB()
	pges := &pgEventSource{
		DataSource: cfg.String(),
		QueueSize:  4,
		Logger:     ctxlog.TestLogger(c),
		Reg:        prometheus.NewRegistry(),
	}
	go pges.Run()
	sinks := make([]eventSink, 18)
	for i := range sinks {
		sinks[i] = pges.NewSink()
	}

	pges.WaitReady()
	defer pges.cancel()

	done := make(chan bool, 1)

	go func() {
		for _, id := range logIDs {
			_, err := db.Exec(fmt.Sprintf(`NOTIFY logs, '%d'`, id))
			if err != nil {
				done <- true
				c.Fatal(err)
				return
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(sinks))
	for si, s := range sinks {
		go func(si int, s eventSink) {
			defer wg.Done()
			defer sinks[si].Stop()
			for _, logID := range logIDs {
				ev := <-sinks[si].Channel()
				c.Logf("sink %d received event %d", si, logID)
				c.Check(ev.LogID, check.Equals, int64(logID))
				row := ev.Detail()
				if c.Check(row, check.NotNil) {
					c.Check(row.ID, check.Equals, int64(logID))
					c.Check(row.UUID, check.Not(check.Equals), "")
				}
			}
		}(si, s)
	}
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		c.Fatal("timed out")
	}

	c.Check(pges.DBHealth(), check.IsNil)
}
