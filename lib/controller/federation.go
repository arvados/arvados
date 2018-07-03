// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"

	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

var wfRe = regexp.MustCompile(`^/arvados/v1/workflows/([0-9a-z]{5})-[^/]+$`)

func (h *Handler) proxyRemoteCluster(w http.ResponseWriter, req *http.Request, next http.Handler) {
	m := wfRe.FindStringSubmatch(req.URL.Path)
	if len(m) < 2 || m[1] == h.Cluster.ClusterID {
		next.ServeHTTP(w, req)
		return
	}
	remoteID := m[1]
	remote, ok := h.Cluster.RemoteClusters[remoteID]
	if !ok {
		httpserver.Error(w, "no proxy available for cluster "+remoteID, http.StatusNotFound)
		return
	}
	scheme := remote.Scheme
	if scheme == "" {
		scheme = "https"
	}
	err := h.saltAuthToken(req, remoteID)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	urlOut := &url.URL{
		Scheme:   scheme,
		Host:     remote.Host,
		Path:     req.URL.Path,
		RawPath:  req.URL.RawPath,
		RawQuery: req.URL.RawQuery,
	}
	client := h.secureClient
	if remote.Insecure {
		client = h.insecureClient
	}
	h.proxy.Do(w, req, urlOut, client)
}

// Extract the auth token supplied in req, and replace it with a
// salted token for the remote cluster.
func (h *Handler) saltAuthToken(req *http.Request, remote string) error {
	creds := auth.NewCredentials()
	creds.LoadTokensFromHTTPRequest(req)
	if len(creds.Tokens) == 0 && req.Header.Get("Content-Type") == "application/x-www-form-encoded" {
		// Override ParseForm's 10MiB limit by ensuring
		// req.Body is a *http.maxBytesReader.
		req.Body = http.MaxBytesReader(nil, req.Body, 1<<28) // 256MiB. TODO: use MaxRequestSize from discovery doc or config.
		if err := creds.LoadTokensFromHTTPRequestBody(req); err != nil {
			return err
		}
		// Replace req.Body with a buffer that re-encodes the
		// form without api_token, in case we end up
		// forwarding the request.
		if req.PostForm != nil {
			req.PostForm.Del("api_token")
		}
		req.Body = ioutil.NopCloser(bytes.NewBufferString(req.PostForm.Encode()))
	}
	if len(creds.Tokens) == 0 {
		return nil
	}
	token, err := auth.SaltToken(creds.Tokens[0], remote)
	if err == auth.ErrObsoleteToken {
		// FIXME: If the token exists in our own database,
		// salt it for the remote. Otherwise, assume it was
		// issued by the remote, and pass it through
		// unmodified.
		token = creds.Tokens[0]
	} else if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Remove api_token=... from the the query string, in case we
	// end up forwarding the request.
	if values, err := url.ParseQuery(req.URL.RawQuery); err != nil {
		return err
	} else if _, ok := values["api_token"]; ok {
		delete(values, "api_token")
		req.URL.RawQuery = values.Encode()
	}
	return nil
}
