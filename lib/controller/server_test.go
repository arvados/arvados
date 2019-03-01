// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"net/http"
	"os"
	"path/filepath"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
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

	nodeProfile := arvados.NodeProfile{
		Controller: arvados.SystemServiceInstance{Listen: ":"},
		RailsAPI:   arvados.SystemServiceInstance{Listen: os.Getenv("ARVADOS_TEST_API_HOST"), TLS: true, Insecure: true},
	}
	handler := &Handler{Cluster: &arvados.Cluster{
		ClusterID:  "zzzzz",
		PostgreSQL: integrationTestCluster().PostgreSQL,
		NodeProfiles: map[string]arvados.NodeProfile{
			"*": nodeProfile,
		},
	}, NodeProfile: &nodeProfile}

	srv := &httpserver.Server{
		Server: http.Server{
			Handler: httpserver.AddRequestIDs(httpserver.LogRequests(log, handler)),
		},
		Addr: nodeProfile.Controller.Listen,
	}
	return srv
}
