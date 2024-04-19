// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"context"
	"errors"

	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Command = service.Command(arvados.ServiceNameKeepstore, newHandlerOrErrorHandler)
)

func newHandlerOrErrorHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) service.Handler {
	serviceURL, ok := service.URLFromContext(ctx)
	if !ok {
		return service.ErrorHandler(ctx, cluster, errors.New("BUG: no URL from service.URLFromContext"))
	}
	ks, err := newKeepstore(ctx, cluster, token, reg, serviceURL)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, err)
	}
	puller := newPuller(ctx, ks, reg)
	trasher := newTrasher(ctx, ks, reg)
	_ = newTrashEmptier(ctx, ks, reg)
	return newRouter(ks, puller, trasher)
}
