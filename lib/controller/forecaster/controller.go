// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package forecaster

import (
	"context"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// New forecaster controller that has a cluster and an API server to connect to
func New(cluster *arvados.Cluster, parent arvados.API) *Controller {
	return &Controller{
		cluster: cluster,
		parent:  parent,
	}
}

// Controller struct is used by the cotroler to route 
type Controller struct {
	cluster *arvados.Cluster
	parent  arvados.API
}

// CheckpointsGet endpont as discribed in:
// https://dev.arvados.org/projects/arvados/wiki/API_HistoricalForcasting_data_for_CR#checkpoints
func (ctrl *Controller) CheckpointsGet(ctx context.Context, opts arvados.GetOptions) (resp arvados.Checkpoints, err error) {	
	resp.UUID = opts.UUID
	// fill in resp.Checkpoints ...
	return resp, nil
}

