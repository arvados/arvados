// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"
	"sync"

	"git.arvados.org/arvados.git/lib/controller/router"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
)

// WrapCallsInTransactions returns a call wrapper (suitable for
// assigning to router.router.WrapCalls) that starts a new transaction
// for each API call, and commits only if the call succeeds.
//
// The wrapper calls getdb() to get a database handle before each API
// call.
func WrapCallsInTransactions(getdb func(context.Context) (*sqlx.DB, error)) func(router.RoutableFunc) router.RoutableFunc {
	return func(origFunc router.RoutableFunc) router.RoutableFunc {
		return func(ctx context.Context, opts interface{}) (_ interface{}, err error) {
			ctx, finishtx := starttx(ctx, getdb)
			defer finishtx(&err)
			return origFunc(ctx, opts)
		}
	}
}

// ContextWithTransaction returns a child context in which the given
// transaction will be used by any localdb API call that needs one.
// The caller is responsible for calling Commit or Rollback on tx.
func ContextWithTransaction(ctx context.Context, tx *sqlx.Tx) context.Context {
	txn := &transaction{tx: tx}
	txn.setup.Do(func() {})
	return context.WithValue(ctx, contextKeyTransaction, txn)
}

type contextKeyT string

var contextKeyTransaction = contextKeyT("transaction")

type transaction struct {
	tx    *sqlx.Tx
	err   error
	getdb func(context.Context) (*sqlx.DB, error)
	setup sync.Once
}

type transactionFinishFunc func(*error)

// starttx returns a new child context that can be used with
// currenttx(). It does not open a database transaction until the
// first call to currenttx().
//
// The caller must eventually call the returned finishtx() func to
// commit or rollback the transaction, if any.
//
//	func example(ctx context.Context) (err error) {
//		ctx, finishtx := starttx(ctx, dber)
//		defer finishtx(&err)
//		// ...
//		tx, err := currenttx(ctx)
//		if err != nil {
//			return fmt.Errorf("example: %s", err)
//		}
//		return tx.ExecContext(...)
//	}
//
// If *err is nil, finishtx() commits the transaction and assigns any
// resulting error to *err.
//
// If *err is non-nil, finishtx() rolls back the transaction, and
// does not modify *err.
func starttx(ctx context.Context, getdb func(context.Context) (*sqlx.DB, error)) (context.Context, transactionFinishFunc) {
	txn := &transaction{getdb: getdb}
	return context.WithValue(ctx, contextKeyTransaction, txn), func(err *error) {
		txn.setup.Do(func() {
			// Using (*sync.Once)Do() prevents a future
			// call to currenttx() from opening a
			// transaction which would never get committed
			// or rolled back. If currenttx() hasn't been
			// called before now, future calls will return
			// this error.
			txn.err = errors.New("refusing to start a transaction after wrapped function already returned")
		})
		if txn.tx == nil {
			// we never [successfully] started a transaction
			return
		}
		if *err != nil {
			ctxlog.FromContext(ctx).Debug("rollback")
			txn.tx.Rollback()
			return
		}
		*err = txn.tx.Commit()
	}
}

func currenttx(ctx context.Context) (*sqlx.Tx, error) {
	txn, ok := ctx.Value(contextKeyTransaction).(*transaction)
	if !ok {
		return nil, errors.New("bug: there is no transaction in this context")
	}
	txn.setup.Do(func() {
		if db, err := txn.getdb(ctx); err != nil {
			txn.err = err
		} else {
			txn.tx, txn.err = db.Beginx()
		}
	})
	return txn.tx, txn.err
}
