// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package githttpd

import (
	"context"

	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"github.com/prometheus/client_golang/prometheus"
)

var Command = service.Command(arvados.ServiceNameGitHTTP, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) service.Handler {
	client, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, err)
	}
	ac, err := arvadosclient.New(client)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, err)
	}
	return &authHandler{
		clientPool: &arvadosclient.ClientPool{Prototype: ac},
		cluster:    cluster,
		handler:    newGitHandler(ctx, cluster),
	}
}
