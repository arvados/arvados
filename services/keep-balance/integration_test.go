package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&integrationSuite{})

type integrationSuite struct {
	config     Config
	keepClient *keepclient.KeepClient
}

func (s *integrationSuite) SetUpSuite(c *check.C) {
	if testing.Short() {
		c.Skip("-short")
	}
	arvadostest.ResetEnv()
	arvadostest.StartAPI()
	arvadostest.StartKeep(4, true)

	arv, err := arvadosclient.MakeArvadosClient()
	arv.ApiToken = arvadostest.DataManagerToken
	c.Assert(err, check.IsNil)
	s.keepClient = &keepclient.KeepClient{
		Arvados: &arv,
		Client:  &http.Client{},
	}
	c.Assert(s.keepClient.DiscoverKeepServers(), check.IsNil)
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
	arvadostest.StopAPI()
}

func (s *integrationSuite) SetUpTest(c *check.C) {
	s.config = Config{
		Client: arvados.Client{
			APIHost:   os.Getenv("ARVADOS_API_HOST"),
			AuthToken: arvadostest.DataManagerToken,
			Insecure:  true,
		},
		KeepServiceTypes: []string{"disk"},
	}
}

func (s *integrationSuite) TestBalanceAPIFixtures(c *check.C) {
	var logBuf *bytes.Buffer
	for iter := 0; iter < 20; iter++ {
		logBuf := &bytes.Buffer{}
		opts := RunOptions{
			CommitPulls: true,
			CommitTrash: true,
			Logger:      log.New(logBuf, "", log.LstdFlags),
		}
		nextOpts, err := (&Balancer{}).Run(s.config, opts)
		c.Check(err, check.IsNil)
		c.Check(nextOpts.SafeRendezvousState, check.Not(check.Equals), "")
		c.Check(nextOpts.CommitPulls, check.Equals, true)
		if iter == 0 {
			c.Check(logBuf.String(), check.Matches, `(?ms).*ChangeSet{Pulls:1.*`)
			c.Check(logBuf.String(), check.Not(check.Matches), `(?ms).*ChangeSet{.*Trashes:[^0]}*`)
		} else if strings.Contains(logBuf.String(), "ChangeSet{Pulls:0") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	c.Check(logBuf.String(), check.Not(check.Matches), `(?ms).*0 replicas (0 blocks, 0 bytes) underreplicated.*`)
}
