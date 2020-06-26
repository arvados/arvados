// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"database/sql"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	_ "github.com/lib/pq"
	check "gopkg.in/check.v1"
)

// testdb returns a DB connection for the given cluster config.
func testdb(c *check.C, cluster *arvados.Cluster) *sql.DB {
	db, err := sql.Open("postgres", cluster.PostgreSQL.Connection.String())
	c.Assert(err, check.IsNil)
	return db
}

// testctx returns a context suitable for running a test case in a new
// transaction, and a rollback func which the caller should call after
// the test.
func testctx(c *check.C, db *sql.DB) (ctx context.Context, rollback func()) {
	tx, err := db.Begin()
	c.Assert(err, check.IsNil)
	return Transaction(context.Background(), tx), func() {
		c.Check(tx.Rollback(), check.IsNil)
	}
}
