// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ctrlctx

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"git.arvados.org/arvados.git/lib/controller/api"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
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
	return func(origFunc api.RoutableFunc) api.RoutableFunc {
		return func(ctx context.Context, opts interface{}) (_ interface{}, err error) {
			var tokens []string
			if creds, ok := auth.FromContext(ctx); ok {
				tokens = creds.Tokens
			}
			return origFunc(context.WithValue(ctx, contextKeyAuth, &authcontext{cluster: cluster, tokens: tokens}), opts)
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
	ac.lookupOnce.Do(func() { ac.user, ac.apiClientAuthorization, ac.err = aclookup(ctx, ac.cluster, ac.tokens) })
	return ac.user, ac.apiClientAuthorization, ac.err
}

type contextKeyA string

var contextKeyAuth = contextKeyT("auth")

type authcontext struct {
	cluster                *arvados.Cluster
	tokens                 []string
	user                   *arvados.User
	apiClientAuthorization *arvados.APIClientAuthorization
	err                    error
	lookupOnce             sync.Once
}

func aclookup(ctx context.Context, cluster *arvados.Cluster, tokens []string) (*arvados.User, *arvados.APIClientAuthorization, error) {
	if len(tokens) == 0 {
		return nil, nil, ErrUnauthenticated
	}
	tx, err := CurrentTx(ctx)
	if err != nil {
		return nil, nil, err
	}
	var aca arvados.APIClientAuthorization
	var user arvados.User
	for _, token := range tokens {
		var cond string
		var args []interface{}
		if token == "" {
			continue
		} else if len(token) > 30 && strings.HasPrefix(token, "v2/") && token[30] == '/' {
			fields := strings.Split(token, "/")
			cond = `aca.uuid=$1 and aca.api_token=$2`
			args = []interface{}{fields[1], fields[2]}
		} else {
			// Bare token or OIDC access token
			mac := hmac.New(sha256.New, []byte(cluster.SystemRootToken))
			io.WriteString(mac, token)
			hmac := fmt.Sprintf("%x", mac.Sum(nil))
			cond = `aca.api_token in ($1, $2)`
			args = []interface{}{token, hmac}
		}
		var scopesJSON []byte
		err = tx.QueryRowContext(ctx, `
select aca.uuid, aca.expires_at, aca.api_token, aca.scopes, users.uuid, users.is_active, users.is_admin
 from api_client_authorizations aca
 left join users on aca.user_id = users.id
 where `+cond+`
 and (expires_at is null or expires_at > current_timestamp at time zone 'UTC')`, args...).Scan(
			&aca.UUID, &aca.ExpiresAt, &aca.APIToken, &scopesJSON,
			&user.UUID, &user.IsActive, &user.IsAdmin)
		if err == sql.ErrNoRows {
			continue
		} else if err != nil {
			return nil, nil, err
		}
		if len(scopesJSON) > 0 {
			err = json.Unmarshal(scopesJSON, &aca.Scopes)
			if err != nil {
				return nil, nil, err
			}
		}
		return &user, &aca, nil
	}
	return nil, nil, ErrUnauthenticated
}
