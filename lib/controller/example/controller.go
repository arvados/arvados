// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package example

import (
	"context"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

func New(cluster *arvados.Cluster, parent arvados.API) *Controller {
	return &Controller{
		cluster: cluster,
		parent:  parent,
	}
}

type Controller struct {
	cluster *arvados.Cluster
	parent  arvados.API
}

// @Description This method is used as an example to create new endpoints. The response are meaningless, just a help for a developer to know how to interact with the database and create new endpoints.
// @Success  200  object  arvados.ExampleCountResponse  "kind:arvados#exampleCountResponse JSON"
// @Failure  400  object  arvados.ExampleCountResponse  "kind:arvados#exampleCountResponse JSON"
// @Resource example
// @Route /arvados/v1/examples/count [get]
func (ctrl *Controller) ExampleCount(ctx context.Context, opts arvados.ExampleCountOptions) (resp arvados.ExampleCountResponse, err error) {
	// Example of direct database access
	tx, err := ctrlctx.CurrentTx(ctx)
	if err != nil {
		return
	}
	err = tx.QueryRowContext(ctx, `select count(*) from users`).Scan(&resp.Count)
	if err != nil {
		return
	}

	// Example of calling other controller APIs that are
	// implemented in different packages
	userlist, err := ctrl.parent.UserList(ctx, arvados.ListOptions{Limit: 0, Count: "exact"})
	if err != nil {
		return
	}
	resp.Count += userlist.ItemsAvailable

	return
}

func (ctrl *Controller) ExampleGet(ctx context.Context, opts arvados.GetOptions) (resp arvados.Example, err error) {
	resp.UUID = opts.UUID
	resp.HairStyle = "bob"
	return
}
