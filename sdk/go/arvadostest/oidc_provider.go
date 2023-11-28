// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"gopkg.in/check.v1"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type OIDCProvider struct {
	// expected token request
	ValidCode         string
	ValidClientID     string
	ValidClientSecret string
	// desired response from token endpoint
	AuthEmail          string
	AuthEmailVerified  bool
	AuthName           string
	AuthGivenName      string
	AuthFamilyName     string
	AccessTokenPayload map[string]interface{}
	// end_session_endpoint metadata URL.
	// If nil or empty, not included in discovery.
	// If relative, built from Issuer.URL.
	EndSessionEndpoint *url.URL

	PeopleAPIResponse map[string]interface{}

	// send incoming /userinfo requests to HoldUserInfo (if not
	// nil), then receive from ReleaseUserInfo (if not nil),
	// before responding (these are used to set up races)
	HoldUserInfo        chan *http.Request
	ReleaseUserInfo     chan struct{}
	UserInfoErrorStatus int // if non-zero, return this http status (probably 5xx)

	key       *rsa.PrivateKey
	Issuer    *httptest.Server
	PeopleAPI *httptest.Server
	c         *check.C
}

func NewOIDCProvider(c *check.C) *OIDCProvider {
	p := &OIDCProvider{c: c}
	var err error
	p.key, err = rsa.GenerateKey(rand.Reader, 2048)
	c.Assert(err, check.IsNil)
	p.Issuer = httptest.NewServer(http.HandlerFunc(p.serveOIDC))
	p.PeopleAPI = httptest.NewServer(http.HandlerFunc(p.servePeopleAPI))
	p.AccessTokenPayload = map[string]interface{}{"sub": "example"}
	return p
}

func (p *OIDCProvider) ValidAccessToken() string {
	buf, _ := json.Marshal(p.AccessTokenPayload)
	return p.fakeToken(buf)
}

func (p *OIDCProvider) serveOIDC(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	p.c.Logf("serveOIDC: got req: %s %s %s", req.Method, req.URL, req.Form)
	w.Header().Set("Content-Type", "application/json")
	switch req.URL.Path {
	case "/.well-known/openid-configuration":
		configuration := map[string]interface{}{
			"issuer":                 p.Issuer.URL,
			"authorization_endpoint": p.Issuer.URL + "/auth",
			"token_endpoint":         p.Issuer.URL + "/token",
			"jwks_uri":               p.Issuer.URL + "/jwks",
			"userinfo_endpoint":      p.Issuer.URL + "/userinfo",
		}
		if p.EndSessionEndpoint == nil {
			// Not included in configuration
		} else if p.EndSessionEndpoint.Scheme != "" {
			configuration["end_session_endpoint"] = p.EndSessionEndpoint.String()
		} else {
			u, err := url.Parse(p.Issuer.URL)
			p.c.Check(err, check.IsNil,
				check.Commentf("error parsing IssuerURL for EndSessionEndpoint"))
			u.Scheme = "https"
			u.Path = u.Path + p.EndSessionEndpoint.Path
			configuration["end_session_endpoint"] = u.String()
		}
		json.NewEncoder(w).Encode(configuration)
	case "/token":
		var clientID, clientSecret string
		auth, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(req.Header.Get("Authorization"), "Basic "))
		authsplit := strings.Split(string(auth), ":")
		if len(authsplit) == 2 {
			clientID, _ = url.QueryUnescape(authsplit[0])
			clientSecret, _ = url.QueryUnescape(authsplit[1])
		}
		if clientID != p.ValidClientID || clientSecret != p.ValidClientSecret {
			p.c.Logf("OIDCProvider: expected (%q, %q) got (%q, %q)", p.ValidClientID, p.ValidClientSecret, clientID, clientSecret)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if req.Form.Get("code") != p.ValidCode || p.ValidCode == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		idToken, _ := json.Marshal(map[string]interface{}{
			"iss":            p.Issuer.URL,
			"aud":            []string{clientID},
			"sub":            "fake-user-id",
			"exp":            time.Now().UTC().Add(time.Minute).Unix(),
			"iat":            time.Now().UTC().Unix(),
			"nonce":          "fake-nonce",
			"email":          p.AuthEmail,
			"email_verified": p.AuthEmailVerified,
			"name":           p.AuthName,
			"given_name":     p.AuthGivenName,
			"family_name":    p.AuthFamilyName,
			"alt_verified":   true,                    // for custom claim tests
			"alt_email":      "alt_email@example.com", // for custom claim tests
			"alt_username":   "desired-username",      // for custom claim tests
		})
		json.NewEncoder(w).Encode(struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int32  `json:"expires_in"`
			IDToken      string `json:"id_token"`
		}{
			AccessToken:  p.ValidAccessToken(),
			TokenType:    "Bearer",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    30,
			IDToken:      p.fakeToken(idToken),
		})
	case "/jwks":
		json.NewEncoder(w).Encode(jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				{Key: p.key.Public(), Algorithm: string(jose.RS256), KeyID: ""},
			},
		})
	case "/auth":
		w.WriteHeader(http.StatusInternalServerError)
	case "/userinfo":
		if p.HoldUserInfo != nil {
			p.HoldUserInfo <- req
		}
		if p.ReleaseUserInfo != nil {
			<-p.ReleaseUserInfo
		}
		if p.UserInfoErrorStatus > 0 {
			w.WriteHeader(p.UserInfoErrorStatus)
			fmt.Fprintf(w, "%T error body", p)
			return
		}
		authhdr := req.Header.Get("Authorization")
		if _, err := jwt.ParseSigned(strings.TrimPrefix(authhdr, "Bearer ")); err != nil {
			p.c.Logf("OIDCProvider: bad auth %q", authhdr)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":            "fake-user-id",
			"name":           p.AuthName,
			"given_name":     p.AuthGivenName,
			"family_name":    p.AuthFamilyName,
			"alt_username":   "desired-username",
			"email":          p.AuthEmail,
			"email_verified": p.AuthEmailVerified,
		})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *OIDCProvider) servePeopleAPI(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	p.c.Logf("servePeopleAPI: got req: %s %s %s", req.Method, req.URL, req.Form)
	w.Header().Set("Content-Type", "application/json")
	switch req.URL.Path {
	case "/v1/people/me":
		if f := req.Form.Get("personFields"); f != "emailAddresses,names" {
			w.WriteHeader(http.StatusBadRequest)
			break
		}
		json.NewEncoder(w).Encode(p.PeopleAPIResponse)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *OIDCProvider) fakeToken(payload []byte) string {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: p.key}, nil)
	if err != nil {
		p.c.Error(err)
		return ""
	}
	object, err := signer.Sign(payload)
	if err != nil {
		p.c.Error(err)
		return ""
	}
	t, err := object.CompactSerialize()
	if err != nil {
		p.c.Error(err)
		return ""
	}
	p.c.Logf("fakeToken(%q) == %q", payload, t)
	return t
}
