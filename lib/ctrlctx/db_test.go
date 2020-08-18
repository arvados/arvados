// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ctrlctx

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&DatabaseSuite{})

type DatabaseSuite struct{}

func (*DatabaseSuite) TestTransactionContext(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)

	var getterCalled int64
	getter := func(context.Context) (*sqlx.DB, error) {
		atomic.AddInt64(&getterCalled, 1)
		db, err := sqlx.Open("postgres", cluster.PostgreSQL.Connection.String())
		c.Assert(err, check.IsNil)
		return db, nil
	}
	wrapper := WrapCallsInTransactions(getter)
	wrappedFunc := wrapper(func(ctx context.Context, opts interface{}) (interface{}, error) {
		txes := make([]*sqlx.Tx, 20)
		var wg sync.WaitGroup
		for i := range txes {
			i := i
			wg.Add(1)
			go func() {
				// Concurrent calls to CurrentTx(),
				// with different children of the same
				// parent context, will all return the
				// same transaction.
				defer wg.Done()
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				tx, err := CurrentTx(ctx)
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

	// When a wrapped func returns without calling CurrentTx(),
	// calling CurrentTx() later shouldn't start a new
	// transaction.
	var savedctx context.Context
	ok, err = wrapper(func(ctx context.Context, opts interface{}) (interface{}, error) {
		savedctx = ctx
		return true, nil
	})(context.Background(), "blah")
	c.Check(ok, check.Equals, true)
	c.Check(err, check.IsNil)
	tx, err := CurrentTx(savedctx)
	c.Check(tx, check.IsNil)
	c.Check(err, check.NotNil)
}
