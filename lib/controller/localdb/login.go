// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type loginController interface {
	Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error)
	Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error)
}

func chooseLoginController(cluster *arvados.Cluster, railsProxy *railsProxy) loginController {
	wantGoogle := cluster.Login.GoogleClientID != ""
	wantSSO := cluster.Login.ProviderAppID != ""
	wantPAM := cluster.Login.PAM
	switch {
	case wantGoogle && !wantSSO && !wantPAM:
		return &googleLoginController{Cluster: cluster, RailsProxy: railsProxy}
	case !wantGoogle && wantSSO && !wantPAM:
		return railsProxy
	case !wantGoogle && !wantSSO && wantPAM:
		return &pamLoginController{Cluster: cluster, RailsProxy: railsProxy}
	default:
		return errorLoginController{
			error: errors.New("configuration problem: exactly one of Login.GoogleClientID, Login.ProviderAppID, or Login.PAM must be configured"),
		}
	}
}

type errorLoginController struct{ error }

func (ctrl errorLoginController) Login(context.Context, arvados.LoginOptions) (arvados.LoginResponse, error) {
	return arvados.LoginResponse{}, ctrl.error
}
func (ctrl errorLoginController) Logout(context.Context, arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return arvados.LogoutResponse{}, ctrl.error
}

func noopLogout(cluster *arvados.Cluster, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	target := opts.ReturnTo
	if target == "" {
		if cluster.Services.Workbench2.ExternalURL.Host != "" {
			target = cluster.Services.Workbench2.ExternalURL.String()
		} else {
			target = cluster.Services.Workbench1.ExternalURL.String()
		}
	}
	return arvados.LogoutResponse{RedirectLocation: target}, nil
}
