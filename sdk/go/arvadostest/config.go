// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"bytes"
	"os"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	. "gopkg.in/check.v1"
)

func DefaultCluster(c *C, clusterID string) arvados.Cluster {
	logger := ctxlog.New(os.Stderr, "text", "info")
	confdata := []byte(`Clusters: {zzzzz: {}}`)
	loader := config.NewLoader(bytes.NewBuffer(confdata), logger)
	loader.SkipLegacy = true
	loader.Path = "-"
	cfg, err := loader.Load()
	c.Assert(err, IsNil)
	return cfg.Clusters["zzzzz"]
}
