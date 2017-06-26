package main

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
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
	s.srv.WaitReady()
	resp, err := http.Get("http://" + s.srv.listener.Addr().String() + "/_health/ping")
	c.Check(err, check.IsNil)
	buf, err := ioutil.ReadAll(resp.Body)
	c.Check(err, check.IsNil)
	c.Check(string(buf), check.Equals, `{"health":"OK"}`+"\n")
}
