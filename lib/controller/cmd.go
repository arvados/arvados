// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/lib/service"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var Command cmd.Handler = service.Command(arvados.ServiceNameController, newHandler)

func newHandler(cluster *arvados.Cluster, np *arvados.NodeProfile) service.Handler {
	return &Handler{Cluster: cluster, NodeProfile: np}
}
