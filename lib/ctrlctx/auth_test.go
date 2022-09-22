// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ctrlctx

import (
	"context"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	check "gopkg.in/check.v1"
)

func (*DatabaseSuite) TestAuthContext(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)

	getter := func(context.Context) (*sqlx.DB, error) {
		return sqlx.Open("postgres", cluster.PostgreSQL.Connection.String())
	}
	authwrapper := WrapCallsWithAuth(cluster)
	dbwrapper := WrapCallsInTransactions(getter)

	// valid tokens
	for _, token := range []string{
		"3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi",
		"v2/zzzzz-gj3su-077z32aux8dg2s1/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi",
		"v2/zzzzz-gj3su-077z32aux8dg2s1/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi/asdfasdfasdf",
	} {
		ok, err := dbwrapper(authwrapper(func(ctx context.Context, opts interface{}) (interface{}, error) {
			user, aca, err := CurrentAuth(ctx)
			if c.Check(err, check.IsNil) {
				c.Check(user.UUID, check.Equals, "zzzzz-tpzed-xurymjxw79nv3jz")
				c.Check(aca.UUID, check.Equals, "zzzzz-gj3su-077z32aux8dg2s1")
				c.Check(aca.Scopes, check.DeepEquals, []string{"all"})
			}
			return true, nil
		}))(auth.NewContext(context.Background(), auth.NewCredentials(token)), "blah")
		c.Check(ok, check.Equals, true)
		c.Check(err, check.IsNil)
	}

	// bad tokens
	for _, token := range []string{
		"3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmI", // note last char mangled
		"v2/zzzzz-gj3su-077z32aux8dg2s1/",
		"bogus",
		"",
	} {
		ok, err := dbwrapper(authwrapper(func(ctx context.Context, opts interface{}) (interface{}, error) {
			user, aca, err := CurrentAuth(ctx)
			c.Check(err, check.Equals, ErrUnauthenticated)
			c.Check(user, check.IsNil)
			c.Check(aca, check.IsNil)
			return true, err
		}))(auth.NewContext(context.Background(), auth.NewCredentials(token)), "blah")
		c.Check(ok, check.Equals, true)
		c.Check(err, check.Equals, ErrUnauthenticated)
	}

	// no auth context
	{
		ok, err := dbwrapper(authwrapper(func(ctx context.Context, opts interface{}) (interface{}, error) {
			user, aca, err := CurrentAuth(ctx)
			c.Check(err, check.Equals, ErrUnauthenticated)
			c.Check(user, check.IsNil)
			c.Check(aca, check.IsNil)
			return true, err
		}))(context.Background(), "blah")
		c.Check(ok, check.Equals, true)
		c.Check(err, check.Equals, ErrUnauthenticated)
	}
}
