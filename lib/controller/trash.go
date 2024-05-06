// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"time"

	"git.arvados.org/arvados.git/lib/controller/dblock"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

func (h *Handler) periodicWorker(workerName string, interval time.Duration, locker *dblock.DBLocker, run func(context.Context) error) {
	logger := ctxlog.FromContext(h.BackgroundContext).WithField("worker", workerName)
	ctx := ctxlog.Context(h.BackgroundContext, logger)
	if interval <= 0 {
		logger.Debugf("interval is %v, not running worker", interval)
		return
	}
	if !locker.Lock(ctx, h.dbConnector.GetDB) {
		// context canceled
		return
	}
	defer locker.Unlock()
	for ctxSleep(ctx, interval); ctx.Err() == nil; ctxSleep(ctx, interval) {
		if !locker.Check() {
			// context canceled
			return
		}
		err := run(ctx)
		if err != nil {
			logger.WithError(err).Infof("%s failed", workerName)
		}
	}
}

func (h *Handler) trashSweepWorker() {
	h.periodicWorker("trash sweep", h.Cluster.Collections.TrashSweepInterval.Duration(), dblock.TrashSweep, func(ctx context.Context) error {
		ctx = auth.NewContext(ctx, &auth.Credentials{Tokens: []string{h.Cluster.SystemRootToken}})
		_, err := h.federation.SysTrashSweep(ctx, struct{}{})
		return err
	})
}

func (h *Handler) containerLogSweepWorker() {
	// Since #21611 we don't expect any new log entries, so the
	// periodic worker only runs once, then becomes a no-op.
	//
	// The old Containers.Logging.SweepInterval config is removed.
	// We use TrashSweepInterval here instead, for testing
	// reasons: it prevents the default integration-testing
	// controller service (whose TrashSweepInterval is 0) from
	// acquiring the dblock.
	done := false
	h.periodicWorker("container log sweep", h.Cluster.Collections.TrashSweepInterval.Duration(), dblock.ContainerLogSweep, func(ctx context.Context) error {
		if done {
			return nil
		}
		db, err := h.dbConnector.GetDB(ctx)
		if err != nil {
			return err
		}
		res, err := db.ExecContext(ctx, `
DELETE FROM logs
 USING containers
 WHERE logs.object_uuid=containers.uuid
 AND logs.event_type in ('stdout', 'stderr', 'arv-mount', 'crunch-run', 'crunchstat', 'hoststat', 'node', 'container', 'keepstore')
 AND containers.log IS NOT NULL`)
		if err != nil {
			return err
		}
		logger := ctxlog.FromContext(ctx)
		rows, err := res.RowsAffected()
		if err != nil {
			logger.WithError(err).Warn("unexpected error from RowsAffected()")
		} else {
			logger.WithField("rows", rows).Info("deleted rows from logs table")
			if rows == 0 {
				done = true
			}
		}
		return nil
	})
}

// Sleep for the given duration, but return early if ctx cancels
// before that.
func ctxSleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
