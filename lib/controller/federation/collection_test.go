// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"fmt"
	"net/http"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&collectionSuite{})

type collectionSuite struct {
	FederationSuite
}

func (s *collectionSuite) TestMultipleBackendFailureStatus(c *check.C) {
	nxPDH := "a4f995dd0c08216f37cb1bdec990f0cd+1234"
	s.cluster.ClusterID = "local"
	for _, trial := range []struct {
		label        string
		token        string
		localStatus  int
		remoteStatus map[string]int
		expectStatus int
	}{
		{
			"all backends return 404 => 404",
			arvadostest.SystemRootToken,
			http.StatusNotFound,
			map[string]int{
				"aaaaa": http.StatusNotFound,
				"bbbbb": http.StatusNotFound,
			},
			http.StatusNotFound,
		},
		{
			"all backends return 401 => 401 (e.g., bad token)",
			arvadostest.SystemRootToken,
			http.StatusUnauthorized,
			map[string]int{
				"aaaaa": http.StatusUnauthorized,
				"bbbbb": http.StatusUnauthorized,
			},
			http.StatusUnauthorized,
		},
		{
			"local 404, remotes 403 => 422 (mix of non-retryable errors)",
			arvadostest.SystemRootToken,
			http.StatusNotFound,
			map[string]int{
				"aaaaa": http.StatusForbidden,
				"bbbbb": http.StatusForbidden,
			},
			http.StatusUnprocessableEntity,
		},
		{
			"local 404, remotes 401/403/404 => 422 (mix of non-retryable errors)",
			arvadostest.SystemRootToken,
			http.StatusNotFound,
			map[string]int{
				"aaaaa": http.StatusUnauthorized,
				"bbbbb": http.StatusForbidden,
				"ccccc": http.StatusNotFound,
			},
			http.StatusUnprocessableEntity,
		},
		{
			"local 404, remotes 401/403/500 => 502 (at least one remote is retryable)",
			arvadostest.SystemRootToken,
			http.StatusNotFound,
			map[string]int{
				"aaaaa": http.StatusUnauthorized,
				"bbbbb": http.StatusForbidden,
				"ccccc": http.StatusInternalServerError,
			},
			http.StatusBadGateway,
		},
	} {
		c.Logf("trial: %v", trial)
		s.fed = New(s.ctx, s.cluster, nil, (&ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}).GetDB)
		s.fed.local = &arvadostest.APIStub{Error: httpserver.ErrorWithStatus(fmt.Errorf("stub error %d", trial.localStatus), trial.localStatus)}
		for id, status := range trial.remoteStatus {
			s.addDirectRemote(c, id, &arvadostest.APIStub{Error: httpserver.ErrorWithStatus(fmt.Errorf("stub error %d", status), status)})
		}

		ctx := context.Background()
		ctx = ctxlog.Context(ctx, ctxlog.TestLogger(c))
		if trial.token != "" {
			ctx = auth.NewContext(ctx, &auth.Credentials{Tokens: []string{trial.token}})
		}

		_, err := s.fed.CollectionGet(s.ctx, arvados.GetOptions{UUID: nxPDH})
		c.Check(err.(httpserver.HTTPStatusError).HTTPStatus(), check.Equals, trial.expectStatus)
	}
}
