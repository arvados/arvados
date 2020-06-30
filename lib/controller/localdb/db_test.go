// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
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
	return ContextWithTransaction(context.Background(), tx), func() {
		c.Check(tx.Rollback(), check.IsNil)
	}
}

var _ = check.Suite(&DatabaseSuite{})

type DatabaseSuite struct{}

func (*DatabaseSuite) TestTransactionContext(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)

	var getterCalled int64
	getter := func(context.Context) (*sql.DB, error) {
		atomic.AddInt64(&getterCalled, 1)
		return testdb(c, cluster), nil
	}
	wrapper := WrapCallsInTransactions(getter)
	wrappedFunc := wrapper(func(ctx context.Context, opts interface{}) (interface{}, error) {
		txes := make([]*sql.Tx, 20)
		var wg sync.WaitGroup
		for i := range txes {
			i := i
			wg.Add(1)
			go func() {
				// Concurrent calls to currenttx(),
				// with different children of the same
				// parent context, will all return the
				// same transaction.
				defer wg.Done()
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				tx, err := currenttx(ctx)
				c.Check(err, check.IsNil)
				txes[i] = tx
			}()
		}
		wg.Wait()
		for i := range txes[1:] {
			c.Check(txes[i], check.Equals, txes[i+1])
		}
		return true, nil
	})

	ok, err := wrappedFunc(context.Background(), "blah")
	c.Check(ok, check.Equals, true)
	c.Check(err, check.IsNil)
	c.Check(getterCalled, check.Equals, int64(1))

	// When a wrapped func returns without calling currenttx(),
	// calling currenttx() later shouldn't start a new
	// transaction.
	var savedctx context.Context
	ok, err = wrapper(func(ctx context.Context, opts interface{}) (interface{}, error) {
		savedctx = ctx
		return true, nil
	})(context.Background(), "blah")
	c.Check(ok, check.Equals, true)
	c.Check(err, check.IsNil)
	tx, err := currenttx(savedctx)
	c.Check(tx, check.IsNil)
	c.Check(err, check.NotNil)
}
