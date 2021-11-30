// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dblock

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
)

var (
	TrashSweep = &DBLocker{key: 10001}
	retryDelay = 5 * time.Second
)

// DBLocker uses pg_advisory_lock to maintain a cluster-wide lock for
// a long-running task like "do X every N seconds".
type DBLocker struct {
	key   int
	mtx   sync.Mutex
	ctx   context.Context
	getdb func(context.Context) (*sqlx.DB, error)
	conn  *sql.Conn // != nil if advisory lock has been acquired
}

// Lock acquires the advisory lock, waiting/reconnecting if needed.
func (dbl *DBLocker) Lock(ctx context.Context, getdb func(context.Context) (*sqlx.DB, error)) {
	logger := ctxlog.FromContext(ctx)
	for ; ; time.Sleep(retryDelay) {
		dbl.mtx.Lock()
		if dbl.conn != nil {
			// Another goroutine is already locked/waiting
			// on this lock. Wait for them to release.
			dbl.mtx.Unlock()
			continue
		}
		db, err := getdb(ctx)
		if err != nil {
			logger.WithError(err).Infof("error getting database pool")
			dbl.mtx.Unlock()
			continue
		}
		conn, err := db.Conn(ctx)
		if err != nil {
			logger.WithError(err).Info("error getting database connection")
			dbl.mtx.Unlock()
			continue
		}
		var locked bool
		err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, dbl.key).Scan(&locked)
		if err != nil {
			logger.WithError(err).Infof("error getting pg_try_advisory_lock %d", dbl.key)
			conn.Close()
			dbl.mtx.Unlock()
			continue
		}
		if !locked {
			conn.Close()
			dbl.mtx.Unlock()
			continue
		}
		logger.Debugf("acquired pg_advisory_lock %d", dbl.key)
		dbl.ctx, dbl.getdb, dbl.conn = ctx, getdb, conn
		dbl.mtx.Unlock()
		return
	}
}

// Check confirms that the lock is still active (i.e., the session is
// still alive), and re-acquires if needed. Panics if Lock is not
// acquired first.
func (dbl *DBLocker) Check() {
	dbl.mtx.Lock()
	err := dbl.conn.PingContext(dbl.ctx)
	if err == nil {
		ctxlog.FromContext(dbl.ctx).Debugf("pg_advisory_lock %d connection still alive", dbl.key)
		dbl.mtx.Unlock()
		return
	}
	ctxlog.FromContext(dbl.ctx).WithError(err).Info("database connection ping failed")
	dbl.conn.Close()
	dbl.conn = nil
	ctx, getdb := dbl.ctx, dbl.getdb
	dbl.mtx.Unlock()
	dbl.Lock(ctx, getdb)
}

func (dbl *DBLocker) Unlock() {
	dbl.mtx.Lock()
	defer dbl.mtx.Unlock()
	if dbl.conn != nil {
		_, err := dbl.conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, dbl.key)
		if err != nil {
			ctxlog.FromContext(dbl.ctx).WithError(err).Infof("error releasing pg_advisory_lock %d", dbl.key)
		} else {
			ctxlog.FromContext(dbl.ctx).Debugf("released pg_advisory_lock %d", dbl.key)
		}
		dbl.conn.Close()
		dbl.conn = nil
	}
}
