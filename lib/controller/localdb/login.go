// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"git.curoverse.com/arvados.git/lib/controller/rpc"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

type googleLoginController struct {
	issuer   string // override OIDC issuer URL (normally https://accounts.google.com) for testing
	provider *oidc.Provider
	mu       sync.Mutex
}

func (ctrl *googleLoginController) getProvider() (*oidc.Provider, error) {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	if ctrl.provider == nil {
		issuer := ctrl.issuer
		if issuer == "" {
			issuer = "https://accounts.google.com"
		}
		provider, err := oidc.NewProvider(context.Background(), issuer)
		if err != nil {
			return nil, err
		}
		ctrl.provider = provider
	}
	return ctrl.provider, nil
}

func (ctrl *googleLoginController) Login(ctx context.Context, cluster *arvados.Cluster, railsproxy *railsProxy, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	provider, err := ctrl.getProvider()
	if err != nil {
		return ctrl.loginError(fmt.Errorf("error setting up OpenID Connect provider: %s", err))
	}
	redirURL, err := (*url.URL)(&cluster.Services.Controller.ExternalURL).Parse("/login")
	if err != nil {
		return ctrl.loginError(fmt.Errorf("error making redirect URL: %s", err))
	}
	conf := &oauth2.Config{
		ClientID:     cluster.Login.GoogleClientID,
		ClientSecret: cluster.Login.GoogleClientSecret,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		RedirectURL:  redirURL.String(),
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: conf.ClientID,
	})
	if opts.State == "" {
		// Initiate Google sign-in.
		if opts.ReturnTo == "" {
			return ctrl.loginError(errors.New("missing return_to parameter"))
		}
		me := url.URL(cluster.Services.Controller.ExternalURL)
		callback, err := me.Parse("/" + arvados.EndpointLogin.Path)
		if err != nil {
			return ctrl.loginError(err)
		}
		conf.RedirectURL = callback.String()
		state := ctrl.newOAuth2State([]byte(cluster.SystemRootToken), opts.Remote, opts.ReturnTo)
		return arvados.LoginResponse{
			RedirectLocation: conf.AuthCodeURL(state.String(),
				// prompt=select_account tells Google
				// to show the "choose which Google
				// account" page, even if the client
				// is currently logged in to exactly
				// one Google account.
				oauth2.SetAuthURLParam("prompt", "select_account")),
		}, nil
	} else {
		// Callback after Google sign-in.
		state := ctrl.parseOAuth2State(opts.State)
		if !state.verify([]byte(cluster.SystemRootToken)) {
			return ctrl.loginError(errors.New("invalid OAuth2 state"))
		}
		oauth2Token, err := conf.Exchange(ctx, opts.Code)
		if err != nil {
			return ctrl.loginError(fmt.Errorf("error in OAuth2 exchange: %s", err))
		}
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			return ctrl.loginError(errors.New("error in OAuth2 exchange: no ID token in OAuth2 token"))
		}
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			return ctrl.loginError(fmt.Errorf("error verifying ID token: %s", err))
		}
		var claims struct {
			Email    string `json:"email"`
			Verified bool   `json:"email_verified"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return ctrl.loginError(fmt.Errorf("error extracting claims from ID token: %s", err))
		}
		if !claims.Verified {
			return ctrl.loginError(errors.New("cannot authenticate using an unverified email address"))
		}
		ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{cluster.SystemRootToken}})
		return railsproxy.UserSessionCreate(ctxRoot, rpc.UserSessionCreateOptions{
			ReturnTo: state.Remote + "," + state.ReturnTo,
			AuthInfo: map[string]interface{}{
				"email": claims.Email,
			},
		})
	}
}

func (ctrl *googleLoginController) loginError(sendError error) (resp arvados.LoginResponse, err error) {
	tmpl, err := template.New("error").Parse(`<h2>Login error:</h2><p>{{.}}</p>`)
	if err != nil {
		return
	}
	err = tmpl.Execute(&resp.HTML, sendError.Error())
	return
}

func (ctrl *googleLoginController) newOAuth2State(key []byte, remote, returnTo string) oauth2State {
	s := oauth2State{
		Time:     time.Now().Unix(),
		Remote:   remote,
		ReturnTo: returnTo,
	}
	s.HMAC = s.computeHMAC(key)
	return s
}

type oauth2State struct {
	HMAC     []byte // hash of other fields; see computeHMAC()
	Time     int64  // creation time (unix timestamp)
	Remote   string // remote cluster if requesting a salted token, otherwise blank
	ReturnTo string // redirect target
}

func (ctrl *googleLoginController) parseOAuth2State(encoded string) (s oauth2State) {
	// Errors are not checked. If decoding/parsing fails, the
	// token will be rejected by verify().
	decoded, _ := base64.RawURLEncoding.DecodeString(encoded)
	f := strings.Split(string(decoded), "\n")
	if len(f) != 4 {
		return
	}
	fmt.Sscanf(f[0], "%x", &s.HMAC)
	fmt.Sscanf(f[1], "%x", &s.Time)
	fmt.Sscanf(f[2], "%s", &s.Remote)
	fmt.Sscanf(f[3], "%s", &s.ReturnTo)
	return
}

func (s oauth2State) verify(key []byte) bool {
	if delta := time.Now().Unix() - s.Time; delta < 0 || delta > 300 {
		return false
	}
	return hmac.Equal(s.computeHMAC(key), s.HMAC)
}

func (s oauth2State) String() string {
	var buf bytes.Buffer
	enc := base64.NewEncoder(base64.RawURLEncoding, &buf)
	fmt.Fprintf(enc, "%x\n%x\n%s\n%s", s.HMAC, s.Time, s.Remote, s.ReturnTo)
	enc.Close()
	return buf.String()
}

func (s oauth2State) computeHMAC(key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	fmt.Fprintf(mac, "%x %s %s", s.Time, s.Remote, s.ReturnTo)
	return mac.Sum(nil)
}
