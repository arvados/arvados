// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"context"
	"mime"
	"os"

	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
)

var Command = service.Command(arvados.ServiceNameKeepweb, newHandlerOrErrorHandler)

func newHandlerOrErrorHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) service.Handler {
	h, err := newHandler(ctx, cluster, token, reg)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, err)
	}
	return h
}

func newHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) (*handler, error) {
	logger := ctxlog.FromContext(ctx)
	if ext := ".txt"; mime.TypeByExtension(ext) == "" {
		logger.Warnf("cannot look up MIME type for %q -- this probably means /etc/mime.types is missing -- clients will see incorrect content types", ext)
	}

	keepclient.RefreshServiceDiscoveryOnSIGHUP()
	os.Setenv("ARVADOS_API_HOST", cluster.Services.Controller.ExternalURL.Host)
	return &handler{
		Cluster: cluster,
		Cache: cache{
			cluster:  cluster,
			logger:   logger,
			registry: reg,
		},
	}, nil
}
