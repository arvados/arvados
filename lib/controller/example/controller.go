// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package example

import (
	"context"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

func New(cluster *arvados.Cluster) *Controller {
	return &Controller{
		cluster: cluster,
	}
}

type Controller struct {
	cluster *arvados.Cluster
}

func (ctrl *Controller) ExampleCount(ctx context.Context, opts arvados.ExampleCountOptions) (resp arvados.ExampleCountResponse, err error) {
	tx, err := ctrlctx.CurrentTx(ctx)
	if err != nil {
		return
	}
	err = tx.QueryRowContext(ctx, `select count(*) from users`).Scan(&resp.Count)
	return
}

func (ctrl *Controller) ExampleGet(ctx context.Context, opts arvados.GetOptions) (resp arvados.Example, err error) {
	resp.UUID = opts.UUID
	resp.HairStyle = "bob"
	return
}
