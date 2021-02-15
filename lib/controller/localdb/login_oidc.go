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
	"net/http"
	"net/url"
	"strings"
	"sync"
	"text/template"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

type oidcLoginController struct {
	Cluster            *arvados.Cluster
	RailsProxy         *railsProxy
	Issuer             string // OIDC issuer URL, e.g., "https://accounts.google.com"
	ClientID           string
	ClientSecret       string
	UseGooglePeopleAPI bool              // Use Google People API to look up alternate email addresses
	EmailClaim         string            // OpenID claim to use as email address; typically "email"
	EmailVerifiedClaim string            // If non-empty, ensure claim value is true before accepting EmailClaim; typically "email_verified"
	UsernameClaim      string            // If non-empty, use as preferred username
	AuthParams         map[string]string // Additional parameters to pass with authentication request

	// override Google People API base URL for testing purposes
	// (normally empty, set by google pkg to
	// https://people.googleapis.com/)
	peopleAPIBasePath string

	provider   *oidc.Provider        // initialized by setup()
	oauth2conf *oauth2.Config        // initialized by setup()
	verifier   *oidc.IDTokenVerifier // initialized by setup()
	mu         sync.Mutex            // protects setup()
}

// Initialize ctrl.provider and ctrl.oauth2conf.
func (ctrl *oidcLoginController) setup() error {
	ctrl.mu.Lock()
	defer ctrl.mu.Unlock()
	if ctrl.provider != nil {
		// already set up
		return nil
	}
	redirURL, err := (*url.URL)(&ctrl.Cluster.Services.Controller.ExternalURL).Parse("/" + arvados.EndpointLogin.Path)
	if err != nil {
		return fmt.Errorf("error making redirect URL: %s", err)
	}
	provider, err := oidc.NewProvider(context.Background(), ctrl.Issuer)
	if err != nil {
		return err
	}
	ctrl.oauth2conf = &oauth2.Config{
		ClientID:     ctrl.ClientID,
		ClientSecret: ctrl.ClientSecret,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		RedirectURL:  redirURL.String(),
	}
	ctrl.verifier = provider.Verifier(&oidc.Config{
		ClientID: ctrl.ClientID,
	})
	ctrl.provider = provider
	return nil
}

func (ctrl *oidcLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return noopLogout(ctrl.Cluster, opts)
}

func (ctrl *oidcLoginController) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	err := ctrl.setup()
	if err != nil {
		return loginError(fmt.Errorf("error setting up OpenID Connect provider: %s", err))
	}
	if opts.State == "" {
		// Initiate OIDC sign-in.
		if opts.ReturnTo == "" {
			return loginError(errors.New("missing return_to parameter"))
		}
		state := ctrl.newOAuth2State([]byte(ctrl.Cluster.SystemRootToken), opts.Remote, opts.ReturnTo)
		var authparams []oauth2.AuthCodeOption
		for k, v := range ctrl.AuthParams {
			authparams = append(authparams, oauth2.SetAuthURLParam(k, v))
		}
		return arvados.LoginResponse{
			RedirectLocation: ctrl.oauth2conf.AuthCodeURL(state.String(), authparams...),
		}, nil
	}
	// Callback after OIDC sign-in.
	state := ctrl.parseOAuth2State(opts.State)
	if !state.verify([]byte(ctrl.Cluster.SystemRootToken)) {
		return loginError(errors.New("invalid OAuth2 state"))
	}
	oauth2Token, err := ctrl.oauth2conf.Exchange(ctx, opts.Code)
	if err != nil {
		return loginError(fmt.Errorf("error in OAuth2 exchange: %s", err))
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return loginError(errors.New("error in OAuth2 exchange: no ID token in OAuth2 token"))
	}
	idToken, err := ctrl.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return loginError(fmt.Errorf("error verifying ID token: %s", err))
	}
	authinfo, err := ctrl.getAuthInfo(ctx, oauth2Token, idToken)
	if err != nil {
		return loginError(err)
	}
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{ctrl.Cluster.SystemRootToken}})
	return ctrl.RailsProxy.UserSessionCreate(ctxRoot, rpc.UserSessionCreateOptions{
		ReturnTo: state.Remote + "," + state.ReturnTo,
		AuthInfo: *authinfo,
	})
}

func (ctrl *oidcLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return arvados.APIClientAuthorization{}, httpserver.ErrorWithStatus(errors.New("username/password authentication is not available"), http.StatusBadRequest)
}

// Use a person's token to get all of their email addresses, with the
// primary address at index 0. The provided defaultAddr is always
// included in the returned slice, and is used as the primary if the
// Google API does not indicate one.
func (ctrl *oidcLoginController) getAuthInfo(ctx context.Context, token *oauth2.Token, idToken *oidc.IDToken) (*rpc.UserSessionAuthInfo, error) {
	var ret rpc.UserSessionAuthInfo
	defer ctxlog.FromContext(ctx).WithField("ret", &ret).Debug("getAuthInfo returned")

	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("error extracting claims from ID token: %s", err)
	} else if verified, _ := claims[ctrl.EmailVerifiedClaim].(bool); verified || ctrl.EmailVerifiedClaim == "" {
		// Fall back to this info if the People API call
		// (below) doesn't return a primary && verified email.
		name, _ := claims["name"].(string)
		if names := strings.Fields(strings.TrimSpace(name)); len(names) > 1 {
			ret.FirstName = strings.Join(names[0:len(names)-1], " ")
			ret.LastName = names[len(names)-1]
		} else if len(names) > 0 {
			ret.FirstName = names[0]
		}
		ret.Email, _ = claims[ctrl.EmailClaim].(string)
	}

	if ctrl.UsernameClaim != "" {
		ret.Username, _ = claims[ctrl.UsernameClaim].(string)
	}

	if !ctrl.UseGooglePeopleAPI {
		if ret.Email == "" {
			return nil, fmt.Errorf("cannot log in with unverified email address %q", claims[ctrl.EmailClaim])
		}
		return &ret, nil
	}

	svc, err := people.NewService(ctx, option.WithTokenSource(ctrl.oauth2conf.TokenSource(ctx, token)), option.WithScopes(people.UserEmailsReadScope))
	if err != nil {
		return nil, fmt.Errorf("error setting up People API: %s", err)
	}
	if p := ctrl.peopleAPIBasePath; p != "" {
		// Override normal API endpoint (for testing)
		svc.BasePath = p
	}
	person, err := people.NewPeopleService(svc).Get("people/me").PersonFields("emailAddresses,names").Do()
	if err != nil {
		if strings.Contains(err.Error(), "Error 403") && strings.Contains(err.Error(), "accessNotConfigured") {
			// Log the original API error, but display
			// only the "fix config" advice to the user.
			ctxlog.FromContext(ctx).WithError(err).WithField("email", ret.Email).Error("People API is not enabled")
			return nil, errors.New("configuration error: Login.GoogleAlternateEmailAddresses is true, but Google People API is not enabled")
		}
		return nil, fmt.Errorf("error getting profile info from People API: %s", err)
	}

	// The given/family names returned by the People API and
	// flagged as "primary" (if any) take precedence over the
	// split-by-whitespace result from above.
	for _, name := range person.Names {
		if name.Metadata != nil && name.Metadata.Primary {
			ret.FirstName = name.GivenName
			ret.LastName = name.FamilyName
			break
		}
	}

	altEmails := map[string]bool{}
	if ret.Email != "" {
		altEmails[ret.Email] = true
	}
	for _, ea := range person.EmailAddresses {
		if ea.Metadata == nil || !ea.Metadata.Verified {
			ctxlog.FromContext(ctx).WithField("address", ea.Value).Info("skipping unverified email address")
			continue
		}
		altEmails[ea.Value] = true
		if ea.Metadata.Primary || ret.Email == "" {
			ret.Email = ea.Value
		}
	}
	if len(altEmails) == 0 {
		return nil, errors.New("cannot log in without a verified email address")
	}
	for ae := range altEmails {
		if ae == ret.Email {
			continue
		}
		ret.AlternateEmails = append(ret.AlternateEmails, ae)
		if ret.Username == "" {
			i := strings.Index(ae, "@")
			if i > 0 && strings.ToLower(ae[i+1:]) == strings.ToLower(ctrl.Cluster.Users.PreferDomainForUsername) {
				ret.Username = strings.SplitN(ae[:i], "+", 2)[0]
			}
		}
	}
	return &ret, nil
}

func loginError(sendError error) (resp arvados.LoginResponse, err error) {
	tmpl, err := template.New("error").Parse(`<h2>Login error:</h2><p>{{.}}</p>`)
	if err != nil {
		return
	}
	err = tmpl.Execute(&resp.HTML, sendError.Error())
	return
}

func (ctrl *oidcLoginController) newOAuth2State(key []byte, remote, returnTo string) oauth2State {
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

func (ctrl *oidcLoginController) parseOAuth2State(encoded string) (s oauth2State) {
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
