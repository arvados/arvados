// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

func remoteContainerRequestCreate(
	h *genericFederatedRequestHandler,
	effectiveMethod string,
	clusterId *string,
	uuid string,
	remainder string,
	w http.ResponseWriter,
	req *http.Request) bool {

	if effectiveMethod != "POST" || uuid != "" || remainder != "" ||
		*clusterId == "" || *clusterId == h.handler.Cluster.ClusterID {
		return false
	}

	defer req.Body.Close()
	var request map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&request)

	containerRequest, ok := request["container_request"].(map[string]interface{})
	if !ok {
		log.Printf("wah wah")
		return false
	}

	// If runtime_token is not set, create a new token
	if _, ok := containerRequest["runtime_token"]; !ok {
		log.Printf("ok %v", ok)

		// First make sure supplied token is valid.
		creds := auth.NewCredentials()
		creds.LoadTokensFromHTTPRequest(req)

		currentUser, err := h.handler.validateAPItoken(req, creds.Tokens[0])
		if err != nil {
			httpserver.Error(w, err.Error(), http.StatusForbidden)
			return true
		}

		if len(currentUser.Authorization.Scopes) != 1 || currentUser.Authorization.Scopes[0] != "all" {
			return false
		}

		newtok, err := h.handler.createAPItoken(req, currentUser.UUID, nil)
		if err != nil {
			httpserver.Error(w, err.Error(), http.StatusForbidden)
			return true
		}
		containerRequest["runtime_token"] = newtok.TokenV2()
	}

	newbody, err := json.Marshal(request)
	buf := bytes.NewBuffer(newbody)
	req.Body = ioutil.NopCloser(buf)
	req.ContentLength = int64(buf.Len())
	req.Header.Set("Content-Length", fmt.Sprintf("%v", buf.Len()))

	resp, err := h.handler.remoteClusterRequest(*clusterId, req)
	h.handler.proxy.ForwardResponse(w, resp, err)
	return true
}
