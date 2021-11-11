// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// GroupCreate defers to railsProxy for everything except vocabulary
// checking.
func (conn *Conn) GroupCreate(ctx context.Context, opts arvados.CreateOptions) (arvados.Group, error) {
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Group{}, err
	}
	resp, err := conn.railsProxy.GroupCreate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// GroupUpdate defers to railsProxy for everything except vocabulary
// checking.
func (conn *Conn) GroupUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.Group, error) {
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Group{}, err
	}
	resp, err := conn.railsProxy.GroupUpdate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}
