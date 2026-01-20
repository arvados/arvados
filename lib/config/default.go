// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	. "gopkg.in/check.v1"
)

func DefaultCluster(c *C, clusterID string) (arvados.Cluster, error) {
	logger := ctxlog.New(os.Stderr, "text", "info")
	confdata := []byte(`Clusters: {zzzzz: {}}`)
	loader := NewLoader(bytes.NewBuffer(confdata), logger)
	loader.SkipLegacy = true
	loader.Path = "-"
	cfg, err := loader.Load()
	if err != nil {
		return arvados.Cluster{}, err
	}
	return cfg.Clusters["zzzzz"], nil
}
