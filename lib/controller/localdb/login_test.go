// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
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
	check "gopkg.in/check.v1"
	jose "gopkg.in/square/go-jose.v2"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&LoginSuite{})

type LoginSuite struct {
	cluster               *arvados.Cluster
	ctx                   context.Context
	localdb               *Conn
	railsSpy              *arvadostest.Proxy
	fakeIssuer            *httptest.Server
	fakePeopleAPI         *httptest.Server
	fakePeopleAPIResponse map[string]interface{}
	issuerKey             *rsa.PrivateKey

	// expected token request
	validCode string
	// desired response from token endpoint
	authEmail         string
	authEmailVerified bool
	authName          string
}

func (s *LoginSuite) TearDownSuite(c *check.C) {
	// Undo any changes/additions to the user database so they
	// don't affect subsequent tests.
	arvadostest.ResetEnv()
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
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
	s.validCode = fmt.Sprintf("abcdefgh-%d", time.Now().Unix())

	s.fakePeopleAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		c.Logf("fakePeopleAPI: got req: %s %s %s", req.Method, req.URL, req.Form)
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/v1/people/me":
			if f := req.Form.Get("personFields"); f != "emailAddresses,names" {
				w.WriteHeader(http.StatusBadRequest)
				break
			}
			json.NewEncoder(w).Encode(s.fakePeopleAPIResponse)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	s.fakePeopleAPIResponse = map[string]interface{}{}

	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	s.cluster, err = cfg.GetCluster("")
	s.cluster.Login.GoogleClientID = "test%client$id"
	s.cluster.Login.GoogleClientSecret = "test#client/secret"
	s.cluster.Users.PreferDomainForUsername = "PreferDomainForUsername.example.com"
	c.Assert(err, check.IsNil)

	s.localdb = NewConn(s.cluster)
	s.localdb.googleLoginController.issuer = s.fakeIssuer.URL
	s.localdb.googleLoginController.peopleAPIBasePath = s.fakePeopleAPI.URL

	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	s.localdb.railsProxy = rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)
}

func (s *LoginSuite) TearDownTest(c *check.C) {
	s.railsSpy.Close()
}

func (s *LoginSuite) TestGoogleLogout(c *check.C) {
	resp, err := s.localdb.Logout(context.Background(), arvados.LogoutOptions{ReturnTo: "https://foo.example.com/bar"})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "https://foo.example.com/bar")
}

func (s *LoginSuite) TestGoogleLogin_Start_Bogus(c *check.C) {
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `.*missing return_to parameter.*`)
}

func (s *LoginSuite) TestGoogleLogin_Start(c *check.C) {
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

func (s *LoginSuite) TestGoogleLogin_InvalidCode(c *check.C) {
	state := s.startLogin(c)
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  "first-try-a-bogus-code",
		State: state,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*error in OAuth2 exchange.*cannot fetch token.*`)
}

func (s *LoginSuite) TestGoogleLogin_InvalidState(c *check.C) {
	s.startLogin(c)
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.validCode,
		State: "bogus-state",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
	c.Check(resp.HTML.String(), check.Matches, `(?ms).*invalid OAuth2 state.*`)
}

func (s *LoginSuite) setupPeopleAPIError(c *check.C) {
	s.fakePeopleAPI = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, `Error 403: accessNotConfigured`)
	}))
	s.localdb.googleLoginController.peopleAPIBasePath = s.fakePeopleAPI.URL
}

func (s *LoginSuite) TestGoogleLogin_PeopleAPIDisabled(c *check.C) {
	s.cluster.Login.GoogleAlternateEmailAddresses = false
	s.authEmail = "joe.smith@primary.example.com"
	s.setupPeopleAPIError(c)
	state := s.startLogin(c)
	_, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.validCode,
		State: state,
	})
	c.Check(err, check.IsNil)
	authinfo := s.getCallbackAuthInfo(c)
	c.Check(authinfo.Email, check.Equals, "joe.smith@primary.example.com")
}

func (s *LoginSuite) TestGoogleLogin_PeopleAPIError(c *check.C) {
	s.setupPeopleAPIError(c)
	state := s.startLogin(c)
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.validCode,
		State: state,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.RedirectLocation, check.Equals, "")
}

func (s *LoginSuite) TestGoogleLogin_Success(c *check.C) {
	state := s.startLogin(c)
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.validCode,
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

	authinfo := s.getCallbackAuthInfo(c)
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

func (s *LoginSuite) TestGoogleLogin_RealName(c *check.C) {
	s.authEmail = "joe.smith@primary.example.com"
	s.fakePeopleAPIResponse = map[string]interface{}{
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
		Code:  s.validCode,
		State: state,
	})

	authinfo := s.getCallbackAuthInfo(c)
	c.Check(authinfo.FirstName, check.Equals, "Joseph")
	c.Check(authinfo.LastName, check.Equals, "Psmith")
}

func (s *LoginSuite) TestGoogleLogin_OIDCRealName(c *check.C) {
	s.authName = "Joe P. Smith"
	s.authEmail = "joe.smith@primary.example.com"
	state := s.startLogin(c)
	s.localdb.Login(context.Background(), arvados.LoginOptions{
		Code:  s.validCode,
		State: state,
	})

	authinfo := s.getCallbackAuthInfo(c)
	c.Check(authinfo.FirstName, check.Equals, "Joe P.")
	c.Check(authinfo.LastName, check.Equals, "Smith")
}

// People API returns some additional email addresses.
func (s *LoginSuite) TestGoogleLogin_AlternateEmailAddresses(c *check.C) {
	s.authEmail = "joe.smith@primary.example.com"
	s.fakePeopleAPIResponse = map[string]interface{}{
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
		Code:  s.validCode,
		State: state,
	})

	authinfo := s.getCallbackAuthInfo(c)
	c.Check(authinfo.Email, check.Equals, "joe.smith@primary.example.com")
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string{"joe.smith@home.example.com", "joe.smith@work.example.com"})
}

// Primary address is not the one initially returned by oidc.
func (s *LoginSuite) TestGoogleLogin_AlternateEmailAddresses_Primary(c *check.C) {
	s.authEmail = "joe.smith@alternate.example.com"
	s.fakePeopleAPIResponse = map[string]interface{}{
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
		Code:  s.validCode,
		State: state,
	})
	authinfo := s.getCallbackAuthInfo(c)
	c.Check(authinfo.Email, check.Equals, "joe.smith@primary.example.com")
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string{"joe.smith@alternate.example.com", "jsmith+123@preferdomainforusername.example.com"})
	c.Check(authinfo.Username, check.Equals, "jsmith")
}

func (s *LoginSuite) TestGoogleLogin_NoPrimaryEmailAddress(c *check.C) {
	s.authEmail = "joe.smith@unverified.example.com"
	s.authEmailVerified = false
	s.fakePeopleAPIResponse = map[string]interface{}{
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
		Code:  s.validCode,
		State: state,
	})

	authinfo := s.getCallbackAuthInfo(c)
	c.Check(authinfo.Email, check.Equals, "joe.smith@work.example.com") // first verified email in People response
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string{"joe.smith@home.example.com"})
	c.Check(authinfo.Username, check.Equals, "")
}

func (s *LoginSuite) getCallbackAuthInfo(c *check.C) (authinfo rpc.UserSessionAuthInfo) {
	for _, dump := range s.railsSpy.RequestDumps {
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

func (s *LoginSuite) startLogin(c *check.C) (state string) {
	// Initiate login, but instead of following the redirect to
	// the provider, just grab state from the redirect URL.
	resp, err := s.localdb.Login(context.Background(), arvados.LoginOptions{ReturnTo: "https://app.example.com/foo?bar"})
	c.Check(err, check.IsNil)
	target, err := url.Parse(resp.RedirectLocation)
	c.Check(err, check.IsNil)
	state = target.Query().Get("state")
	c.Check(state, check.Not(check.Equals), "")
	return
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
