package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&IntegrationSuite{})

// IntegrationSuite tests need an API server and a keep-web server
type IntegrationSuite struct{}

func (s *IntegrationSuite) SetUpSuite(c *check.C) {
	arvadostest.StartAPI()
	arvadostest.StartKeep(2, true)

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)
	arv.ApiToken = arvadostest.ActiveToken
	kc, err := keepclient.MakeKeepClient(arv)
	c.Assert(err, check.Equals, nil)
	kc.PutB([]byte("Hello world\n"))
	kc.PutB([]byte("foo"))
	kc.PutB([]byte("foobar"))
}

func (s *IntegrationSuite) TearDownSuite(c *check.C) {
	arvadostest.StopKeep(2)
	arvadostest.StopAPI()
}
