// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"io"
	"net/http"
	"net/url"

	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

type proxy struct {
	Name string // to use in Via header
}

type HTTPError struct {
	Message string
	Code    int
}

func (h HTTPError) Error() string {
	return h.Message
}

var dropHeaders = map[string]bool{
	// Headers that shouldn't be forwarded when proxying. See
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	// (comment/space here makes gofmt1.10 agree with gofmt1.11)
	"TE":      true,
	"Trailer": true,
	"Upgrade": true,

	// Headers that would interfere with Go's automatic
	// compression/decompression if we forwarded them.
	"Accept-Encoding":   true,
	"Content-Encoding":  true,
	"Transfer-Encoding": true,

	// Content-Length depends on encoding.
	"Content-Length": true,
}

type ResponseFilter func(*http.Response, error) (*http.Response, error)

// Forward a request to upstream service, and return response or error.
func (p *proxy) Do(
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
	xff := ""
	for _, xffIn := range reqIn.Header["X-Forwarded-For"] {
		if xffIn != "" {
			xff += xffIn + ","
		}
	}
	xff += reqIn.RemoteAddr
	hdrOut.Set("X-Forwarded-For", xff)
	if hdrOut.Get("X-Forwarded-Proto") == "" {
		hdrOut.Set("X-Forwarded-Proto", reqIn.URL.Scheme)
	}
	hdrOut.Add("Via", reqIn.Proto+" arvados-controller")

	reqOut := (&http.Request{
		Method: reqIn.Method,
		URL:    urlOut,
		Host:   reqIn.Host,
		Header: hdrOut,
		Body:   reqIn.Body,
	}).WithContext(reqIn.Context())
	return client.Do(reqOut)
}

// Copy a response (or error) to the downstream client
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
