// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"github.com/jmoiron/sqlx"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&OIDCLoginSuite{})

type OIDCLoginSuite struct {
	localdbSuite
	trustedURL   *arvados.URL
	fakeProvider *arvadostest.OIDCProvider
}

func (s *OIDCLoginSuite) SetUpTest(c *check.C) {
	s.trustedURL = &arvados.URL{Scheme: "https", Host: "app.example.com:443", Path: "/"}

	s.fakeProvider = arvadostest.NewOIDCProvider(c)
	s.fakeProvider.AuthEmail = "active-user@arvados.local"
	s.fakeProvider.AuthEmailVerified = true
	s.fakeProvider.AuthName = "Fake User Name"
	s.fakeProvider.AuthGivenName = "Fake"
	s.fakeProvider.AuthFamilyName = "User Name"
	s.fakeProvider.ValidCode = fmt.Sprintf("abcdefgh-%d", time.Now().Unix())
	s.fakeProvider.PeopleAPIResponse = map[string]interface{}{}

	s.localdbSuite.SetUpTest(c)

	s.cluster.Login.Test.Enable = false
	s.cluster.Login.Google.Enable = true
	s.cluster.Login.Google.ClientID = "test%client$id"
	s.cluster.Login.Google.ClientSecret = "test#client/secret"
	s.cluster.Login.TrustedClients = map[arvados.URL]struct{}{*s.trustedURL: {}}
	s.cluster.Users.PreferDomainForUsername = "PreferDomainForUsername.example.com"
	s.fakeProvider.ValidClientID = "test%client$id"
	s.fakeProvider.ValidClientSecret = "test#client/secret"

	s.localdb = NewConn(s.ctx, s.cluster, (&ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}).GetDB)
	c.Assert(s.localdb.loginController, check.FitsTypeOf, (*oidcLoginController)(nil))
	s.localdb.loginController.(*oidcLoginController).Issuer = s.fakeProvider.Issuer.URL
	s.localdb.loginController.(*oidcLoginController).peopleAPIBasePath = s.fakeProvider.PeopleAPI.URL

	*s.localdb.railsProxy = *rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)
}

func (s *OIDCLoginSuite) TestGoogleLogout(c *check.C) {
	s.cluster.Login.TrustedClients[arvados.URL{Scheme: "https", Host: "foo.example", Path: "/"}] = struct{}{}
	s.cluster.Login.TrustPrivateNetworks = false

	resp, err := s.localdb.Logout(context.Background(), arvados.LogoutOptions{ReturnTo: "https://foo.example.com/bar"})
	c.Check(err, check.NotNil)
	c.Check(resp.RedirectLocation, check.Equals, "")

	resp, err = s.localdb.Logout(context.Background(), arvados.LogoutOptions{ReturnTo: "https://127.0.0.1/bar"})
	c.Check(err, check.NotNil)
	c.Check(resp.RedirectLocation, check.Equals, "")

	resp, err = s.localdb.Logout(context.Background(), arvados.LogoutOptions{ReturnTo: "https://foo.example/bar"})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "https://foo.example/bar")

	s.cluster.Login.TrustPrivateNetworks = true

	resp, err = s.localdb.Logout(context.Background(), arvados.LogoutOptions{ReturnTo: "https://192.168.1.1/bar"})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "https://192.168.1.1/bar")
}

func (s *OIDCLoginSuite) TestGoogleLogin_Start_Bogus(c *check.C) {
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `.*missing return_to parameter.*`)
}

func (s *OIDCLoginSuite) TestGoogleLogin_Start(c *check.C) {
	for _, remote := range []string{"", "zzzzz"} {
		resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{Remote: remote, ReturnTo: "https://app.example.com/foo?bar"})
		c.Check(err, check.IsNil)
		target, err := url.Parse(resp.RedirectLocation)
		c.Check(err, check.IsNil)
		issuerURL, _ := url.Parse(s.fakeProvider.Issuer.URL)
		c.Check(target.Host, check.Equals, issuerURL.Host)
		q := target.Query()
		c.Check(q.Get("client_id"), check.Equals, "test%client$id")
		state := s.localdb.loginController.(*oidcLoginController).parseOAuth2State(q.Get("state"))
		c.Check(state.verify([]byte(s.cluster.SystemRootToken)), check.Equals, true)
		c.Check(state.Time, check.Not(check.Equals), 0)
		c.Check(state.Remote, check.Equals, remote)
		c.Check(state.ReturnTo, check.Equals, "https://app.example.com/foo?bar")
	}
}

func (s *OIDCLoginSuite) TestGoogleLogin_UnknownClient(c *check.C) {
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{ReturnTo: "https://bad-app.example.com/foo?bar"})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*requesting site is not listed in TrustedClients.*`)
}

func (s *OIDCLoginSuite) TestGoogleLogin_InvalidCode(c *check.C) {
	state := s.startLogin(c)
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  "first-try-a-bogus-code",
		State: state,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*error in OAuth2 exchange.*cannot fetch token.*`)
}

func (s *OIDCLoginSuite) TestGoogleLogin_InvalidState(c *check.C) {
	s.startLogin(c)
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: "bogus-state",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*invalid OAuth2 state.*`)
}

func (s *OIDCLoginSuite) setupPeopleAPIError(c *check.C) {
	s.fakeProvider.PeopleAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, `Error 403: accessNotConfigured`)
	}))
	s.localdb.loginController.(*oidcLoginController).peopleAPIBasePath = s.fakeProvider.PeopleAPI.URL
}

func (s *OIDCLoginSuite) TestGoogleLogin_PeopleAPIDisabled(c *check.C) {
	s.localdb.loginController.(*oidcLoginController).UseGooglePeopleAPI = false
	s.fakeProvider.AuthEmail = "joe.smith@primary.example.com"
	s.setupPeopleAPIError(c)
	state := s.startLogin(c)
	_, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})
	c.Check(err, check.IsNil)
	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.Email, check.Equals, "joe.smith@primary.example.com")
}

func (s *OIDCLoginSuite) TestConfig(c *check.C) {
	s.cluster.Login.Google.Enable = false
	s.cluster.Login.OpenIDConnect.Enable = true
	s.cluster.Login.OpenIDConnect.Issuer = "https://accounts.example.com/"
	s.cluster.Login.OpenIDConnect.ClientID = "oidc-client-id"
	s.cluster.Login.OpenIDConnect.ClientSecret = "oidc-client-secret"
	s.cluster.Login.OpenIDConnect.AuthenticationRequestParameters = map[string]string{"testkey": "testvalue"}
	localdb := NewConn(context.Background(), s.cluster, (&ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}).GetDB)
	ctrl := localdb.loginController.(*oidcLoginController)
	c.Check(ctrl.Issuer, check.Equals, "https://accounts.example.com/")
	c.Check(ctrl.ClientID, check.Equals, "oidc-client-id")
	c.Check(ctrl.ClientSecret, check.Equals, "oidc-client-secret")
	c.Check(ctrl.UseGooglePeopleAPI, check.Equals, false)
	c.Check(ctrl.AuthParams["testkey"], check.Equals, "testvalue")

	for _, enableAltEmails := range []bool{false, true} {
		s.cluster.Login.OpenIDConnect.Enable = false
		s.cluster.Login.Google.Enable = true
		s.cluster.Login.Google.ClientID = "google-client-id"
		s.cluster.Login.Google.ClientSecret = "google-client-secret"
		s.cluster.Login.Google.AlternateEmailAddresses = enableAltEmails
		s.cluster.Login.Google.AuthenticationRequestParameters = map[string]string{"testkey": "testvalue"}
		localdb = NewConn(context.Background(), s.cluster, (&ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}).GetDB)
		ctrl = localdb.loginController.(*oidcLoginController)
		c.Check(ctrl.Issuer, check.Equals, "https://accounts.google.com")
		c.Check(ctrl.ClientID, check.Equals, "google-client-id")
		c.Check(ctrl.ClientSecret, check.Equals, "google-client-secret")
		c.Check(ctrl.UseGooglePeopleAPI, check.Equals, enableAltEmails)
		c.Check(ctrl.AuthParams["testkey"], check.Equals, "testvalue")
	}
}

func (s *OIDCLoginSuite) TestGoogleLogin_PeopleAPIError(c *check.C) {
	s.setupPeopleAPIError(c)
	state := s.startLogin(c)
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
}

func (s *OIDCLoginSuite) TestOIDCAuthorizer(c *check.C) {
	s.cluster.Login.Google.Enable = false
	s.cluster.Login.OpenIDConnect.Enable = true
	json.Unmarshal([]byte(fmt.Sprintf("%q", s.fakeProvider.Issuer.URL)), &s.cluster.Login.OpenIDConnect.Issuer)
	s.cluster.Login.OpenIDConnect.ClientID = "oidc#client#id"
	s.cluster.Login.OpenIDConnect.ClientSecret = "oidc#client#secret"
	s.cluster.Login.OpenIDConnect.AcceptAccessToken = true
	s.cluster.Login.OpenIDConnect.AcceptAccessTokenScope = ""
	s.fakeProvider.ValidClientID = "oidc#client#id"
	s.fakeProvider.ValidClientSecret = "oidc#client#secret"
	db := arvadostest.DB(c, s.cluster)

	tokenCacheTTL = time.Millisecond
	tokenCacheRaceWindow = time.Millisecond
	tokenCacheNegativeTTL = time.Millisecond

	oidcAuthorizer := OIDCAccessTokenAuthorizer(s.cluster, func(context.Context) (*sqlx.DB, error) { return db, nil })
	accessToken := s.fakeProvider.ValidAccessToken()

	mac := hmac.New(sha256.New, []byte(s.cluster.SystemRootToken))
	io.WriteString(mac, accessToken)
	apiToken := fmt.Sprintf("%x", mac.Sum(nil))

	checkTokenInDB := func() time.Time {
		var exp time.Time
		err := db.QueryRow(`select expires_at at time zone 'UTC' from api_client_authorizations where api_token=$1`, apiToken).Scan(&exp)
		c.Check(err, check.IsNil)
		c.Check(exp.Sub(time.Now()) > -time.Second, check.Equals, true)
		c.Check(exp.Sub(time.Now()) < time.Second, check.Equals, true)
		return exp
	}
	cleanup := func() {
		oidcAuthorizer.cache.Purge()
		_, err := db.Exec(`delete from api_client_authorizations where api_token=$1`, apiToken)
		c.Check(err, check.IsNil)
	}
	cleanup()
	defer cleanup()

	ctx := ctrlctx.NewWithToken(s.ctx, s.cluster, accessToken)

	// Check behavior on 5xx/network errors (don't cache) vs 4xx
	// (do cache)
	{
		call := oidcAuthorizer.WrapCalls(func(ctx context.Context, opts interface{}) (interface{}, error) {
			return nil, nil
		})

		// If fakeProvider UserInfo endpoint returns 502, we
		// should fail, return an error, and *not* cache the
		// negative result.
		tokenCacheNegativeTTL = time.Minute
		s.fakeProvider.UserInfoErrorStatus = 502
		_, err := call(ctx, nil)
		c.Check(err, check.NotNil)

		// The negative result was not cached, so retrying
		// immediately (with UserInfo working now) should
		// succeed.
		s.fakeProvider.UserInfoErrorStatus = 0
		_, err = call(ctx, nil)
		c.Check(err, check.IsNil)
		checkTokenInDB()

		cleanup()

		// UserInfo 401 => cache the negative result, but
		// don't return an error (just pass the token through
		// as a v1 token)
		s.fakeProvider.UserInfoErrorStatus = 401
		_, err = call(ctx, nil)
		c.Check(err, check.IsNil)
		ent, ok := oidcAuthorizer.cache.Get(accessToken)
		c.Check(ok, check.Equals, true)
		c.Check(ent, check.FitsTypeOf, time.Time{})

		// UserInfo succeeds now, but we still have a cached
		// negative result.
		s.fakeProvider.UserInfoErrorStatus = 0
		_, err = call(ctx, nil)
		c.Check(err, check.IsNil)
		ent, ok = oidcAuthorizer.cache.Get(accessToken)
		c.Check(ok, check.Equals, true)
		c.Check(ent, check.FitsTypeOf, time.Time{})

		tokenCacheNegativeTTL = time.Millisecond
		cleanup()
	}

	var exp1 time.Time
	concurrent := 4
	s.fakeProvider.HoldUserInfo = make(chan *http.Request)
	s.fakeProvider.ReleaseUserInfo = make(chan struct{})
	go func() {
		for i := 0; ; i++ {
			if i == concurrent {
				close(s.fakeProvider.ReleaseUserInfo)
			}
			<-s.fakeProvider.HoldUserInfo
		}
	}()
	var wg sync.WaitGroup
	for i := 0; i < concurrent; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := oidcAuthorizer.WrapCalls(func(ctx context.Context, opts interface{}) (interface{}, error) {
				c.Logf("concurrent req %d/%d", i, concurrent)

				creds, ok := auth.FromContext(ctx)
				c.Assert(ok, check.Equals, true)
				c.Assert(creds.Tokens, check.HasLen, 1)
				c.Check(creds.Tokens[0], check.Equals, accessToken)
				exp := checkTokenInDB()
				if i == 0 {
					exp1 = exp
				}
				return nil, nil
			})(ctx, nil)
			c.Check(err, check.IsNil)
		}()
	}
	wg.Wait()
	if c.Failed() {
		c.Fatal("giving up")
	}

	// If the token is used again after the in-memory cache
	// expires, oidcAuthorizer must re-check the token and update
	// the expires_at value in the database.
	time.Sleep(3 * time.Millisecond)
	oidcAuthorizer.WrapCalls(func(ctx context.Context, opts interface{}) (interface{}, error) {
		exp := checkTokenInDB()
		c.Check(exp.Sub(exp1) > 0, check.Equals, true, check.Commentf("expect %v > 0", exp.Sub(exp1)))
		c.Check(exp.Sub(exp1) < time.Second, check.Equals, true, check.Commentf("expect %v < 1s", exp.Sub(exp1)))
		return nil, nil
	})(ctx, nil)

	s.fakeProvider.AccessTokenPayload = map[string]interface{}{"scope": "openid profile foobar"}
	accessToken = s.fakeProvider.ValidAccessToken()
	ctx = ctrlctx.NewWithToken(s.ctx, s.cluster, accessToken)

	mac = hmac.New(sha256.New, []byte(s.cluster.SystemRootToken))
	io.WriteString(mac, accessToken)
	apiToken = fmt.Sprintf("%x", mac.Sum(nil))

	for _, trial := range []struct {
		configEnable bool
		configScope  string
		acceptable   bool
		shouldRun    bool
	}{
		{true, "foobar", true, true},
		{true, "foo", false, false},
		{true, "", true, true},
		{false, "", false, true},
		{false, "foobar", false, true},
	} {
		c.Logf("trial = %+v", trial)
		cleanup()
		s.cluster.Login.OpenIDConnect.AcceptAccessToken = trial.configEnable
		s.cluster.Login.OpenIDConnect.AcceptAccessTokenScope = trial.configScope
		oidcAuthorizer = OIDCAccessTokenAuthorizer(s.cluster, func(context.Context) (*sqlx.DB, error) { return db, nil })
		checked := false
		oidcAuthorizer.WrapCalls(func(ctx context.Context, opts interface{}) (interface{}, error) {
			var n int
			err := db.QueryRowContext(ctx, `select count(*) from api_client_authorizations where api_token=$1`, apiToken).Scan(&n)
			c.Check(err, check.IsNil)
			if trial.acceptable {
				c.Check(n, check.Equals, 1)
			} else {
				c.Check(n, check.Equals, 0)
			}
			checked = true
			return nil, nil
		})(ctx, nil)
		c.Check(checked, check.Equals, trial.shouldRun)
	}
}

func (s *OIDCLoginSuite) TestGenericOIDCLogin(c *check.C) {
	s.cluster.Login.Google.Enable = false
	s.cluster.Login.OpenIDConnect.Enable = true
	json.Unmarshal([]byte(fmt.Sprintf("%q", s.fakeProvider.Issuer.URL)), &s.cluster.Login.OpenIDConnect.Issuer)
	s.cluster.Login.OpenIDConnect.ClientID = "oidc#client#id"
	s.cluster.Login.OpenIDConnect.ClientSecret = "oidc#client#secret"
	s.cluster.Login.OpenIDConnect.AuthenticationRequestParameters = map[string]string{"testkey": "testvalue"}
	s.fakeProvider.ValidClientID = "oidc#client#id"
	s.fakeProvider.ValidClientSecret = "oidc#client#secret"
	for _, trial := range []struct {
		expectEmail string // "" if failure expected
		setup       func()
	}{
		{
			expectEmail: "user@oidc.example.com",
			setup: func() {
				c.Log("=== succeed because email_verified is false but not required")
				s.fakeProvider.AuthEmail = "user@oidc.example.com"
				s.fakeProvider.AuthEmailVerified = false
				s.cluster.Login.OpenIDConnect.EmailClaim = "email"
				s.cluster.Login.OpenIDConnect.EmailVerifiedClaim = ""
				s.cluster.Login.OpenIDConnect.UsernameClaim = ""
			},
		},
		{
			expectEmail: "",
			setup: func() {
				c.Log("=== fail because email_verified is false and required")
				s.fakeProvider.AuthEmail = "user@oidc.example.com"
				s.fakeProvider.AuthEmailVerified = false
				s.cluster.Login.OpenIDConnect.EmailClaim = "email"
				s.cluster.Login.OpenIDConnect.EmailVerifiedClaim = "email_verified"
				s.cluster.Login.OpenIDConnect.UsernameClaim = ""
			},
		},
		{
			expectEmail: "user@oidc.example.com",
			setup: func() {
				c.Log("=== succeed because email_verified is false but config uses custom 'verified' claim")
				s.fakeProvider.AuthEmail = "user@oidc.example.com"
				s.fakeProvider.AuthEmailVerified = false
				s.cluster.Login.OpenIDConnect.EmailClaim = "email"
				s.cluster.Login.OpenIDConnect.EmailVerifiedClaim = "alt_verified"
				s.cluster.Login.OpenIDConnect.UsernameClaim = ""
			},
		},
		{
			expectEmail: "alt_email@example.com",
			setup: func() {
				c.Log("=== succeed with custom 'email' and 'email_verified' claims")
				s.fakeProvider.AuthEmail = "bad@wrong.example.com"
				s.fakeProvider.AuthEmailVerified = false
				s.cluster.Login.OpenIDConnect.EmailClaim = "alt_email"
				s.cluster.Login.OpenIDConnect.EmailVerifiedClaim = "alt_verified"
				s.cluster.Login.OpenIDConnect.UsernameClaim = "alt_username"
			},
		},
	} {
		trial.setup()
		if s.railsSpy != nil {
			s.railsSpy.Close()
		}
		s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
		s.localdb = NewConn(context.Background(), s.cluster, (&ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}).GetDB)
		*s.localdb.railsProxy = *rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)

		state := s.startLogin(c, func(form url.Values) {
			c.Check(form.Get("testkey"), check.Equals, "testvalue")
		})
		resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
			Code:  s.fakeProvider.ValidCode,
			State: state,
		})
		c.Assert(err, check.IsNil)
		if trial.expectEmail == "" {
			c.Check(resp.HTML.String(), check.Matches, `(?ms).*Login error.*`)
			c.Check(resp.RedirectLocation, check.Equals, "")
			continue
		}
		c.Check(resp.HTML.String(), check.Equals, "")
		target, err := url.Parse(resp.RedirectLocation)
		c.Assert(err, check.IsNil)
		token := target.Query().Get("api_token")
		c.Check(token, check.Matches, `v2/zzzzz-gj3su-.{15}/.{32,50}`)
		authinfo := getCallbackAuthInfo(c, s.railsSpy)
		c.Check(authinfo.Email, check.Equals, trial.expectEmail)

		switch s.cluster.Login.OpenIDConnect.UsernameClaim {
		case "alt_username":
			c.Check(authinfo.Username, check.Equals, "desired-username")
		case "":
			c.Check(authinfo.Username, check.Equals, "")
		default:
			c.Fail() // bad test case
		}
	}
}

func (s *OIDCLoginSuite) TestGoogleLogin_Success(c *check.C) {
	s.cluster.Login.Google.AuthenticationRequestParameters["prompt"] = "consent"
	s.cluster.Login.Google.AuthenticationRequestParameters["foo"] = "bar"
	state := s.startLogin(c, func(form url.Values) {
		c.Check(form.Get("foo"), check.Equals, "bar")
		c.Check(form.Get("prompt"), check.Equals, "consent")
	})
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.HTML.String(), check.Equals, "")
	target, err := url.Parse(resp.RedirectLocation)
	c.Check(err, check.IsNil)
	c.Check(target.Host, check.Equals, "app.example.com")
	c.Check(target.Path, check.Equals, "/foo")
	token := target.Query().Get("api_token")
	c.Check(token, check.Matches, `v2/zzzzz-gj3su-.{15}/.{32,50}`)

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.FirstName, check.Equals, "Fake")
	c.Check(authinfo.LastName, check.Equals, "User Name")
	c.Check(authinfo.Email, check.Equals, "active-user@arvados.local")
	c.Check(authinfo.AlternateEmails, check.HasLen, 0)

	// Try using the returned Arvados token.
	c.Logf("trying an API call with new token %q", token)
	ctx := ctrlctx.NewWithToken(s.ctx, s.cluster, token)
	cl, err := s.localdb.CollectionList(ctx, arvados.ListOptions{Limit: -1})
	c.Check(cl.ItemsAvailable, check.Not(check.Equals), 0)
	c.Check(cl.Items, check.Not(check.HasLen), 0)
	c.Check(err, check.IsNil)

	// Might as well check that bogus tokens aren't accepted.
	badtoken := token + "plussomeboguschars"
	c.Logf("trying an API call with mangled token %q", badtoken)
	ctx = ctrlctx.NewWithToken(s.ctx, s.cluster, badtoken)
	cl, err = s.localdb.CollectionList(ctx, arvados.ListOptions{Limit: -1})
	c.Check(cl.Items, check.HasLen, 0)
	c.Check(err, check.NotNil)
	c.Check(err, check.ErrorMatches, `.*401 Unauthorized: Not logged in.*`)
}

func (s *OIDCLoginSuite) TestGoogleLogin_RealName(c *check.C) {
	s.fakeProvider.AuthEmail = "joe.smith@primary.example.com"
	s.fakeProvider.AuthEmailVerified = true
	s.fakeProvider.PeopleAPIResponse = map[string]interface{}{
		"names": []map[string]interface{}{
			{
				"metadata":   map[string]interface{}{"primary": false},
				"givenName":  "Joe",
				"familyName": "Smith",
			},
			{
				"metadata":   map[string]interface{}{"primary": true},
				"givenName":  "Joseph",
				"familyName": "Psmith",
			},
		},
	}
	state := s.startLogin(c)
	s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.FirstName, check.Equals, "Joseph")
	c.Check(authinfo.LastName, check.Equals, "Psmith")
}

func (s *OIDCLoginSuite) TestGoogleLogin_OIDCNameWithoutGivenAndFamilyNames(c *check.C) {
	s.fakeProvider.AuthName = "Joe P. Smith"
	s.fakeProvider.AuthGivenName = ""
	s.fakeProvider.AuthFamilyName = ""
	s.fakeProvider.AuthEmail = "joe.smith@primary.example.com"
	state := s.startLogin(c)
	s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.FirstName, check.Equals, "Joe P.")
	c.Check(authinfo.LastName, check.Equals, "Smith")
}

// People API returns some additional email addresses.
func (s *OIDCLoginSuite) TestGoogleLogin_AlternateEmailAddresses(c *check.C) {
	s.fakeProvider.AuthEmail = "joe.smith@primary.example.com"
	s.fakeProvider.PeopleAPIResponse = map[string]interface{}{
		"emailAddresses": []map[string]interface{}{
			{
				"metadata": map[string]interface{}{"verified": true},
				"value":    "joe.smith@work.example.com",
			},
			{
				"value": "joe.smith@unverified.example.com", // unverified, so this one will be ignored
			},
			{
				"metadata": map[string]interface{}{"verified": true},
				"value":    "joe.smith@home.example.com",
			},
		},
	}
	state := s.startLogin(c)
	s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.Email, check.Equals, "joe.smith@primary.example.com")
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string{"joe.smith@home.example.com", "joe.smith@work.example.com"})
}

// Primary address is not the one initially returned by oidc.
func (s *OIDCLoginSuite) TestGoogleLogin_AlternateEmailAddresses_Primary(c *check.C) {
	s.fakeProvider.AuthEmail = "joe.smith@alternate.example.com"
	s.fakeProvider.PeopleAPIResponse = map[string]interface{}{
		"emailAddresses": []map[string]interface{}{
			{
				"metadata": map[string]interface{}{"verified": true, "primary": true},
				"value":    "joe.smith@primary.example.com",
			},
			{
				"metadata": map[string]interface{}{"verified": true},
				"value":    "joe.smith@alternate.example.com",
			},
			{
				"metadata": map[string]interface{}{"verified": true},
				"value":    "jsmith+123@preferdomainforusername.example.com",
			},
		},
	}
	state := s.startLogin(c)
	s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})
	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.Email, check.Equals, "joe.smith@primary.example.com")
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string{"joe.smith@alternate.example.com", "jsmith+123@preferdomainforusername.example.com"})
	c.Check(authinfo.Username, check.Equals, "jsmith")
}

func (s *OIDCLoginSuite) TestGoogleLogin_NoPrimaryEmailAddress(c *check.C) {
	s.fakeProvider.AuthEmail = "joe.smith@unverified.example.com"
	s.fakeProvider.AuthEmailVerified = false
	s.fakeProvider.PeopleAPIResponse = map[string]interface{}{
		"emailAddresses": []map[string]interface{}{
			{
				"metadata": map[string]interface{}{"verified": true},
				"value":    "joe.smith@work.example.com",
			},
			{
				"metadata": map[string]interface{}{"verified": true},
				"value":    "joe.smith@home.example.com",
			},
		},
	}
	state := s.startLogin(c)
	s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.fakeProvider.ValidCode,
		State: state,
	})

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.Email, check.Equals, "joe.smith@work.example.com") // first verified email in People response
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string{"joe.smith@home.example.com"})
	c.Check(authinfo.Username, check.Equals, "")
}

func (s *OIDCLoginSuite) startLogin(c *check.C, checks ...func(url.Values)) (state string) {
	// Initiate login, but instead of following the redirect to
	// the provider, just grab state from the redirect URL.
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{ReturnTo: "https://app.example.com/foo?bar"})
	c.Check(err, check.IsNil)
	c.Check(resp.HTML.String(), check.Not(check.Matches), `(?ms).*error:.*`)
	target, err := url.Parse(resp.RedirectLocation)
	c.Check(err, check.IsNil)
	state = target.Query().Get("state")
	if !c.Check(state, check.Not(check.Equals), "") {
		c.Logf("Redirect target: %q", target)
		c.Logf("HTML: %q", resp.HTML)
	}
	for _, fn := range checks {
		fn(target.Query())
	}
	s.cluster.Login.OpenIDConnect.AuthenticationRequestParameters = map[string]string{"testkey": "testvalue"}
	return
}

func (s *OIDCLoginSuite) TestValidateLoginRedirectTarget(c *check.C) {
	for _, trial := range []struct {
		permit       bool
		trustPrivate bool
		url          string
	}{
		// wb1, wb2 => accept
		{true, false, s.cluster.Services.Workbench1.ExternalURL.String()},
		{true, false, s.cluster.Services.Workbench2.ExternalURL.String()},
		// explicitly listed host => accept
		{true, false, "https://app.example.com/"},
		{true, false, "https://app.example.com:443/foo?bar=baz"},
		// non-listed hostname => deny (regardless of TrustPrivateNetworks)
		{false, false, "https://bad.example/"},
		{false, true, "https://bad.example/"},
		// non-listed non-private IP addr => deny (regardless of TrustPrivateNetworks)
		{false, true, "https://1.2.3.4/"},
		{false, true, "https://1.2.3.4/"},
		{false, true, "https://[ab::cd]:1234/"},
		// localhost or non-listed private IP addr => accept only if TrustPrivateNetworks is set
		{false, false, "https://localhost/"},
		{true, true, "https://localhost/"},
		{false, false, "https://[10.9.8.7]:80/foo"},
		{true, true, "https://[10.9.8.7]:80/foo"},
		{false, false, "https://[::1]:80/foo"},
		{true, true, "https://[::1]:80/foo"},
		{true, true, "http://192.168.1.1/"},
		{true, true, "http://172.17.2.0/"},
		// bad url => deny
		{false, true, "https://10.1.1.1:blorp/foo"},        // non-numeric port
		{false, true, "https://app.example.com:blorp/foo"}, // non-numeric port
		{false, true, "https://]:443"},
		{false, true, "https://"},
		{false, true, "https:"},
		{false, true, ""},
		// explicitly listed host but different port, protocol, or user/pass => deny
		{false, true, "http://app.example.com/"},
		{false, true, "http://app.example.com:443/"},
		{false, true, "https://app.example.com:80/"},
		{false, true, "https://app.example.com:4433/"},
		{false, true, "https://u:p@app.example.com:443/foo?bar=baz"},
	} {
		c.Logf("trial %+v", trial)
		s.cluster.Login.TrustPrivateNetworks = trial.trustPrivate
		err := validateLoginRedirectTarget(s.cluster, trial.url)
		c.Check(err == nil, check.Equals, trial.permit)
	}

}

func getCallbackAuthInfo(c *check.C, railsSpy *arvadostest.Proxy) (authinfo rpc.UserSessionAuthInfo) {
	for _, dump := range railsSpy.RequestDumps {
		c.Logf("spied request: %q", dump)
		split := bytes.Split(dump, []byte("\r\n\r\n"))
		c.Assert(split, check.HasLen, 2)
		hdr, body := string(split[0]), string(split[1])
		if strings.Contains(hdr, "POST /auth/controller/callback") {
			vs, err := url.ParseQuery(body)
			c.Check(json.Unmarshal([]byte(vs.Get("auth_info")), &authinfo), check.IsNil)
			c.Check(err, check.IsNil)
			sort.Strings(authinfo.AlternateEmails)
			return
		}
	}
	c.Error("callback not found")
	return
}
