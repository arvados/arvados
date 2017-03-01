package main

import (
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&serverSuite{})

type serverSuite struct {
}

func testConfig() *wsConfig {
	cfg := defaultConfig()
	cfg.Client = *(arvados.NewClientFromEnv())
	cfg.Postgres = testDBConfig()
	cfg.Listen = ":"
	return &cfg
}

// TestBadDB ensures Run() returns an error (instead of panicking or
// deadlocking) if it can't connect to the database server at startup.
func (s *serverSuite) TestBadDB(c *check.C) {
	cfg := testConfig()
	cfg.Postgres["password"] = "1234"
	srv := &server{wsConfig: cfg}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		err := srv.Run()
		c.Check(err, check.NotNil)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		srv.WaitReady()
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

func newTestServer() *server {
	srv := &server{wsConfig: testConfig()}
	go srv.Run()
	srv.WaitReady()
	return srv
}
