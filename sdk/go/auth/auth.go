// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

type Credentials struct {
	Tokens []string
}

func NewCredentials(tokens ...string) *Credentials {
	return &Credentials{Tokens: tokens}
}

func NewContext(ctx context.Context, c *Credentials) context.Context {
	return context.WithValue(ctx, contextKeyCredentials{}, c)
}

func FromContext(ctx context.Context) (*Credentials, bool) {
	c, ok := ctx.Value(contextKeyCredentials{}).(*Credentials)
	return c, ok
}

func CredentialsFromRequest(r *http.Request) *Credentials {
	if c, ok := FromContext(r.Context()); ok {
		// preloaded by middleware
		return c
	}
	c := NewCredentials()
	c.LoadTokensFromHTTPRequest(r)
	return c
}

// EncodeTokenCookie accepts a token and returns a byte slice suitable
// for use as a cookie value, such that it will be decoded correctly
// by LoadTokensFromHTTPRequest.
var EncodeTokenCookie func([]byte) string = base64.URLEncoding.EncodeToString

// DecodeTokenCookie accepts a cookie value and returns the encoded
// token.
var DecodeTokenCookie func(string) ([]byte, error) = base64.URLEncoding.DecodeString

// LoadTokensFromHTTPRequest loads all tokens it can find in the
// headers and query string of an http query.
func (a *Credentials) LoadTokensFromHTTPRequest(r *http.Request) {
	// Load plain token from "Authorization: Bearer ..." header
	// (typically used by smart API clients).  Note many pre-3.0
	// clients send "OAuth2 ..." instead of "Bearer ..." and that
	// is still accepted.
	if toks := strings.SplitN(r.Header.Get("Authorization"), " ", 2); len(toks) == 2 && (toks[0] == "OAuth2" || toks[0] == "Bearer") {
		a.Tokens = append(a.Tokens, strings.TrimSpace(toks[1]))
	}

	// Load base64-encoded token from "Authorization: Basic ..."
	// header (typically used by git via credential helper)
	if _, password, ok := r.BasicAuth(); ok {
		a.Tokens = append(a.Tokens, strings.TrimSpace(password))
	}

	// Load tokens from query string. It's generally not a good
	// idea to pass tokens around this way, but passing a narrowly
	// scoped token is a reasonable way to implement "secret link
	// to an object" in a generic way.
	//
	// ParseQuery always returns a non-nil map which might have
	// valid parameters, even when a decoding error causes it to
	// return a non-nil err. We ignore err; hopefully the caller
	// will also need to parse the query string for
	// application-specific purposes and will therefore
	// find/report decoding errors in a suitable way.
	qvalues, _ := url.ParseQuery(r.URL.RawQuery)
	if val, ok := qvalues["api_token"]; ok {
		for _, token := range val {
			a.Tokens = append(a.Tokens, strings.TrimSpace(token))
		}
	}

	a.loadTokenFromCookie(r)

	// TODO: Load token from Rails session cookie (if Rails site
	// secret is known)
}

func (a *Credentials) loadTokenFromCookie(r *http.Request) {
	cookie, err := r.Cookie("arvados_api_token")
	if err != nil || len(cookie.Value) == 0 {
		return
	}
	token, err := DecodeTokenCookie(cookie.Value)
	if err != nil {
		return
	}
	a.Tokens = append(a.Tokens, strings.TrimSpace(string(token)))
}

// LoadTokensFromHTTPRequestBody loads credentials from the request
// body.
//
// This is separate from LoadTokensFromHTTPRequest() because it's not
// always desirable to read the request body. This has to be requested
// explicitly by the application.
func (a *Credentials) LoadTokensFromHTTPRequestBody(r *http.Request) error {
	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		return nil
	}
	if err := r.ParseForm(); err != nil {
		return err
	}
	if t := r.PostFormValue("api_token"); t != "" {
		a.Tokens = append(a.Tokens, strings.TrimSpace(t))
	}
	return nil
}
