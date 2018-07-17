// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"os"

	"gopkg.in/check.v1"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
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
