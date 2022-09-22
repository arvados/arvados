// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/jmoiron/sqlx"

	// sqlx needs lib/pq to talk to PostgreSQL
	_ "github.com/lib/pq"
	"gopkg.in/check.v1"
)

// DB returns a DB connection for the given cluster config.
func DB(c *check.C, cluster *arvados.Cluster) *sqlx.DB {
	db, err := sqlx.Open("postgres", cluster.PostgreSQL.Connection.String())
	c.Assert(err, check.IsNil)
	return db
}
