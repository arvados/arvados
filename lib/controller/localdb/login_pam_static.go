// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

//go:build static

package localdb

import (
	"context"
	"errors"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type pamLoginController struct {
	Cluster *arvados.Cluster
	Parent  *Conn
}

func (ctrl *pamLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return logout(ctx, ctrl.Cluster, opts)
}

func (ctrl *pamLoginController) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	return arvados.LoginResponse{}, errors.New("interactive login is not available")
}

func (ctrl *pamLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return arvados.APIClientAuthorization{}, errors.New("support not available due to static compilation")
}
