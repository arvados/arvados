// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"net/url"
	"os"
	"testing"

	"git.arvados.org/arvados.git/lib/controller/router"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

// FederationSuite does some generic setup/teardown. Don't add Test*
// methods to FederationSuite itself.
type FederationSuite struct {
	cluster *arvados.Cluster
	ctx     context.Context
	fed     *Conn
}

func (s *FederationSuite) SetUpTest(c *check.C) {
	s.cluster = &arvados.Cluster{
		ClusterID:       "aaaaa",
		SystemRootToken: arvadostest.SystemRootToken,
		RemoteClusters: map[string]arvados.RemoteCluster{
			"aaaaa": arvados.RemoteCluster{
				Host: os.Getenv("ARVADOS_API_HOST"),
			},
		},
	}
	arvadostest.SetServiceURL(&s.cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	s.cluster.TLS.Insecure = true
	s.cluster.API.MaxItemsPerResponse = 3

	ctx := context.Background()
	ctx = ctxlog.Context(ctx, ctxlog.TestLogger(c))
	ctx = auth.NewContext(ctx, &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})
	s.ctx = ctx

	s.fed = New(s.cluster)
}

func (s *FederationSuite) addDirectRemote(c *check.C, id string, backend backend) {
	s.cluster.RemoteClusters[id] = arvados.RemoteCluster{
		Host: "in-process.local",
	}
	s.fed.remotes[id] = backend
}

func (s *FederationSuite) addHTTPRemote(c *check.C, id string, backend backend) {
	srv := httpserver.Server{Addr: ":"}
	srv.Handler = router.New(backend, nil)
	c.Check(srv.Start(), check.IsNil)
	s.cluster.RemoteClusters[id] = arvados.RemoteCluster{
		Scheme: "http",
		Host:   srv.Addr,
		Proxy:  true,
	}
	s.fed.remotes[id] = rpc.NewConn(id, &url.URL{Scheme: "http", Host: srv.Addr}, true, saltedTokenProvider(s.fed.local, id))
}
