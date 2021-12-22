// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	check "gopkg.in/check.v1"
)

// TestMissedCollections exercises EachCollection's sanity check:
// #collections processed >= #old collections that exist in database
// after processing.
func (s *integrationSuite) TestMissedCollections(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	db, err := sqlx.Open("postgres", cluster.PostgreSQL.Connection.String())
	c.Assert(err, check.IsNil)

	defer db.Exec(`delete from collections where uuid = 'zzzzz-4zz18-404040404040404'`)
	insertedOld := false
	err = EachCollection(context.Background(), db, s.client, func(coll arvados.Collection) error {
		if !insertedOld {
			insertedOld = true
			_, err := db.Exec(`insert into collections (uuid, created_at, updated_at, modified_at) values ('zzzzz-4zz18-404040404040404', '2002-02-02T02:02:02Z', '2002-02-02T02:02:02Z', '2002-02-02T02:02:02Z')`)
			return err
		}
		return nil
	}, nil)
	c.Check(err, check.ErrorMatches, `Retrieved .* collections .* but server now reports .* collections.*`)
}
