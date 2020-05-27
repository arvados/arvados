// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

type loginController interface {
	Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error)
	Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error)
	UserAuthenticate(ctx context.Context, options arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error)
}

func chooseLoginController(cluster *arvados.Cluster, railsProxy *railsProxy) loginController {
	wantGoogle := cluster.Login.Google.Enable
	wantSSO := cluster.Login.SSO.Enable
	wantPAM := cluster.Login.PAM.Enable
	wantLDAP := cluster.Login.LDAP.Enable
	switch {
	case wantGoogle && !wantSSO && !wantPAM && !wantLDAP:
		return &oidcLoginController{Cluster: cluster, RailsProxy: railsProxy, Issuer: "https://accounts.google.com", GoogleAPI: true}
	case !wantGoogle && wantSSO && !wantPAM && !wantLDAP:
		return &ssoLoginController{railsProxy}
	case !wantGoogle && !wantSSO && wantPAM && !wantLDAP:
		return &pamLoginController{Cluster: cluster, RailsProxy: railsProxy}
	case !wantGoogle && !wantSSO && !wantPAM && wantLDAP:
		return &ldapLoginController{Cluster: cluster, RailsProxy: railsProxy}
	default:
		return errorLoginController{
			error: errors.New("configuration problem: exactly one of Login.Google, Login.SSO, Login.PAM, and Login.LDAP must be enabled"),
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

func createAPIClientAuthorization(ctx context.Context, conn *rpc.Conn, rootToken string, authinfo rpc.UserSessionAuthInfo) (arvados.APIClientAuthorization, error) {
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{rootToken}})
	resp, err := conn.UserSessionCreate(ctxRoot, rpc.UserSessionCreateOptions{
		// Send a fake ReturnTo value instead of the caller's
		// opts.ReturnTo. We won't follow the resulting
		// redirect target anyway.
		ReturnTo: ",https://none.invalid",
		AuthInfo: authinfo,
	})
	if err != nil {
		return arvados.APIClientAuthorization{}, err
	}
	target, err := url.Parse(resp.RedirectLocation)
	if err != nil {
		return arvados.APIClientAuthorization{}, err
	}
	token := target.Query().Get("api_token")
	return conn.APIClientAuthorizationCurrent(auth.NewContext(ctx, auth.NewCredentials(token)), arvados.GetOptions{})
}
