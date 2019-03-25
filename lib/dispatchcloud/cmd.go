// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"fmt"

	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/lib/service"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var Command cmd.Handler = service.Command(arvados.ServiceNameDispatchCloud, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, np *arvados.NodeProfile, token string) service.Handler {
	ac, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, np, fmt.Errorf("error initializing client from cluster config: %s", err))
	}
	d := &dispatcher{
		Cluster:   cluster,
		Context:   ctx,
		ArvClient: ac,
		AuthToken: token,
	}
	go d.Start()
	return d
}
