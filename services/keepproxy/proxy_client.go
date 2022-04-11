// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepproxy

import (
	"net/http"

	"git.arvados.org/arvados.git/sdk/go/keepclient"
)

var viaAlias = "keepproxy"

type proxyClient struct {
	client keepclient.HTTPClient
	proto  string
}

func (pc *proxyClient) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Via", pc.proto+" "+viaAlias)
	return pc.client.Do(req)
}
