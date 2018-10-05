// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"path/filepath"
)

// IntegrationTestCluster returns the cluster that has been set up by
// the integration test framework (see /build/run-tests.sh). It panics
// on error.
func IntegrationTestCluster() *Cluster {
	config, err := GetConfig(filepath.Join(os.Getenv("WORKSPACE"), "tmp", "arvados.yml"))
	if err != nil {
		panic(err)
	}
	cluster, err := config.GetCluster("")
	if err != nil {
		panic(err)
	}
	return cluster
}
