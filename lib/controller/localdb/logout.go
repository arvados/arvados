// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/auth"
)

func (conn *Conn) ExpireAPIClientAuthorization(ctx context.Context) error {
	creds, ok := auth.FromContext(ctx)

	if !ok {
		return errors.New("credentials not found from context")
	}

	if len(creds.Tokens) < 1 {
		return errors.New("no tokens found to expire")
	}

	token := creds.Tokens[0]
	tokensecret := token
	if strings.Contains(token, "/") {
		tokenparts := strings.Split(token, "/")
		if len(tokenparts) >= 3 {
			tokensecret = tokenparts[2]
		}
	}

	tx, err := ctrlctx.CurrentTx(ctx)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, "UPDATE api_client_authorizations SET expires_at=current_timestamp WHERE api_token=$1", tokensecret)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("token expiration affected rows: %d -  token: %s", rows, token)
	}

	return nil
}
