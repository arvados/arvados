// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

func (conn *Conn) expireAPIClientAuthorization(ctx context.Context) error {
	creds, ok := auth.FromContext(ctx)
	if !ok {
		return errors.New("credentials not found from context")
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

	res, err := tx.ExecContext(ctx, "UPDATE api_client_authorizations SET expires_at=current_timestamp AT TIME ZONE 'UTC' WHERE api_token=$1 AND (expires_at IS NULL OR expires_at > current_timestamp AT TIME ZONE 'UTC')", tokenSecret)
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
