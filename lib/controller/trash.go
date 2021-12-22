// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"time"

	"git.arvados.org/arvados.git/lib/controller/dblock"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

func (h *Handler) trashSweepWorker() {
	sleep := h.Cluster.Collections.TrashSweepInterval.Duration()
	logger := ctxlog.FromContext(h.BackgroundContext).WithField("worker", "trash sweep")
	ctx := ctxlog.Context(h.BackgroundContext, logger)
	if sleep <= 0 {
		logger.Debugf("Collections.TrashSweepInterval is %v, not running worker", sleep)
		return
	}
	dblock.TrashSweep.Lock(ctx, h.db)
	defer dblock.TrashSweep.Unlock()
	for time.Sleep(sleep); ctx.Err() == nil; time.Sleep(sleep) {
		dblock.TrashSweep.Check()
		ctx := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{h.Cluster.SystemRootToken}})
		_, err := h.federation.SysTrashSweep(ctx, struct{}{})
		if err != nil {
			logger.WithError(err).Info("trash sweep failed")
		}
	}
}
