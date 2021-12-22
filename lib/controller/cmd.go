// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/prometheus/client_golang/prometheus"
)

// Command starts a controller service. See cmd/arvados-server/cmd.go
var Command cmd.Handler = service.Command(arvados.ServiceNameController, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, _ string, _ *prometheus.Registry) service.Handler {
	return &Handler{Cluster: cluster, BackgroundContext: ctx}
}
