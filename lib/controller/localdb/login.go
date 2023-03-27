// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

type loginController interface {
	Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error)
	Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error)
	UserAuthenticate(ctx context.Context, options arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error)
}

func chooseLoginController(cluster *arvados.Cluster, parent *Conn) loginController {
	wantGoogle := cluster.Login.Google.Enable
	wantOpenIDConnect := cluster.Login.OpenIDConnect.Enable
	wantPAM := cluster.Login.PAM.Enable
	wantLDAP := cluster.Login.LDAP.Enable
	wantTest := cluster.Login.Test.Enable
	wantLoginCluster := cluster.Login.LoginCluster != "" && cluster.Login.LoginCluster != cluster.ClusterID
	switch {
	case 1 != countTrue(wantGoogle, wantOpenIDConnect, wantPAM, wantLDAP, wantTest, wantLoginCluster):
		return errorLoginController{
			error: errors.New("configuration problem: exactly one of Login.Google, Login.OpenIDConnect, Login.PAM, Login.LDAP, Login.Test, or Login.LoginCluster must be set"),
		}
	case wantGoogle:
		return &oidcLoginController{
			Cluster:            cluster,
			Parent:             parent,
			Issuer:             "https://accounts.google.com",
			ClientID:           cluster.Login.Google.ClientID,
			ClientSecret:       cluster.Login.Google.ClientSecret,
			AuthParams:         cluster.Login.Google.AuthenticationRequestParameters,
			UseGooglePeopleAPI: cluster.Login.Google.AlternateEmailAddresses,
			EmailClaim:         "email",
			EmailVerifiedClaim: "email_verified",
		}
	case wantOpenIDConnect:
		return &oidcLoginController{
			Cluster:                cluster,
			Parent:                 parent,
			Issuer:                 cluster.Login.OpenIDConnect.Issuer,
			ClientID:               cluster.Login.OpenIDConnect.ClientID,
			ClientSecret:           cluster.Login.OpenIDConnect.ClientSecret,
			AuthParams:             cluster.Login.OpenIDConnect.AuthenticationRequestParameters,
			EmailClaim:             cluster.Login.OpenIDConnect.EmailClaim,
			EmailVerifiedClaim:     cluster.Login.OpenIDConnect.EmailVerifiedClaim,
			UsernameClaim:          cluster.Login.OpenIDConnect.UsernameClaim,
			AcceptAccessToken:      cluster.Login.OpenIDConnect.AcceptAccessToken,
			AcceptAccessTokenScope: cluster.Login.OpenIDConnect.AcceptAccessTokenScope,
		}
	case wantPAM:
		return &pamLoginController{Cluster: cluster, Parent: parent}
	case wantLDAP:
		return &ldapLoginController{Cluster: cluster, Parent: parent}
	case wantTest:
		return &testLoginController{Cluster: cluster, Parent: parent}
	case wantLoginCluster:
		return &federatedLoginController{Cluster: cluster}
	default:
		return errorLoginController{
			error: errors.New("BUG: missing case in login controller setup switch"),
		}
	}
}

func countTrue(vals ...bool) int {
	n := 0
	for _, val := range vals {
		if val {
			n++
		}
	}
	return n
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

type federatedLoginController struct {
	Cluster *arvados.Cluster
}

func (ctrl federatedLoginController) Login(context.Context, arvados.LoginOptions) (arvados.LoginResponse, error) {
	return arvados.LoginResponse{}, httpserver.ErrorWithStatus(errors.New("Should have been redirected to login cluster"), http.StatusBadRequest)
}
func (ctrl federatedLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return logout(ctx, ctrl.Cluster, opts)
}
func (ctrl federatedLoginController) UserAuthenticate(context.Context, arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return arvados.APIClientAuthorization{}, httpserver.ErrorWithStatus(errors.New("username/password authentication is not available"), http.StatusBadRequest)
}

func (conn *Conn) CreateAPIClientAuthorization(ctx context.Context, rootToken string, authinfo rpc.UserSessionAuthInfo) (resp arvados.APIClientAuthorization, err error) {
	if rootToken == "" {
		return arvados.APIClientAuthorization{}, errors.New("configuration error: empty SystemRootToken")
	}
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{rootToken}})
	newsession, err := conn.railsProxy.UserSessionCreate(ctxRoot, rpc.UserSessionCreateOptions{
		// Send a fake ReturnTo value instead of the caller's
		// opts.ReturnTo. We won't follow the resulting
		// redirect target anyway.
		ReturnTo: ",https://controller.api.client.invalid",
		AuthInfo: authinfo,
	})
	if err != nil {
		return
	}
	target, err := url.Parse(newsession.RedirectLocation)
	if err != nil {
		return
	}
	token := target.Query().Get("api_token")
	tx, err := ctrlctx.CurrentTx(ctx)
	if err != nil {
		return
	}
	tokensecret := token
	if strings.Contains(token, "/") {
		tokenparts := strings.Split(token, "/")
		if len(tokenparts) >= 3 {
			tokensecret = tokenparts[2]
		}
	}
	var exp sql.NullTime
	var scopes []byte
	err = tx.QueryRowxContext(ctx, "select uuid, api_token, expires_at, scopes from api_client_authorizations where api_token=$1", tokensecret).Scan(&resp.UUID, &resp.APIToken, &exp, &scopes)
	if err != nil {
		return
	}
	resp.ExpiresAt = exp.Time
	if len(scopes) > 0 {
		err = json.Unmarshal(scopes, &resp.Scopes)
		if err != nil {
			return resp, fmt.Errorf("unmarshal scopes: %s", err)
		}
	}
	return
}

func validateLoginRedirectTarget(cluster *arvados.Cluster, returnTo string) error {
	u, err := url.Parse(returnTo)
	if err != nil {
		return err
	}
	u, err = u.Parse("/")
	if err != nil {
		return err
	}
	target := origin(*u)
	for trusted := range cluster.Login.TrustedClients {
		if origin(url.URL(trusted)) == target {
			return nil
		}
	}
	if target == origin(url.URL(cluster.Services.Workbench1.ExternalURL)) ||
		target == origin(url.URL(cluster.Services.Workbench2.ExternalURL)) {
		return nil
	}
	if cluster.Login.TrustPrivateNetworks {
		if u.Hostname() == "localhost" {
			return nil
		}
		if ip := net.ParseIP(u.Hostname()); len(ip) > 0 {
			for _, n := range privateNetworks {
				if n.Contains(ip) {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("requesting site is not listed in TrustedClients config")
}

// origin returns the canonical origin of a URL, e.g.,
// origin("https://example:443/foo") returns "https://example/"
func origin(u url.URL) string {
	origin := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   "/",
	}
	if origin.Port() == "80" && origin.Scheme == "http" {
		origin.Host = origin.Hostname()
	} else if origin.Port() == "443" && origin.Scheme == "https" {
		origin.Host = origin.Hostname()
	}
	return origin.String()
}
