// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	check "gopkg.in/check.v1"
)

func integrationTestCluster() *arvados.Cluster {
	cfg, err := arvados.GetConfig(filepath.Join(os.Getenv("WORKSPACE"), "tmp", "arvados.yml"))
	if err != nil {
		panic(err)
	}
	cc, err := cfg.GetCluster("zzzzz")
	if err != nil {
		panic(err)
	}
	return cc
}

// Return a new unstarted controller server, using the Rails API
// provided by the integration-testing environment.
func newServerFromIntegrationTestEnv(c *check.C) *httpserver.Server {
	log := ctxlog.TestLogger(c)
	ctx := ctxlog.Context(context.Background(), log)
	handler := &Handler{
		Cluster: &arvados.Cluster{
			ClusterID:  "zzzzz",
			PostgreSQL: integrationTestCluster().PostgreSQL,
		},
		BackgroundContext: ctx,
	}
	handler.Cluster.TLS.Insecure = true
	handler.Cluster.Collections.BlobSigning = true
	handler.Cluster.Collections.BlobSigningKey = arvadostest.BlobSigningKey
	handler.Cluster.Collections.BlobSigningTTL = arvados.Duration(time.Hour * 24 * 14)
	arvadostest.SetServiceURL(&handler.Cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	arvadostest.SetServiceURL(&handler.Cluster.Services.Controller, "http://localhost:/")

	srv := &httpserver.Server{
		Server: http.Server{
			BaseContext: func(net.Listener) context.Context { return ctx },
			Handler:     httpserver.AddRequestIDs(httpserver.LogRequests(handler)),
		},
		Addr: ":",
	}
	return srv
}
