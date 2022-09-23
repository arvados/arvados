// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ctrlctx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/controller/api"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"github.com/ghodss/yaml"
)

var (
	ErrNoAuthContext   = errors.New("bug: there is no authorization in this context")
	ErrUnauthenticated = errors.New("unauthenticated request")
)

// WrapCallsWithAuth returns a call wrapper (suitable for assigning to
// router.router.WrapCalls) that makes CurrentUser(ctx) et al. work
// from inside the wrapped functions.
//
// The incoming context must come from WrapCallsInTransactions or
// NewWithTransaction.
func WrapCallsWithAuth(cluster *arvados.Cluster) func(api.RoutableFunc) api.RoutableFunc {
	var authcache authcache
	return func(origFunc api.RoutableFunc) api.RoutableFunc {
		return func(ctx context.Context, opts interface{}) (_ interface{}, err error) {
			var tokens []string
			if creds, ok := auth.FromContext(ctx); ok {
				tokens = creds.Tokens
			}
			return origFunc(context.WithValue(ctx, contextKeyAuth, &authcontext{
				authcache: &authcache,
				cluster:   cluster,
				tokens:    tokens,
			}), opts)
		}
	}
}

// CurrentAuth returns the arvados.User whose privileges should be
// used in the given context, and the arvados.APIClientAuthorization
// the caller presented in order to authenticate the current request.
//
// Returns ErrUnauthenticated if the current request was not
// authenticated (no token provided, token is expired, etc).
func CurrentAuth(ctx context.Context) (*arvados.User, *arvados.APIClientAuthorization, error) {
	ac, ok := ctx.Value(contextKeyAuth).(*authcontext)
	if !ok {
		return nil, nil, ErrNoAuthContext
	}
	ac.lookupOnce.Do(func() {
		// We only validate/lookup the token once per API
		// call, even though authcache should be efficient
		// enough to do a lookup each time. This guarantees we
		// always return the same result when called multiple
		// times in the course of handling a single API call.
		for _, token := range ac.tokens {
			user, aca, err := ac.authcache.lookup(ctx, ac.cluster, token)
			if err != nil {
				ac.err = err
				return
			}
			if user != nil {
				ac.user, ac.apiClientAuthorization = user, aca
				return
			}
		}
		ac.err = ErrUnauthenticated
	})
	return ac.user, ac.apiClientAuthorization, ac.err
}

type contextKeyA string

var contextKeyAuth = contextKeyT("auth")

type authcontext struct {
	authcache              *authcache
	cluster                *arvados.Cluster
	tokens                 []string
	user                   *arvados.User
	apiClientAuthorization *arvados.APIClientAuthorization
	err                    error
	lookupOnce             sync.Once
}

var authcacheTTL = time.Minute

type authcacheent struct {
	expireTime             time.Time
	apiClientAuthorization arvados.APIClientAuthorization
	user                   arvados.User
}

type authcache struct {
	mtx         sync.Mutex
	entries     map[string]*authcacheent
	nextCleanup time.Time
}

// lookup returns the user and aca info for a given token. Returns nil
// if the token is not valid. Returns a non-nil error if there was an
// unexpected error from the database, etc.
func (ac *authcache) lookup(ctx context.Context, cluster *arvados.Cluster, token string) (*arvados.User, *arvados.APIClientAuthorization, error) {
	ac.mtx.Lock()
	ent := ac.entries[token]
	ac.mtx.Unlock()
	if ent != nil && ent.expireTime.After(time.Now()) {
		return &ent.user, &ent.apiClientAuthorization, nil
	}
	if token == "" {
		return nil, nil, nil
	}
	tx, err := CurrentTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	var aca arvados.APIClientAuthorization
	var user arvados.User

	var cond string
	var args []interface{}
	if len(token) > 30 && strings.HasPrefix(token, "v2/") && token[30] == '/' {
		fields := strings.Split(token, "/")
		cond = `aca.uuid = $1 and aca.api_token = $2`
		args = []interface{}{fields[1], fields[2]}
	} else {
		// Bare token or OIDC access token
		mac := hmac.New(sha256.New, []byte(cluster.SystemRootToken))
		io.WriteString(mac, token)
		hmac := fmt.Sprintf("%x", mac.Sum(nil))
		cond = `aca.api_token in ($1, $2)`
		args = []interface{}{token, hmac}
	}
	var expiresAt sql.NullTime
	var scopesYAML []byte
	err = tx.QueryRowContext(ctx, `
select aca.uuid, aca.expires_at, aca.api_token, aca.scopes, users.uuid, users.is_active, users.is_admin
 from api_client_authorizations aca
 left join users on aca.user_id = users.id
 where `+cond+`
 and (expires_at is null or expires_at > current_timestamp at time zone 'UTC')`, args...).Scan(
		&aca.UUID, &expiresAt, &aca.APIToken, &scopesYAML,
		&user.UUID, &user.IsActive, &user.IsAdmin)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}
	aca.ExpiresAt = expiresAt.Time
	if len(scopesYAML) > 0 {
		err = yaml.Unmarshal(scopesYAML, &aca.Scopes)
		if err != nil {
			return nil, nil, fmt.Errorf("loading scopes for %s: %w", aca.UUID, err)
		}
	}
	ent = &authcacheent{
		expireTime:             time.Now().Add(authcacheTTL),
		apiClientAuthorization: aca,
		user:                   user,
	}
	ac.mtx.Lock()
	defer ac.mtx.Unlock()
	if ac.entries == nil {
		ac.entries = map[string]*authcacheent{}
	}
	if ac.nextCleanup.IsZero() || ac.nextCleanup.Before(time.Now()) {
		for token, ent := range ac.entries {
			if !ent.expireTime.After(time.Now()) {
				delete(ac.entries, token)
			}
		}
		ac.nextCleanup = time.Now().Add(authcacheTTL)
	}
	ac.entries[token] = ent
	return &ent.user, &ent.apiClientAuthorization, nil
}
