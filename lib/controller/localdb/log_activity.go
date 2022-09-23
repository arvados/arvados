// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"time"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

func (conn *Conn) logActivity(ctx context.Context) {
	p := conn.cluster.Users.ActivityLoggingPeriod.Duration()
	if p < 1 {
		ctxlog.FromContext(ctx).Debug("logActivity disabled by config")
		return
	}
	user, _, err := ctrlctx.CurrentAuth(ctx)
	if err == ctrlctx.ErrUnauthenticated {
		ctxlog.FromContext(ctx).Debug("logActivity skipped for unauthenticated request")
		return
	} else if err != nil {
		ctxlog.FromContext(ctx).WithError(err).Error("logActivity CurrentAuth failed")
		return
	}
	now := time.Now()
	conn.activeUsersLock.Lock()
	if conn.activeUsers == nil || conn.activeUsersReset.IsZero() || conn.activeUsersReset.Before(now) {
		conn.activeUsersReset = alignedPeriod(now, p)
		conn.activeUsers = map[string]bool{}
	}
	logged := conn.activeUsers[user.UUID]
	if !logged {
		// Prevent other concurrent calls from logging about
		// this user until we finish.
		conn.activeUsers[user.UUID] = true
	}
	conn.activeUsersLock.Unlock()
	if logged {
		return
	}
	defer func() {
		// If we return without logging, reset the flag so we
		// try again on the user's next API call.
		if !logged {
			conn.activeUsersLock.Lock()
			conn.activeUsers[user.UUID] = false
			conn.activeUsersLock.Unlock()
		}
	}()

	tx, err := ctrlctx.NewTx(ctx)
	if err != nil {
		ctxlog.FromContext(ctx).WithError(err).Error("logActivity NewTx failed")
		return
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
insert into logs
 (uuid,
  owner_uuid, modified_by_user_uuid, object_owner_uuid,
  event_type,
  summary,
  object_uuid,
  properties,
  event_at, created_at, updated_at, modified_at)
 values
 ($1, $2, $2, $2, $3, $4, $5, $6,
  current_timestamp at time zone 'UTC',
  current_timestamp at time zone 'UTC',
  current_timestamp at time zone 'UTC',
  current_timestamp at time zone 'UTC')
 returning id`,
		arvados.RandomUUID(conn.cluster.ClusterID, "57u5n"),
		conn.cluster.ClusterID+"-tpzed-000000000000000", // both modified_by and object_owner
		"activity",
		"activity of "+user.UUID,
		user.UUID,
		"{}")
	if err != nil {
		ctxlog.FromContext(ctx).WithError(err).Error("logActivity query failed")
		return
	}
	err = tx.Commit()
	if err != nil {
		ctxlog.FromContext(ctx).WithError(err).Error("logActivity commit failed")
		return
	}
	logged = true
}

// alignedPeriod computes a time interval that includes now and aligns
// to local clock times that are multiples of p. For example, if local
// time is UTC-5 and ActivityLoggingPeriod=4h, periodStart and
// periodEnd will be 0000-0400, 0400-0800, etc., in local time. If p
// is a multiple of 24h, periods will start and end at midnight.
//
// If DST starts or ends during this period, the boundaries will be
// aligned based on either DST or non-DST time depending on whether
// now is before or after the DST transition. The consequences are
// presumed to be inconsequential, e.g., logActivity may unnecessarily
// log activity more than once in a period that includes a DST
// transition.
//
// In all cases, the period ends in the future.
//
// Only the end of the period is returned.
func alignedPeriod(now time.Time, p time.Duration) time.Time {
	_, tzsec := now.Zone()
	tzoff := time.Duration(tzsec) * time.Second
	periodStart := now.Add(tzoff).Truncate(p).Add(-tzoff)
	return periodStart.Add(p)
}
