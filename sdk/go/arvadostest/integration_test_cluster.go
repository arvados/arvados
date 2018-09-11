// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"os"
	"path/filepath"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

func IntegrationTestCluster(c *check.C) *arvados.Cluster {
	config, err := arvados.GetConfig(filepath.Join(os.Getenv("WORKSPACE"), "tmp", "arvados.yml"))
	c.Assert(err, check.IsNil)
	cluster, err := config.GetCluster("")
	c.Assert(err, check.IsNil)
	return cluster
}
