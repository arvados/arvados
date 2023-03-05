// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

// ContainerUpdate defers to railsProxy and then notifies the
// container priority updater thread.
func (conn *Conn) ContainerUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.Container, error) {
	resp, err := conn.railsProxy.ContainerUpdate(ctx, opts)
	if err == nil {
		select {
		case conn.wantContainerPriorityUpdate <- struct{}{}:
		default:
			// update already pending
		}
	}
	return resp, err
}

// runContainerPriorityUpdateThread periodically (and immediately
// after each container update request) corrects any inconsistent
// container priorities caused by races.
func (conn *Conn) runContainerPriorityUpdateThread(ctx context.Context) {
	ctx = ctrlctx.NewWithToken(ctx, conn.cluster, conn.cluster.SystemRootToken)
	log := ctxlog.FromContext(ctx).WithField("worker", "runContainerPriorityUpdateThread")
	ticker := time.NewTicker(5 * time.Minute)
	for {
		err := conn.containerPriorityUpdate(ctx, log)
		if err != nil {
			log.WithError(err).Warn("error updating container priorities")
		}
		select {
		case <-ticker.C:
		case <-conn.wantContainerPriorityUpdate:
		}
	}
}

func (conn *Conn) containerPriorityUpdate(ctx context.Context, log logrus.FieldLogger) error {
	db, err := conn.getdb(ctx)
	if err != nil {
		return fmt.Errorf("getdb: %w", err)
	}
	res, err := db.ExecContext(ctx, `
		UPDATE containers AS c
		SET priority=0
		WHERE state IN ('Queued', 'Locked', 'Running')
		 AND priority>0
		 AND uuid NOT IN (
			SELECT container_uuid
			FROM container_requests
			WHERE priority > 0
			 AND state = 'Committed')`)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	} else if rows, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("update: %w", err)
	} else if rows > 0 {
		log.Infof("found %d containers with no active requests but priority>0, updated to priority=0", rows)
	}
	// In this loop we look for a single container that needs
	// fixing, call out to Rails to fix it, and repeat until we
	// don't find any more.
	//
	// We could get a batch of UUIDs that need attention by
	// increasing LIMIT 1, however, updating priority on one
	// container typically cascades to other containers, so we
	// would often end up repeating work.
	for lastUUID := ""; ; {
		var uuid string
		err := db.QueryRowxContext(ctx, `
			SELECT containers.uuid from containers
			JOIN container_requests
			 ON container_requests.container_uuid=containers.uuid
			 AND container_requests.state = 'Committed' AND container_requests.priority > 0
			WHERE containers.state IN ('Queued', 'Locked', 'Running')
			 AND containers.priority = 0
			 AND container_requests.uuid IS NOT NULL
			LIMIT 1`).Scan(&uuid)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			return fmt.Errorf("join: %w", err)
		}
		if uuid == lastUUID {
			// We don't want to keep hammering this
			// forever if the ContainerPriorityUpdate call
			// didn't achieve anything.
			return fmt.Errorf("possible lack of progress: container %s still has priority=0 after updating", uuid)
		}
		lastUUID = uuid
		_, err = conn.railsProxy.ContainerPriorityUpdate(ctx, arvados.UpdateOptions{UUID: uuid, Select: []string{"uuid"}})
		if err != nil {
			return err
		}
	}
	return nil
}
