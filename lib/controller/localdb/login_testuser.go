// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"
	"fmt"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

type testLoginController struct {
	Cluster    *arvados.Cluster
	RailsProxy *railsProxy
}

func (ctrl *testLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return noopLogout(ctrl.Cluster, opts)
}

func (ctrl *testLoginController) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	return arvados.LoginResponse{}, errors.New("interactive login is not available")
}

func (ctrl *testLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	for username, user := range ctrl.Cluster.Login.Test.Users {
		if (opts.Username == username || opts.Username == user.Email) && opts.Password == user.Password {
			ctxlog.FromContext(ctx).WithFields(logrus.Fields{
				"username": username,
				"email":    user.Email,
			}).Debug("test authentication succeeded")
			return createAPIClientAuthorization(ctx, ctrl.RailsProxy, ctrl.Cluster.SystemRootToken, rpc.UserSessionAuthInfo{
				Username: username,
				Email:    user.Email,
			})
		}
	}
	return arvados.APIClientAuthorization{}, fmt.Errorf("authentication failed for user %q with password len=%d", opts.Username, len(opts.Password))
}
