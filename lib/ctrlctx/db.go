// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ctrlctx

import (
	"context"
	"errors"
	"sync"

	"git.arvados.org/arvados.git/lib/controller/api"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"

	// sqlx needs lib/pq to talk to PostgreSQL
	_ "github.com/lib/pq"
)

var (
	ErrNoTransaction   = errors.New("bug: there is no transaction in this context")
	ErrContextFinished = errors.New("refusing to start a transaction after wrapped function already returned")
)

// WrapCallsInTransactions returns a call wrapper (suitable for
// assigning to router.router.WrapCalls) that starts a new transaction
// for each API call, and commits only if the call succeeds.
//
// The wrapper calls getdb() to get a database handle before each API
// call.
func WrapCallsInTransactions(getdb func(context.Context) (*sqlx.DB, error)) func(api.RoutableFunc) api.RoutableFunc {
	return func(origFunc api.RoutableFunc) api.RoutableFunc {
		return func(ctx context.Context, opts interface{}) (_ interface{}, err error) {
			ctx, finishtx := New(ctx, getdb)
			defer finishtx(&err)
			return origFunc(ctx, opts)
		}
	}
}

// NewWithTransaction returns a child context in which the given
// transaction will be used by any localdb API call that needs one.
// The caller is responsible for calling Commit or Rollback on tx.
func NewWithTransaction(ctx context.Context, tx *sqlx.Tx) context.Context {
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

type finishFunc func(*error)

// New returns a new child context that can be used with
// CurrentTx(). It does not open a database transaction until the
// first call to CurrentTx().
//
// The caller must eventually call the returned finishtx() func to
// commit or rollback the transaction, if any.
//
//	func example(ctx context.Context) (err error) {
//		ctx, finishtx := New(ctx, dber)
//		defer finishtx(&err)
//		// ...
//		tx, err := CurrentTx(ctx)
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
func New(ctx context.Context, getdb func(context.Context) (*sqlx.DB, error)) (context.Context, finishFunc) {
	txn := &transaction{getdb: getdb}
	return context.WithValue(ctx, contextKeyTransaction, txn), func(err *error) {
		txn.setup.Do(func() {
			// Using (*sync.Once)Do() prevents a future
			// call to CurrentTx() from opening a
			// transaction which would never get committed
			// or rolled back. If CurrentTx() hasn't been
			// called before now, future calls will return
			// this error.
			txn.err = ErrContextFinished
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

// NewTx starts a new transaction. The caller is responsible for
// calling Commit or Rollback. This is suitable for database queries
// that are separate from the API transaction (see CurrentTx), e.g.,
// ones that will be committed even if the API call fails, or held
// open after the API call finishes.
func NewTx(ctx context.Context) (*sqlx.Tx, error) {
	txn, ok := ctx.Value(contextKeyTransaction).(*transaction)
	if !ok {
		return nil, ErrNoTransaction
	}
	db, err := txn.getdb(ctx)
	if err != nil {
		return nil, err
	}
	return db.Beginx()
}

// CurrentTx returns a transaction that will be committed after the
// current API call completes, or rolled back if the current API call
// returns an error.
func CurrentTx(ctx context.Context) (*sqlx.Tx, error) {
	txn, ok := ctx.Value(contextKeyTransaction).(*transaction)
	if !ok {
		return nil, ErrNoTransaction
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
