// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"
	"net/http"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

type loginController interface {
	Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error)
	Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error)
	UserAuthenticate(ctx context.Context, options arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error)
}

func chooseLoginController(cluster *arvados.Cluster, railsProxy *railsProxy) loginController {
	wantGoogle := cluster.Login.GoogleClientID != ""
	wantSSO := cluster.Login.ProviderAppID != ""
	wantPAM := cluster.Login.PAM
	switch {
	case wantGoogle && !wantSSO && !wantPAM:
		return &googleLoginController{Cluster: cluster, RailsProxy: railsProxy}
	case !wantGoogle && wantSSO && !wantPAM:
		return &ssoLoginController{railsProxy}
	case !wantGoogle && !wantSSO && wantPAM:
		return &pamLoginController{Cluster: cluster, RailsProxy: railsProxy}
	default:
		return errorLoginController{
			error: errors.New("configuration problem: exactly one of Login.GoogleClientID, Login.ProviderAppID, or Login.PAM must be configured"),
		}
	}
}

// Login and Logout are passed through to the wrapped railsProxy;
// UserAuthenticate is rejected.
type ssoLoginController struct{ *railsProxy }

func (ctrl *ssoLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return arvados.APIClientAuthorization{}, httpserver.ErrorWithStatus(errors.New("username/password authentication is not available"), http.StatusBadRequest)
}

type errorLoginController struct{ error }

func (ctrl errorLoginController) Login(context.Context, arvados.LoginOptions) (arvados.LoginResponse, error) {
	return arvados.LoginResponse{}, ctrl.error
}
func (ctrl errorLoginController) Logout(context.Context, arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return arvados.LogoutResponse{}, ctrl.error
}
func (ctrl errorLoginController) UserAuthenticate(context.Context, arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return arvados.APIClientAuthorization{}, ctrl.error
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
