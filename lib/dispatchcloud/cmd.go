// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"fmt"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/prometheus/client_golang/prometheus"
)

var Command cmd.Handler = service.Command(arvados.ServiceNameDispatchCloud, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) service.Handler {
	ac, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, fmt.Errorf("error initializing client from cluster config: %s", err))
	}
	// Disable auto-retry.  We have transient failure recovery at
	// the application level, so we would rather receive/report
	// upstream errors right away.
	ac.Timeout = 0
	d := &dispatcher{
		Cluster:   cluster,
		Context:   ctx,
		ArvClient: ac,
		AuthToken: token,
		Registry:  reg,
	}
	go d.Start()
	return d
}
