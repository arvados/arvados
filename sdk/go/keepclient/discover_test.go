package keepclient

import (
	"crypto/md5"
	"fmt"
	"gopkg.in/check.v1"
	"net/http"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
)

func ExampleKeepClient_RefreshServices() {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		panic(err)
	}
	kc, err := MakeKeepClient(&arv)
	if err != nil {
		panic(err)
	}
	go kc.RefreshServices(5*time.Minute, 3*time.Second)
	fmt.Printf("LocalRoots: %#v\n", kc.LocalRoots())
}

func (s *ServerRequiredSuite) TestOverrideDiscovery(c *check.C) {
	defer os.Setenv("ARVADOS_KEEP_SERVICES", "")

	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("TestOverrideDiscovery")))
	st := StubGetHandler{
		c,
		hash,
		arvadostest.ActiveToken,
		http.StatusOK,
		[]byte("TestOverrideDiscovery")}
	ks := RunSomeFakeKeepServers(st, 2)

	os.Setenv("ARVADOS_KEEP_SERVICES", "")
	arv1, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	arv1.ApiToken = arvadostest.ActiveToken

	os.Setenv("ARVADOS_KEEP_SERVICES", ks[0].url+"  "+ks[1].url+" ")
	arv2, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	arv2.ApiToken = arvadostest.ActiveToken

	// ARVADOS_KEEP_SERVICES was empty when we created arv1, but
	// it pointed to our stub servers when we created
	// arv2. Regardless of what it's set to now, a keepclient for
	// arv2 should use our stub servers, but one created for arv1
	// should not.

	kc1, err := MakeKeepClient(&arv1)
	c.Assert(err, check.IsNil)
	kc2, err := MakeKeepClient(&arv2)
	c.Assert(err, check.IsNil)

	_, _, _, err = kc1.Get(hash)
	c.Check(err, check.NotNil)
	_, _, _, err = kc2.Get(hash)
	c.Check(err, check.IsNil)
}
