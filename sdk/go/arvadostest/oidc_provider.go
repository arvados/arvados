// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
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
	AccessTokenPayload map[string]interface{}

	PeopleAPIResponse map[string]interface{}

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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                 p.Issuer.URL,
			"authorization_endpoint": p.Issuer.URL + "/auth",
			"token_endpoint":         p.Issuer.URL + "/token",
			"jwks_uri":               p.Issuer.URL + "/jwks",
			"userinfo_endpoint":      p.Issuer.URL + "/userinfo",
		})
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
		authhdr := req.Header.Get("Authorization")
		if _, err := jwt.ParseSigned(strings.TrimPrefix(authhdr, "Bearer ")); err != nil {
			p.c.Logf("OIDCProvider: bad auth %q", authhdr)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":            "fake-user-id",
			"name":           p.AuthName,
			"given_name":     p.AuthName,
			"family_name":    "",
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
