// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

func logout(ctx context.Context, cluster *arvados.Cluster, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	err := expireAPIClientAuthorization(ctx)
	if err != nil {
		ctxlog.FromContext(ctx).Errorf("attempting to expire token on logout: %q", err)
		return arvados.LogoutResponse{}, httpserver.ErrorWithStatus(errors.New("could not expire token on logout"), http.StatusInternalServerError)
	}

	target := opts.ReturnTo
	if target == "" {
		if cluster.Services.Workbench2.ExternalURL.Host != "" {
			target = cluster.Services.Workbench2.ExternalURL.String()
		} else {
			target = cluster.Services.Workbench1.ExternalURL.String()
		}
	} else if err := validateLoginRedirectTarget(cluster, target); err != nil {
		return arvados.LogoutResponse{}, httpserver.ErrorWithStatus(fmt.Errorf("invalid return_to parameter: %s", err), http.StatusBadRequest)
	}
	return arvados.LogoutResponse{RedirectLocation: target}, nil
}

func expireAPIClientAuthorization(ctx context.Context) error {
	creds, ok := auth.FromContext(ctx)
	if !ok {
		// Tests could be passing empty contexts
		ctxlog.FromContext(ctx).Debugf("expireAPIClientAuthorization: credentials not found from context")
		return nil
	}

	if len(creds.Tokens) == 0 {
		// Old client may not have provided the token to expire
		return nil
	}

	tx, err := ctrlctx.CurrentTx(ctx)
	if err != nil {
		return err
	}

	token := creds.Tokens[0]
	tokenSecret := token
	var tokenUuid string
	if strings.HasPrefix(token, "v2/") {
		tokenParts := strings.Split(token, "/")
		if len(tokenParts) >= 3 {
			tokenUuid = tokenParts[1]
			tokenSecret = tokenParts[2]
		}
	}

	var retrievedUuid string
	err = tx.QueryRowContext(ctx, `SELECT uuid FROM api_client_authorizations WHERE api_token=$1 AND (expires_at IS NULL OR expires_at > current_timestamp AT TIME ZONE 'UTC') LIMIT 1`, tokenSecret).Scan(&retrievedUuid)
	if err == sql.ErrNoRows {
		ctxlog.FromContext(ctx).Debugf("expireAPIClientAuthorization(%s): not found in database", token)
		return nil
	} else if err != nil {
		ctxlog.FromContext(ctx).WithError(err).Debugf("expireAPIClientAuthorization(%s): database error", token)
		return err
	}

	if tokenUuid != "" && retrievedUuid != tokenUuid {
		// secret part matches, but UUID doesn't -- somewhat surprising
		ctxlog.FromContext(ctx).Debugf("expireAPIClientAuthorization(%s): secret part found, but with different UUID: %s", tokenSecret, retrievedUuid)
		return nil
	}

	res, err := tx.ExecContext(ctx, "UPDATE api_client_authorizations SET expires_at=current_timestamp AT TIME ZONE 'UTC' WHERE uuid=$1", retrievedUuid)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		ctxlog.FromContext(ctx).Debugf("expireAPIClientAuthorization(%s): no rows were updated", tokenSecret)
		return fmt.Errorf("couldn't expire provided token")
	} else if rows > 1 {
		ctxlog.FromContext(ctx).Debugf("expireAPIClientAuthorization(%s): multiple (%d) rows updated", tokenSecret, rows)
	} else {
		ctxlog.FromContext(ctx).Debugf("expireAPIClientAuthorization(%s): ok", tokenSecret)
	}

	return nil
}
