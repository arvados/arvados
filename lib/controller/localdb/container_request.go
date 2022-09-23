// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// ContainerRequestCreate defers to railsProxy for everything except
// vocabulary checking.
func (conn *Conn) ContainerRequestCreate(ctx context.Context, opts arvados.CreateOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.ContainerRequest{}, err
	}
	resp, err := conn.railsProxy.ContainerRequestCreate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// ContainerRequestUpdate defers to railsProxy for everything except
// vocabulary checking.
func (conn *Conn) ContainerRequestUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.ContainerRequest{}, err
	}
	resp, err := conn.railsProxy.ContainerRequestUpdate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (conn *Conn) ContainerRequestGet(ctx context.Context, opts arvados.GetOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.ContainerRequestGet(ctx, opts)
}

func (conn *Conn) ContainerRequestList(ctx context.Context, opts arvados.ListOptions) (arvados.ContainerRequestList, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.ContainerRequestList(ctx, opts)
}

func (conn *Conn) ContainerRequestDelete(ctx context.Context, opts arvados.DeleteOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.ContainerRequestDelete(ctx, opts)
}
