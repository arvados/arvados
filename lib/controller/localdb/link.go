// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// LinkCreate defers to railsProxy for everything except vocabulary
// checking.
func (conn *Conn) LinkCreate(ctx context.Context, opts arvados.CreateOptions) (arvados.Link, error) {
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Link{}, err
	}
	resp, err := conn.railsProxy.LinkCreate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// LinkUpdate defers to railsProxy for everything except vocabulary
// checking.
func (conn *Conn) LinkUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.Link, error) {
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Link{}, err
	}
	resp, err := conn.railsProxy.LinkUpdate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
