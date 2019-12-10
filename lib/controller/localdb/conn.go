// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"

	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type railsProxy = rpc.Conn

type Conn struct {
	cluster     *arvados.Cluster
	*railsProxy // handles API methods that aren't defined on Conn itself

	googleLoginController
}

func NewConn(cluster *arvados.Cluster) *Conn {
	return &Conn{
		cluster:    cluster,
		railsProxy: railsproxy.NewConn(cluster),
	}
}

func (conn *Conn) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	wantGoogle := conn.cluster.Login.GoogleClientID != ""
	wantSSO := conn.cluster.Login.ProviderAppID != ""
	if wantGoogle == wantSSO {
		return arvados.LoginResponse{}, errors.New("configuration problem: exactly one of Login.GoogleClientID and Login.ProviderAppID must be configured")
	} else if wantGoogle {
		return conn.googleLoginController.Login(ctx, conn.cluster, conn.railsProxy, opts)
	} else {
		// Proxy to RailsAPI, which hands off to sso-provider.
		return conn.railsProxy.Login(ctx, opts)
	}
}
