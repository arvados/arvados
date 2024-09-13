// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"gopkg.in/check.v1"
)

func (s *ServerRequiredSuite) TestOverrideDiscovery(c *check.C) {
	defer os.Setenv("ARVADOS_KEEP_SERVICES", "")

	data := []byte("TestOverrideDiscovery")
	hash := fmt.Sprintf("%x+%d", md5.Sum(data), len(data))
	st := StubGetHandler{
		c,
		hash,
		arvadostest.ActiveToken,
		http.StatusOK,
		data}
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

	kc1, err := MakeKeepClient(arv1)
	c.Assert(err, check.IsNil)
	kc2, err := MakeKeepClient(arv2)
	c.Assert(err, check.IsNil)

	_, _, _, err = kc1.Get(hash)
	c.Check(err, check.NotNil)
	_, _, _, err = kc2.Get(hash)
	c.Check(err, check.IsNil)
}

func (s *StandaloneSuite) TestKeepServicesFromClusterConfig(c *check.C) {
	// This behavior is disabled via env var in the test
	// environment. Clear the env var to test the default
	// production behavior.
	v := "ARVADOS_USE_KEEP_ACCESSIBLE_API"
	defer os.Setenv(v, os.Getenv(v))
	os.Unsetenv(v)

	rdr := bytes.NewReader([]byte(`
Clusters:
 zzzzz:
  Services:
   Keepstore:
    InternalURLs:
     "https://[::1]:12345/":
      Rendezvous: abcdefghijklmno
     "https://[::1]:54321/":
      Rendezvous: xyz
     "http://0.0.0.0:54321/":
      {}
   Keepproxy:
    InternalURLs:
     "https://[::1]:55555/":
      {}
`))
	ldr := config.NewLoader(rdr, ctxlog.TestLogger(c))
	ldr.Path = "-"
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	c.Assert(cluster.ClusterID, check.Equals, "zzzzz")
	ac, err := arvados.NewClientFromConfig(cluster)
	c.Assert(err, check.IsNil)
	arv1, err := arvadosclient.New(ac)
	c.Assert(err, check.IsNil)
	c.Check(arv1.Cluster, check.NotNil)
	kc, err := MakeKeepClient(arv1)
	c.Assert(err, check.IsNil)
	// Note the default rendezvous string is generated based on
	// the MD5 of the keepstore URL and that URL *must* have a
	// trailing slash in order to match the RailsAPI behavior --
	// meanwhile, the keepstore URL given in the localRoots map
	// *must not* have a trailing slash.
	c.Check(kc.localRoots, check.DeepEquals, map[string]string{
		"zzzzz-bi6l4-abcdefghijklmno":                                                "https://[::1]:12345",
		fmt.Sprintf("zzzzz-bi6l4-%x", md5.Sum([]byte("xyz")))[:27]:                   "https://[::1]:54321",
		fmt.Sprintf("zzzzz-bi6l4-%x", md5.Sum([]byte("http://0.0.0.0:54321/")))[:27]: "http://0.0.0.0:54321",
	})
}
