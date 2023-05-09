// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"context"
	"fmt"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
)

var testMode = false

var Command cmd.Handler = service.Command(arvados.ServiceNameWebsocket, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) service.Handler {
	client, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, fmt.Errorf("error initializing client from cluster config: %s", err))
	}
	client.Timeout = time.Minute
	eventSource := &pgEventSource{
		DataSource:   cluster.PostgreSQL.Connection.String(),
		MaxOpenConns: cluster.PostgreSQL.ConnectionPool,
		QueueSize:    cluster.API.WebsocketServerEventQueue,
		Logger:       ctxlog.FromContext(ctx),
		Reg:          reg,
	}
	done := make(chan struct{})
	go func() {
		eventSource.Run()
		ctxlog.FromContext(ctx).Error("event source stopped")
		close(done)
	}()
	eventSource.WaitReady()
	if err := eventSource.DBHealth(); err != nil {
		return service.ErrorHandler(ctx, cluster, err)
	}
	rtr := &router{
		cluster:        cluster,
		client:         client,
		eventSource:    eventSource,
		newPermChecker: func() permChecker { return newPermChecker(*client) },
		done:           done,
		reg:            reg,
	}
	return rtr
}
