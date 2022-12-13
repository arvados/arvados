// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dblock

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
)

var (
	TrashSweep         = &DBLocker{key: 10001}
	ContainerLogSweep  = &DBLocker{key: 10002}
	KeepBalanceService = &DBLocker{key: 10003} // keep-balance service in periodic-sweep loop
	KeepBalanceActive  = &DBLocker{key: 10004} // keep-balance sweep in progress (either -once=true or service loop)
	Dispatch           = &DBLocker{key: 10005} // any dispatcher running
	RailsMigrations    = &DBLocker{key: 10006}
	retryDelay         = 5 * time.Second
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
//
// Returns false if ctx is canceled before the lock is acquired.
func (dbl *DBLocker) Lock(ctx context.Context, getdb func(context.Context) (*sqlx.DB, error)) bool {
	logger := ctxlog.FromContext(ctx).WithField("ID", dbl.key)
	var lastHeldBy string
	for ; ; time.Sleep(retryDelay) {
		dbl.mtx.Lock()
		if dbl.conn != nil {
			// Another goroutine is already locked/waiting
			// on this lock. Wait for them to release.
			dbl.mtx.Unlock()
			continue
		}
		if ctx.Err() != nil {
			dbl.mtx.Unlock()
			return false
		}
		db, err := getdb(ctx)
		if err == context.Canceled {
			dbl.mtx.Unlock()
			return false
		} else if err != nil {
			logger.WithError(err).Info("error getting database pool")
			dbl.mtx.Unlock()
			continue
		}
		conn, err := db.Conn(ctx)
		if err == context.Canceled {
			dbl.mtx.Unlock()
			return false
		} else if err != nil {
			logger.WithError(err).Info("error getting database connection")
			dbl.mtx.Unlock()
			continue
		}
		var locked bool
		err = conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, dbl.key).Scan(&locked)
		if err == context.Canceled {
			return false
		} else if err != nil {
			logger.WithError(err).Info("error getting pg_try_advisory_lock")
			conn.Close()
			dbl.mtx.Unlock()
			continue
		}
		if !locked {
			var host string
			var port int
			err = conn.QueryRowContext(ctx, `SELECT client_addr, client_port FROM pg_stat_activity WHERE pid IN
				(SELECT pid FROM pg_locks
				 WHERE locktype = $1 AND objid = $2)`, "advisory", dbl.key).Scan(&host, &port)
			if err != nil {
				logger.WithError(err).Info("error getting other client info")
			} else {
				heldBy := net.JoinHostPort(host, fmt.Sprintf("%d", port))
				if lastHeldBy != heldBy {
					logger.WithField("DBClient", heldBy).Info("waiting for other process to release lock")
					lastHeldBy = heldBy
				}
			}
			conn.Close()
			dbl.mtx.Unlock()
			continue
		}
		logger.Debug("acquired pg_advisory_lock")
		dbl.ctx, dbl.getdb, dbl.conn = ctx, getdb, conn
		dbl.mtx.Unlock()
		return true
	}
}

// Check confirms that the lock is still active (i.e., the session is
// still alive), and re-acquires if needed. Panics if Lock is not
// acquired first.
//
// Returns false if the context passed to Lock() is canceled before
// the lock is confirmed or reacquired.
func (dbl *DBLocker) Check() bool {
	dbl.mtx.Lock()
	err := dbl.conn.PingContext(dbl.ctx)
	if err == context.Canceled {
		dbl.mtx.Unlock()
		return false
	} else if err == nil {
		ctxlog.FromContext(dbl.ctx).WithField("ID", dbl.key).Debug("connection still alive")
		dbl.mtx.Unlock()
		return true
	}
	ctxlog.FromContext(dbl.ctx).WithError(err).Info("database connection ping failed")
	dbl.conn.Close()
	dbl.conn = nil
	ctx, getdb := dbl.ctx, dbl.getdb
	dbl.mtx.Unlock()
	return dbl.Lock(ctx, getdb)
}

func (dbl *DBLocker) Unlock() {
	dbl.mtx.Lock()
	defer dbl.mtx.Unlock()
	if dbl.conn != nil {
		_, err := dbl.conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, dbl.key)
		if err != nil {
			ctxlog.FromContext(dbl.ctx).WithError(err).WithField("ID", dbl.key).Info("error releasing pg_advisory_lock")
		} else {
			ctxlog.FromContext(dbl.ctx).WithField("ID", dbl.key).Debug("released pg_advisory_lock")
		}
		dbl.conn.Close()
		dbl.conn = nil
	}
}
