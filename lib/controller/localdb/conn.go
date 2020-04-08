// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type railsProxy = rpc.Conn

type Conn struct {
	cluster     *arvados.Cluster
	*railsProxy // handles API methods that aren't defined on Conn itself
	loginController
}

func NewConn(cluster *arvados.Cluster) *Conn {
	railsProxy := railsproxy.NewConn(cluster)
	return &Conn{
		cluster:         cluster,
		railsProxy:      railsProxy,
		loginController: chooseLoginController(cluster, railsProxy),
	}
}

func (conn *Conn) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return conn.loginController.Logout(ctx, opts)
}

func (conn *Conn) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	return conn.loginController.Login(ctx, opts)
}

func (conn *Conn) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return conn.loginController.UserAuthenticate(ctx, opts)
}
