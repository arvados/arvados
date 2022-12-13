// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"bytes"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"gopkg.in/check.v1"
)

type railsDBSuite struct{}

var _ = check.Suite(&railsDBSuite{})

// Check services/api/db/migrate/*.rb match schema_migrations
func (s *railsDBSuite) TestMigrationList(c *check.C) {
	var logbuf bytes.Buffer
	log := ctxlog.New(&logbuf, "text", "info")
	todo, err := migrationList("../../services/api", log)
	c.Check(err, check.IsNil)
	c.Check(todo["20220804133317"], check.Equals, true)
	c.Check(logbuf.String(), check.Equals, "")

	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	db := arvadostest.DB(c, cluster)
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	for rows.Next() {
		var v string
		err = rows.Scan(&v)
		c.Assert(err, check.IsNil)
		if !todo[v] {
			c.Errorf("version is in schema_migrations but not services/api/db/migrate/: %q", v)
		}
		delete(todo, v)
	}
	err = rows.Close()
	c.Assert(err, check.IsNil)

	// In the test suite, the database should be fully migrated.
	// So, if there's anything left in todo here, there is
	// something wrong with our "db/migrate/*.rb ==
	// schema_migrations" reasoning.
	c.Check(todo, check.HasLen, 0)
}
