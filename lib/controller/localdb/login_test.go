// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/lib/config"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
	jose "gopkg.in/square/go-jose.v2"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&LoginSuite{})

type LoginSuite struct {
	cluster    *arvados.Cluster
	ctx        context.Context
	localdb    *Conn
	fakeIssuer *httptest.Server
	issuerKey  *rsa.PrivateKey

	// expected token request
	validCode string
	// desired response from token endpoint
	authEmail         string
	authEmailVerified bool
	authName          string
}

func (s *LoginSuite) SetUpTest(c *check.C) {
	var err error
	s.issuerKey, err = rsa.GenerateKey(rand.Reader, 2048)
	c.Assert(err, check.IsNil)

	s.authEmail = "active-user@arvados.local"
	s.authEmailVerified = true
	s.authName = "Fake User Name"
	s.fakeIssuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		c.Logf("fakeIssuer: got req: %s %s %s", req.Method, req.URL, req.Form)
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"issuer":                 s.fakeIssuer.URL,
				"authorization_endpoint": s.fakeIssuer.URL + "/auth",
				"token_endpoint":         s.fakeIssuer.URL + "/token",
				"jwks_uri":               s.fakeIssuer.URL + "/jwks",
				"userinfo_endpoint":      s.fakeIssuer.URL + "/userinfo",
			})
		case "/token":
			if req.Form.Get("code") != s.validCode || s.validCode == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			idToken, _ := json.Marshal(map[string]interface{}{
				"iss":            s.fakeIssuer.URL,
				"aud":            []string{"test%client$id"},
				"sub":            "fake-user-id",
				"exp":            time.Now().UTC().Add(time.Minute).UnixNano(),
				"iat":            time.Now().UTC().UnixNano(),
				"nonce":          "fake-nonce",
				"email":          s.authEmail,
				"email_verified": s.authEmailVerified,
				"name":           s.authName,
			})
			json.NewEncoder(w).Encode(struct {
				AccessToken  string `json:"access_token"`
				TokenType    string `json:"token_type"`
				RefreshToken string `json:"refresh_token"`
				ExpiresIn    int32  `json:"expires_in"`
				IDToken      string `json:"id_token"`
			}{
				AccessToken:  s.fakeToken(c, []byte("fake access token")),
				TokenType:    "Bearer",
				RefreshToken: "test-refresh-token",
				ExpiresIn:    30,
				IDToken:      s.fakeToken(c, idToken),
			})
		case "/jwks":
			json.NewEncoder(w).Encode(jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{Key: s.issuerKey.Public(), Algorithm: string(jose.RS256), KeyID: ""},
				},
			})
		case "/auth":
			w.WriteHeader(http.StatusInternalServerError)
		case "/userinfo":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	s.cluster, err = cfg.GetCluster("")
	s.cluster.Login.GoogleClientID = "test%client$id"
	s.cluster.Login.GoogleClientSecret = "test#client/secret"
	c.Assert(err, check.IsNil)

	s.localdb = NewConn(s.cluster)
	s.localdb.googleLoginController.issuer = s.fakeIssuer.URL
}

func (s *LoginSuite) TestGoogleLoginStart_Bogus(c *check.C) {
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `.*missing return_to parameter.*`)
}

func (s *LoginSuite) TestGoogleLoginStart(c *check.C) {
	for _, remote := range []string{"", "zzzzz"} {
		resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{Remote: remote, ReturnTo: "https://app.example.com/foo?bar"})
		c.Check(err, check.IsNil)
		target, err := url.Parse(resp.RedirectLocation)
		c.Check(err, check.IsNil)
		issuerURL, _ := url.Parse(s.fakeIssuer.URL)
		c.Check(target.Host, check.Equals, issuerURL.Host)
		q := target.Query()
		c.Check(q.Get("client_id"), check.Equals, "test%client$id")
		state := s.localdb.googleLoginController.parseOAuth2State(q.Get("state"))
		c.Check(state.verify([]byte(s.cluster.SystemRootToken)), check.Equals, true)
		c.Check(state.Time, check.Not(check.Equals), 0)
		c.Check(state.Remote, check.Equals, remote)
		c.Check(state.ReturnTo, check.Equals, "https://app.example.com/foo?bar")
	}
}

func (s *LoginSuite) TestGoogleLoginSuccess(c *check.C) {
	// Initiate login, but instead of following the redirect to
	// the provider, just grab state from the redirect URL.
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{ReturnTo: "https://app.example.com/foo?bar"})
	c.Check(err, check.IsNil)
	target, err := url.Parse(resp.RedirectLocation)
	c.Check(err, check.IsNil)
	state := target.Query().Get("state")
	c.Check(state, check.Not(check.Equals), "")

	// Prime the fake issuer with a valid code.
	s.validCode = fmt.Sprintf("abcdefgh-%d", time.Now().Unix())

	// Callback with invalid code.
	resp, err = s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  "first-try-a-bogus-code",
		State: state,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*error in OAuth2 exchange.*cannot fetch token.*`)

	// Callback with invalid state.
	resp, err = s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.validCode,
		State: "bogus-state",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*invalid OAuth2 state.*`)

	// Callback with valid code and state.
	resp, err = s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.validCode,
		State: state,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.HTML.String(), check.Equals, "")
	c.Check(resp.RedirectLocation, check.Not(check.Equals), "")
	target, err = url.Parse(resp.RedirectLocation)
	c.Check(err, check.IsNil)
	c.Check(target.Host, check.Equals, "app.example.com")
	c.Check(target.Path, check.Equals, "/foo")
	token := target.Query().Get("api_token")
	c.Check(token, check.Matches, `v2/zzzzz-gj3su-.{15}/.{32,50}`)

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

func (s *LoginSuite) fakeToken(c *check.C, payload []byte) string {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: s.issuerKey}, nil)
	if err != nil {
		c.Error(err)
	}
	object, err := signer.Sign(payload)
	if err != nil {
		c.Error(err)
	}
	t, err := object.CompactSerialize()
	if err != nil {
		c.Error(err)
	}
	c.Logf("fakeToken(%q) == %q", payload, t)
	return t
}
