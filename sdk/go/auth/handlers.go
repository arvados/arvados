// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"net/http"
)

type contextKey string

var contextKeyCredentials contextKey = "credentials"

// LoadToken wraps the next handler, adding credentials to the request
// context so subsequent handlers can access them efficiently via
// CredentialsFromRequest.
func LoadToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(contextKeyCredentials).(*Credentials); !ok {
			r = r.WithContext(context.WithValue(r.Context(), contextKeyCredentials, CredentialsFromRequest(r)))
		}
		next.ServeHTTP(w, r)
	})
}

// RequireLiteralToken wraps the next handler, rejecting any request
// that doesn't supply the given token. If the given token is empty,
// RequireLiteralToken returns next (i.e., no auth checks are
// performed).
func RequireLiteralToken(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := CredentialsFromRequest(r)
		if len(c.Tokens) == 0 {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		for _, t := range c.Tokens {
			if t == token {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	})
}
