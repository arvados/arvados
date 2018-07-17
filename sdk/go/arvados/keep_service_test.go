// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"net/http"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&KeepServiceSuite{})

type KeepServiceSuite struct{}

func (*KeepServiceSuite) TestIndexTimeout(c *check.C) {
	client := &Client{
		Client: &http.Client{
			Transport: &timeoutTransport{response: []byte("\n")},
		},
		APIHost:   "zzzzz.arvadosapi.com",
		AuthToken: "xyzzy",
	}
	_, err := (&KeepService{}).IndexMount(client, "fake", "")
	c.Check(err, check.ErrorMatches, `.*timeout.*`)
}
