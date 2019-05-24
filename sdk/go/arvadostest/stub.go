// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"net/http"
	"net/url"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// StubResponse struct with response status and body
type StubResponse struct {
	Status int
	Body   string
}

// ServerStub with response map of path and StubResponse
// Ex:  /arvados/v1/keep_services = arvadostest.StubResponse{200, string(`{}`)}
type ServerStub struct {
	Responses map[string]StubResponse
}

func (stub *ServerStub) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/redirect-loop" {
		http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
		return
	}

	pathResponse := stub.Responses[req.URL.Path]
	if pathResponse.Status == -1 {
		http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
	} else if pathResponse.Body != "" {
		resp.WriteHeader(pathResponse.Status)
		resp.Write([]byte(pathResponse.Body))
	} else {
		resp.WriteHeader(500)
		resp.Write([]byte(``))
	}
}

// SetServiceURL overrides the given service config/discovery with the
// given internalURL.
//
// SetServiceURL panics on errors.
func SetServiceURL(service *arvados.Service, internalURL string) {
	u, err := url.Parse(internalURL)
	if err != nil {
		panic(err)
	}
	service.InternalURLs = map[arvados.URL]arvados.ServiceInstance{arvados.URL(*u): {}}
	service.ExternalURL = arvados.URL(*u)
}
