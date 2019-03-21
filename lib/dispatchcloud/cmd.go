// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"

	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/lib/service"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var Command cmd.Handler = service.Command(arvados.ServiceNameDispatchCloud, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, _ *arvados.NodeProfile) service.Handler {
	d := &dispatcher{
		Cluster:   cluster,
		Context:   ctx,
		AuthToken: service.Token(ctx),
	}
	go d.Start()
	return d
}
