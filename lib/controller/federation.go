// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/auth"
)

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
		// forwarding the request to RailsAPI.
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
	return nil
}
