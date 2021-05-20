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
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&OIDCLoginSuite{})

type OIDCLoginSuite struct {
	cluster      *arvados.Cluster
	localdb      *Conn
	railsSpy     *arvadostest.Proxy
	fakeProvider *arvadostest.OIDCProvider
}

func (s *OIDCLoginSuite) TearDownSuite(c *check.C) {
	// Undo any changes/additions to the user database so they
	// don't affect subsequent tests.
	arvadostest.ResetEnv()
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
}

func (s *OIDCLoginSuite) SetUpTest(c *check.C) {
	s.fakeProvider = arvadostest.NewOIDCProvider(c)
	s.fakeProvider.AuthEmail = "active-user@arvados.local"
	s.fakeProvider.AuthEmailVerified = true
	s.fakeProvider.AuthName = "Fake User Name"
	s.fakeProvider.ValidCode = fmt.Sprintf("abcdefgh-%d", time.Now().Unix())
	s.fakeProvider.PeopleAPIResponse = map[string]interface{}{}

	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.cluster.Login.SSO.Enable = false
	s.cluster.Login.Google.Enable = true
	s.cluster.Login.Google.ClientID = "test%client$id"
	s.cluster.Login.Google.ClientSecret = "test#client/secret"
	s.cluster.Users.PreferDomainForUsername = "PreferDomainForUsername.example.com"
	s.fakeProvider.ValidClientID = "test%client$id"
	s.fakeProvider.ValidClientSecret = "test#client/secret"

	s.localdb = NewConn(s.cluster)
	c.Assert(s.localdb.loginController, check.FitsTypeOf, (*oidcLoginController)(nil))
	s.localdb.loginController.(*oidcLoginController).Issuer = s.fakeProvider.Issuer.URL
	s.localdb.loginController.(*oidcLoginController).peopleAPIBasePath = s.fakeProvider.PeopleAPI.URL

	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	*s.localdb.railsProxy = *rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)
}

func (s *OIDCLoginSuite) TearDownTest(c *check.C) {
	s.railsSpy.Close()
}

func (s *OIDCLoginSuite) TestGoogleLogout(c *check.C) {
	resp, err := s.localdb.Logout(context.Background(), arvados.LogoutOptions{ReturnTo: "https://foo.example.com/bar"})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "https://foo.example.com/bar")
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
	localdb := NewConn(s.cluster)
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
		localdb = NewConn(s.cluster)
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
	s.cluster.Login.OpenIDConnect.AcceptAccessTokenScope = "*"
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

	cleanup := func() {
		_, err := db.Exec(`delete from api_client_authorizations where api_token=$1`, apiToken)
		c.Check(err, check.IsNil)
	}
	cleanup()
	defer cleanup()

	ctx := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{accessToken}})
	var exp1 time.Time
	oidcAuthorizer.WrapCalls(func(ctx context.Context, opts interface{}) (interface{}, error) {
		creds, ok := auth.FromContext(ctx)
		c.Assert(ok, check.Equals, true)
		c.Assert(creds.Tokens, check.HasLen, 1)
		c.Check(creds.Tokens[0], check.Equals, accessToken)

		err := db.QueryRowContext(ctx, `select expires_at at time zone 'UTC' from api_client_authorizations where api_token=$1`, apiToken).Scan(&exp1)
		c.Check(err, check.IsNil)
		c.Check(exp1.Sub(time.Now()) > -time.Second, check.Equals, true)
		c.Check(exp1.Sub(time.Now()) < time.Second, check.Equals, true)
		return nil, nil
	})(ctx, nil)

	// If the token is used again after the in-memory cache
	// expires, oidcAuthorizer must re-check the token and update
	// the expires_at value in the database.
	time.Sleep(3 * time.Millisecond)
	oidcAuthorizer.WrapCalls(func(ctx context.Context, opts interface{}) (interface{}, error) {
		var exp time.Time
		err := db.QueryRowContext(ctx, `select expires_at at time zone 'UTC' from api_client_authorizations where api_token=$1`, apiToken).Scan(&exp)
		c.Check(err, check.IsNil)
		c.Check(exp.Sub(exp1) > 0, check.Equals, true)
		c.Check(exp.Sub(exp1) < time.Second, check.Equals, true)
		return nil, nil
	})(ctx, nil)

	s.fakeProvider.AccessTokenPayload = map[string]interface{}{"scope": "openid profile foobar"}
	accessToken = s.fakeProvider.ValidAccessToken()
	ctx = auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{accessToken}})

	mac = hmac.New(sha256.New, []byte(s.cluster.SystemRootToken))
	io.WriteString(mac, accessToken)
	apiToken = fmt.Sprintf("%x", mac.Sum(nil))

	for _, trial := range []struct {
		configScope string
		acceptable  bool
		shouldRun   bool
	}{
		{"foobar", true, true},
		{"foo", false, false},
		{"*", true, true},
		{"", false, true},
	} {
		c.Logf("trial = %+v", trial)
		cleanup()
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
		s.localdb = NewConn(s.cluster)
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
	c.Check(authinfo.FirstName, check.Equals, "Fake User")
	c.Check(authinfo.LastName, check.Equals, "Name")
	c.Check(authinfo.Email, check.Equals, "active-user@arvados.local")
	c.Check(authinfo.AlternateEmails, check.HasLen, 0)

	// Try using the returned Arvados token.
	c.Logf("trying an API call with new token %q", token)
	ctx := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{token}})
	cl, err := s.localdb.CollectionList(ctx, arvados.ListOptions{Limit: -1})
	c.Check(cl.ItemsAvailable, check.Not(check.Equals), 0)
	c.Check(cl.Items, check.Not(check.HasLen), 0)
	c.Check(err, check.IsNil)

	// Might as well check that bogus tokens aren't accepted.
	badtoken := token + "plussomeboguschars"
	c.Logf("trying an API call with mangled token %q", badtoken)
	ctx = auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{badtoken}})
	cl, err = s.localdb.CollectionList(ctx, arvados.ListOptions{Limit: -1})
	c.Check(cl.Items, check.HasLen, 0)
	c.Check(err, check.NotNil)
	c.Check(err, check.ErrorMatches, `.*401 Unauthorized: Not logged in.*`)
}

func (s *OIDCLoginSuite) TestGoogleLogin_RealName(c *check.C) {
	s.fakeProvider.AuthEmail = "joe.smith@primary.example.com"
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

func (s *OIDCLoginSuite) TestGoogleLogin_OIDCRealName(c *check.C) {
	s.fakeProvider.AuthName = "Joe P. Smith"
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
	target, err := url.Parse(resp.RedirectLocation)
	c.Check(err, check.IsNil)
	state = target.Query().Get("state")
	c.Check(state, check.Not(check.Equals), "")
	for _, fn := range checks {
		fn(target.Query())
	}
	s.cluster.Login.OpenIDConnect.AuthenticationRequestParameters = map[string]string{"testkey": "testvalue"}
	return
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
