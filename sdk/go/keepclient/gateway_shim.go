// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// keepViaHTTP implements arvados.KeepGateway by using a KeepClient to
// do upstream requests to keepstore and keepproxy.
//
// This enables KeepClient to use KeepGateway wrappers (like
// arvados.DiskCache) to wrap its own HTTP client back-end methods
// (getOrHead, httpBlockWrite).
//
// See (*KeepClient)upstreamGateway() for the relevant glue.
type keepViaHTTP struct {
	*KeepClient
}

func (kvh *keepViaHTTP) ReadAt(locator string, dst []byte, offset int) (int, error) {
	rdr, _, _, _, err := kvh.getOrHead("GET", locator, nil)
	if err != nil {
		return 0, err
	}
	defer rdr.Close()
	_, err = io.CopyN(io.Discard, rdr, int64(offset))
	if err != nil {
		return 0, err
	}
	n, err := rdr.Read(dst)
	return int(n), err
}

func (kvh *keepViaHTTP) BlockRead(ctx context.Context, opts arvados.BlockReadOptions) (int, error) {
	rdr, _, _, _, err := kvh.getOrHead("GET", opts.Locator, nil)
	if err != nil {
		return 0, err
	}
	defer rdr.Close()
	n, err := io.Copy(opts.WriteTo, rdr)
	return int(n), err
}

func (kvh *keepViaHTTP) BlockWrite(ctx context.Context, req arvados.BlockWriteOptions) (arvados.BlockWriteResponse, error) {
	return kvh.httpBlockWrite(ctx, req)
}

func (kvh *keepViaHTTP) LocalLocator(locator string) (string, error) {
	if !strings.Contains(locator, "+R") {
		// Either it has +A, or it's unsigned and we assume
		// it's a local locator on a site with signatures
		// disabled.
		return locator, nil
	}
	sighdr := fmt.Sprintf("local, time=%s", time.Now().UTC().Format(time.RFC3339))
	_, _, url, hdr, err := kvh.KeepClient.getOrHead("HEAD", locator, http.Header{"X-Keep-Signature": []string{sighdr}})
	if err != nil {
		return "", err
	}
	loc := hdr.Get("X-Keep-Locator")
	if loc == "" {
		return "", fmt.Errorf("missing X-Keep-Locator header in HEAD response from %s", url)
	}
	return loc, nil
}
