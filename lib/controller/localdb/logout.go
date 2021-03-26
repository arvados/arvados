// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

func (conn *Conn) ExpireAPIClientAuthorization(ctx context.Context) error {
	aca, err := conn.railsProxy.APIClientAuthorizationCurrent(ctx, arvados.GetOptions{})
	if err != nil {
		return err
	}
	tx, err := ctrlctx.CurrentTx(ctx)
	if err != nil {
		return err
	}

	err = tx.QueryRowxContext(ctx, "UPDATE api_client_authorizations SET expires_at=current_timestamp WHERE uuid=$1", aca.UUID).Err()
	if err != nil {
		return err
	}

	return nil
}
