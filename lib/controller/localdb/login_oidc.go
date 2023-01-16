// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"git.arvados.org/arvados.git/lib/controller/api"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/coreos/go-oidc"
	lru "github.com/hashicorp/golang-lru"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
	"gopkg.in/square/go-jose.v2/jwt"
)

var (
	tokenCacheSize        = 1000
	tokenCacheNegativeTTL = time.Minute * 5
	tokenCacheTTL         = time.Minute * 10
	tokenCacheRaceWindow  = time.Minute
	pqCodeUniqueViolation = pq.ErrorCode("23505")
)

type oidcLoginController struct {
	Cluster                *arvados.Cluster
	Parent                 *Conn
	Issuer                 string // OIDC issuer URL, e.g., "https://accounts.google.com"
	ClientID               string
	ClientSecret           string
	UseGooglePeopleAPI     bool              // Use Google People API to look up alternate email addresses
	EmailClaim             string            // OpenID claim to use as email address; typically "email"
	EmailVerifiedClaim     string            // If non-empty, ensure claim value is true before accepting EmailClaim; typically "email_verified"
	UsernameClaim          string            // If non-empty, use as preferred username
	AcceptAccessToken      bool              // Accept access tokens as API tokens
	AcceptAccessTokenScope string            // If non-empty, don't accept access tokens as API tokens unless they contain this scope
	AuthParams             map[string]string // Additional parameters to pass with authentication request

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
	return logout(ctx, ctrl.Cluster, opts)
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
		if err := validateLoginRedirectTarget(ctrl.Parent.cluster, opts.ReturnTo); err != nil {
			return loginError(fmt.Errorf("invalid return_to parameter: %s", err))
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
	ctxlog.FromContext(ctx).WithField("oauth2Token", oauth2Token).Debug("oauth2 exchange succeeded")
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return loginError(errors.New("error in OAuth2 exchange: no ID token in OAuth2 token"))
	}
	ctxlog.FromContext(ctx).WithField("rawIDToken", rawIDToken).Debug("oauth2Token provided ID token")
	idToken, err := ctrl.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return loginError(fmt.Errorf("error verifying ID token: %s", err))
	}
	authinfo, err := ctrl.getAuthInfo(ctx, oauth2Token, idToken)
	if err != nil {
		return loginError(err)
	}
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{ctrl.Cluster.SystemRootToken}})
	return ctrl.Parent.UserSessionCreate(ctxRoot, rpc.UserSessionCreateOptions{
		ReturnTo: state.Remote + "," + state.ReturnTo,
		AuthInfo: *authinfo,
	})
}

func (ctrl *oidcLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return arvados.APIClientAuthorization{}, httpserver.ErrorWithStatus(errors.New("username/password authentication is not available"), http.StatusBadRequest)
}

// claimser can decode arbitrary claims into a map. Implemented by
// *oauth2.IDToken and *oauth2.UserInfo.
type claimser interface {
	Claims(interface{}) error
}

// Use a person's token to get all of their email addresses, with the
// primary address at index 0. The provided defaultAddr is always
// included in the returned slice, and is used as the primary if the
// Google API does not indicate one.
func (ctrl *oidcLoginController) getAuthInfo(ctx context.Context, token *oauth2.Token, claimser claimser) (*rpc.UserSessionAuthInfo, error) {
	var ret rpc.UserSessionAuthInfo
	defer ctxlog.FromContext(ctx).WithField("ret", &ret).Debug("getAuthInfo returned")

	var claims map[string]interface{}
	if err := claimser.Claims(&claims); err != nil {
		return nil, fmt.Errorf("error extracting claims from token: %s", err)
	} else if verified, _ := claims[ctrl.EmailVerifiedClaim].(bool); verified || ctrl.EmailVerifiedClaim == "" {
		// Fall back to this info if the People API call
		// (below) doesn't return a primary && verified email.
		givenName, _ := claims["given_name"].(string)
		familyName, _ := claims["family_name"].(string)
		if givenName != "" && familyName != "" {
			ret.FirstName = givenName
			ret.LastName = familyName
		} else {
			name, _ := claims["name"].(string)
			if names := strings.Fields(strings.TrimSpace(name)); len(names) > 1 {
				ret.FirstName = strings.Join(names[0:len(names)-1], " ")
				ret.LastName = names[len(names)-1]
			} else if len(names) > 0 {
				ret.FirstName = names[0]
			}
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

func OIDCAccessTokenAuthorizer(cluster *arvados.Cluster, getdb func(context.Context) (*sqlx.DB, error)) *oidcTokenAuthorizer {
	// We want ctrl to be nil if the chosen controller is not a
	// *oidcLoginController, so we can ignore the 2nd return value
	// of this type cast.
	ctrl, _ := NewConn(cluster).loginController.(*oidcLoginController)
	cache, err := lru.New2Q(tokenCacheSize)
	if err != nil {
		panic(err)
	}
	return &oidcTokenAuthorizer{
		ctrl:  ctrl,
		getdb: getdb,
		cache: cache,
	}
}

type oidcTokenAuthorizer struct {
	ctrl  *oidcLoginController
	getdb func(context.Context) (*sqlx.DB, error)
	cache *lru.TwoQueueCache
}

func (ta *oidcTokenAuthorizer) Middleware(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if ta.ctrl == nil {
		// Not using a compatible (OIDC) login controller.
	} else if authhdr := strings.Split(r.Header.Get("Authorization"), " "); len(authhdr) > 1 && (authhdr[0] == "OAuth2" || authhdr[0] == "Bearer") {
		err := ta.registerToken(r.Context(), authhdr[1])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	next.ServeHTTP(w, r)
}

func (ta *oidcTokenAuthorizer) WrapCalls(origFunc api.RoutableFunc) api.RoutableFunc {
	if ta.ctrl == nil {
		// Not using a compatible (OIDC) login controller.
		return origFunc
	}
	return func(ctx context.Context, opts interface{}) (_ interface{}, err error) {
		creds, ok := auth.FromContext(ctx)
		if !ok {
			return origFunc(ctx, opts)
		}
		// Check each token in the incoming request. If any
		// are valid OAuth2 access tokens, insert/update them
		// in the database so RailsAPI's auth code accepts
		// them.
		for _, tok := range creds.Tokens {
			err = ta.registerToken(ctx, tok)
			if err != nil {
				return nil, err
			}
		}
		return origFunc(ctx, opts)
	}
}

// Matches error from oidc UserInfo() when receiving HTTP status 5xx
var re5xxError = regexp.MustCompile(`^5\d\d `)

// registerToken checks whether tok is a valid OIDC Access Token and,
// if so, ensures that an api_client_authorizations row exists so that
// RailsAPI will accept it as an Arvados token.
func (ta *oidcTokenAuthorizer) registerToken(ctx context.Context, tok string) error {
	if tok == ta.ctrl.Cluster.SystemRootToken || strings.HasPrefix(tok, "v2/") {
		return nil
	}
	if cached, hit := ta.cache.Get(tok); !hit {
		// Fall through to database and OIDC provider checks
		// below
	} else if exp, ok := cached.(time.Time); ok {
		// cached negative result (value is expiry time)
		if time.Now().Before(exp) {
			return nil
		}
		ta.cache.Remove(tok)
	} else {
		// cached positive result
		aca := cached.(arvados.APIClientAuthorization)
		var expiring bool
		if !aca.ExpiresAt.IsZero() {
			t := aca.ExpiresAt
			expiring = t.Before(time.Now().Add(time.Minute))
		}
		if !expiring {
			return nil
		}
	}

	db, err := ta.getdb(ctx)
	if err != nil {
		return err
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ctx = ctrlctx.NewWithTransaction(ctx, tx)

	// We use hmac-sha256(accesstoken,systemroottoken) as the
	// secret part of our own token, and avoid storing the auth
	// provider's real secret in our database.
	mac := hmac.New(sha256.New, []byte(ta.ctrl.Cluster.SystemRootToken))
	io.WriteString(mac, tok)
	hmac := fmt.Sprintf("%x", mac.Sum(nil))

	var expiring bool
	err = tx.QueryRowContext(ctx, `select (expires_at is not null and expires_at - interval '1 minute' <= current_timestamp at time zone 'UTC') from api_client_authorizations where api_token=$1`, hmac).Scan(&expiring)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("database error while checking token: %w", err)
	} else if err == nil && !expiring {
		// Token is already in the database as an Arvados
		// token, and isn't about to expire, so we can pass it
		// through to RailsAPI etc. regardless of whether it's
		// an OIDC access token.
		return nil
	}
	updating := err == nil

	// Check whether the token is a valid OIDC access token. If
	// so, swap it out for an Arvados token (creating/updating an
	// api_client_authorizations row if needed) which downstream
	// server components will accept.
	err = ta.ctrl.setup()
	if err != nil {
		return fmt.Errorf("error setting up OpenID Connect provider: %s", err)
	}
	if ok, err := ta.checkAccessTokenScope(ctx, tok); err != nil || !ok {
		// Note checkAccessTokenScope logs any interesting errors
		ta.cache.Add(tok, time.Now().Add(tokenCacheNegativeTTL))
		return err
	}
	oauth2Token := &oauth2.Token{
		AccessToken: tok,
	}
	userinfo, err := ta.ctrl.provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		if neterr := net.Error(nil); errors.As(err, &neterr) || re5xxError.MatchString(err.Error()) {
			// If this token is in fact a valid OIDC
			// token, but we failed to validate it here
			// because of a network problem or internal
			// server error, we error out now with a 5xx
			// error, indicating to the client that they
			// can try again.  If we didn't error out now,
			// the unrecognized token would eventually
			// cause a 401 error further down the stack,
			// which the caller would interpret as an
			// unrecoverable failure.
			ctxlog.FromContext(ctx).WithError(err).Debugf("treating OIDC UserInfo lookup error type %T as transient; failing request instead of forwarding token blindly", err)
			return err
		}
		ctxlog.FromContext(ctx).WithError(err).WithField("HMAC", hmac).Debug("UserInfo failed (not an OIDC token?), caching negative result")
		ta.cache.Add(tok, time.Now().Add(tokenCacheNegativeTTL))
		return nil
	}
	ctxlog.FromContext(ctx).WithField("userinfo", userinfo).Debug("(*oidcTokenAuthorizer)registerToken: got userinfo")
	authinfo, err := ta.ctrl.getAuthInfo(ctx, oauth2Token, userinfo)
	if err != nil {
		return err
	}

	// Expiry time for our token is one minute longer than our
	// cache TTL, so we don't pass it through to RailsAPI just as
	// it's expiring.
	exp := time.Now().UTC().Add(tokenCacheTTL + tokenCacheRaceWindow)

	if updating {
		_, err = tx.ExecContext(ctx, `update api_client_authorizations set expires_at=$1 where api_token=$2`, exp, hmac)
		if err != nil {
			return fmt.Errorf("error updating token expiry time: %w", err)
		}
		ctxlog.FromContext(ctx).WithField("HMAC", hmac).Debug("(*oidcTokenAuthorizer)registerToken: updated api_client_authorizations row")
	} else {
		aca, err := ta.ctrl.Parent.CreateAPIClientAuthorization(ctx, ta.ctrl.Cluster.SystemRootToken, *authinfo)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `savepoint upd`)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `update api_client_authorizations set api_token=$1, expires_at=$2 where uuid=$3`, hmac, exp, aca.UUID)
		if e, ok := err.(*pq.Error); ok && e.Code == pqCodeUniqueViolation {
			// unique_violation, given that the above
			// query did not find a row with matching
			// api_token, means another thread/process
			// also received this same token and won the
			// race to insert it -- in which case this
			// thread doesn't need to update the database.
			// Discard the redundant row.
			_, err = tx.ExecContext(ctx, `rollback to savepoint upd`)
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `delete from api_client_authorizations where uuid=$1`, aca.UUID)
			if err != nil {
				return err
			}
			ctxlog.FromContext(ctx).WithField("HMAC", hmac).Debug("(*oidcTokenAuthorizer)registerToken: api_client_authorizations row inserted by another thread")
		} else if err != nil {
			ctxlog.FromContext(ctx).Errorf("%#v", err)
			return fmt.Errorf("error adding OIDC access token to database: %w", err)
		} else {
			ctxlog.FromContext(ctx).WithFields(logrus.Fields{"UUID": aca.UUID, "HMAC": hmac}).Debug("(*oidcTokenAuthorizer)registerToken: inserted api_client_authorizations row")
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	ta.cache.Add(tok, arvados.APIClientAuthorization{ExpiresAt: exp})
	return nil
}

// Check that the provided access token is a JWT with the required
// scope. If it is a valid JWT but missing the required scope, we
// return a 403 error, otherwise true (acceptable as an API token) or
// false (pass through unmodified).
//
// Return false if configured not to accept access tokens at all.
//
// Note we don't check signature or expiry here. We are relying on the
// caller to verify those separately (e.g., by calling the UserInfo
// endpoint).
func (ta *oidcTokenAuthorizer) checkAccessTokenScope(ctx context.Context, tok string) (bool, error) {
	if !ta.ctrl.AcceptAccessToken {
		return false, nil
	} else if ta.ctrl.AcceptAccessTokenScope == "" {
		return true, nil
	}
	var claims struct {
		Scope string `json:"scope"`
	}
	if t, err := jwt.ParseSigned(tok); err != nil {
		ctxlog.FromContext(ctx).WithError(err).Debug("error parsing jwt")
		return false, nil
	} else if err = t.UnsafeClaimsWithoutVerification(&claims); err != nil {
		ctxlog.FromContext(ctx).WithError(err).Debug("error extracting jwt claims")
		return false, nil
	}
	for _, s := range strings.Split(claims.Scope, " ") {
		if s == ta.ctrl.AcceptAccessTokenScope {
			return true, nil
		}
	}
	ctxlog.FromContext(ctx).WithFields(logrus.Fields{"have": claims.Scope, "need": ta.ctrl.AcceptAccessTokenScope}).Info("unacceptable access token scope")
	return false, httpserver.ErrorWithStatus(errors.New("unacceptable access token scope"), http.StatusUnauthorized)
}
