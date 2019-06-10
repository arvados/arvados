// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"os"

	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/lib/service"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
)

var (
	version             = "dev"
	command cmd.Handler = service.Command(arvados.ServiceNameController, newHandler)
)

func newHandler(ctx context.Context, cluster *arvados.Cluster, _ string) service.Handler {
	return &health.Aggregator{Cluster: cluster}
}

func main() {
	os.Exit(command.RunCommand(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
