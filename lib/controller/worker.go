// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"database/sql"
	"time"

	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
)

const (
	// lock keys should be added here with explicit values, to
	// ensure they do not get accidentally renumbered when a key
	// is added or removed.
	lockKeyTrashSweep = 10001
)

// dbLocker uses pg_advisory_lock to maintain a cluster-wide lock for
// a long-running task like "do X every N seconds".
type dbLocker struct {
	GetDB   func(context.Context) (*sqlx.DB, error)
	LockKey int

	conn *sql.Conn // != nil if advisory lock is acquired
}

// Lock acquires the advisory lock the first time it is
// called. Subsequent calls confirm that the lock is still active
// (i.e., the session is still alive), and re-acquires if needed.
func (dbl *dbLocker) Lock(ctx context.Context) {
	logger := ctxlog.FromContext(ctx)
	for ; ; time.Sleep(5 * time.Second) {
		if dbl.conn == nil {
			db, err := dbl.GetDB(ctx)
			if err != nil {
				logger.WithError(err).Infof("error getting database pool")
				continue
			}
			conn, err := db.Conn(ctx)
			if err != nil {
				logger.WithError(err).Info("error getting database connection")
				continue
			}
			_, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, dbl.LockKey)
			if err != nil {
				logger.WithError(err).Info("error getting lock")
				conn.Close()
				continue
			}
			dbl.conn = conn
		}
		err := dbl.conn.PingContext(ctx)
		if err != nil {
			logger.WithError(err).Info("database connection ping failed")
			dbl.conn.Close()
			dbl.conn = nil
			continue
		}
		return
	}
}

func (dbl *dbLocker) Unlock() {
	if dbl.conn != nil {
		dbl.conn.Close()
		dbl.conn = nil
	}
}

func (h *Handler) trashSweepWorker() {
	sleep := h.Cluster.Collections.TrashSweepInterval.Duration()
	logger := ctxlog.FromContext(h.BackgroundContext).WithField("worker", "trash sweep")
	ctx := ctxlog.Context(h.BackgroundContext, logger)
	if sleep <= 0 {
		logger.Debugf("Collections.TrashSweepInterval is %v, not running worker", sleep)
		return
	}
	locker := &dbLocker{GetDB: h.db, LockKey: lockKeyTrashSweep}
	locker.Lock(ctx)
	defer locker.Unlock()
	for time.Sleep(sleep); ctx.Err() == nil; time.Sleep(sleep) {
		locker.Lock(ctx)
		ctx := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{h.Cluster.SystemRootToken}})
		_, err := h.federation.SysTrashSweep(ctx, struct{}{})
		if err != nil {
			logger.WithError(err).Info("trash sweep failed")
		}
	}
}
