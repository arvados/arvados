// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type proxy struct {
	Name           string // to use in Via header
	RequestTimeout time.Duration
}

type HTTPError struct {
	Message string
	Code    int
}

func (h HTTPError) Error() string {
	return h.Message
}

// headers that shouldn't be forwarded when proxying. See
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
var dropHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"TE":                true,
	"Trailer":           true,
	"Transfer-Encoding": true, // *-Encoding headers interfer with Go's automatic compression/decompression
	"Content-Encoding":  true,
	"Accept-Encoding":   true,
	"Upgrade":           true,
}

type ResponseFilter func(*http.Response, error) (*http.Response, error)

// Forward a request to downstream service, and return response or error.
func (p *proxy) ForwardRequest(
	reqIn *http.Request,
	urlOut *url.URL,
	client *http.Client) (*http.Response, error) {

	// Copy headers from incoming request, then add/replace proxy
	// headers like Via and X-Forwarded-For.
	hdrOut := http.Header{}
	for k, v := range reqIn.Header {
		if !dropHeaders[k] {
			hdrOut[k] = v
		}
	}
	xff := reqIn.RemoteAddr
	if xffIn := reqIn.Header.Get("X-Forwarded-For"); xffIn != "" {
		xff = xffIn + "," + xff
	}
	hdrOut.Set("X-Forwarded-For", xff)
	if hdrOut.Get("X-Forwarded-Proto") == "" {
		hdrOut.Set("X-Forwarded-Proto", reqIn.URL.Scheme)
	}
	hdrOut.Add("Via", reqIn.Proto+" arvados-controller")

	ctx := reqIn.Context()
	if p.RequestTimeout > 0 {
		ctx, _ = context.WithDeadline(ctx, time.Now().Add(time.Duration(p.RequestTimeout)))
	}

	reqOut := (&http.Request{
		Method: reqIn.Method,
		URL:    urlOut,
		Host:   reqIn.Host,
		Header: hdrOut,
		Body:   reqIn.Body,
	}).WithContext(ctx)

	return client.Do(reqOut)
}

// Copy a response (or error) to the upstream client
func (p *proxy) ForwardResponse(w http.ResponseWriter, resp *http.Response, err error) (int64, error) {
	if err != nil {
		if he, ok := err.(HTTPError); ok {
			httpserver.Error(w, he.Message, he.Code)
		} else {
			httpserver.Error(w, err.Error(), http.StatusBadGateway)
		}
		return 0, nil
	}

	defer resp.Body.Close()
	for k, v := range resp.Header {
		for _, v := range v {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	return io.Copy(w, resp.Body)
}
