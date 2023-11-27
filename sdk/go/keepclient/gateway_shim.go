// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"context"
	"io"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// keepViaHTTP implements arvados.KeepGateway by using a KeepClient to
// do upstream requests to keepstore and keepproxy.
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

// keepViaBlockCache implements arvados.KeepGateway by using the given
// KeepClient's BlockCache with the wrapped KeepGateway.
//
// Note the whole KeepClient gets passed in instead of just its
// cache. This ensures the new BlockCache gets used if it changes
// after keepViaBlockCache is initialized.
type keepViaBlockCache struct {
	kc *KeepClient
	arvados.KeepGateway
}

func (kvbc *keepViaBlockCache) ReadAt(locator string, dst []byte, offset int) (int, error) {
	return kvbc.kc.cache().ReadAt(kvbc.KeepGateway, locator, dst, offset)
}

func (kvbc *keepViaBlockCache) BlockRead(ctx context.Context, opts arvados.BlockReadOptions) (int, error) {
	rdr, _, _, _, err := kvbc.kc.getOrHead("GET", opts.Locator, nil)
	if err != nil {
		return 0, err
	}
	defer rdr.Close()
	n, err := io.Copy(opts.WriteTo, rdr)
	return int(n), err
}
