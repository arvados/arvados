// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"git.arvados.org/arvados.git/lib/dispatchcloud/scheduler"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

// ContainerRequestCreate defers to railsProxy for everything except
// vocabulary checking.
func (conn *Conn) ContainerRequestCreate(ctx context.Context, opts arvados.CreateOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.ContainerRequest{}, err
	}
	resp, err := conn.railsProxy.ContainerRequestCreate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// ContainerRequestUpdate defers to railsProxy for everything except
// vocabulary checking.
func (conn *Conn) ContainerRequestUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.ContainerRequest{}, err
	}
	resp, err := conn.railsProxy.ContainerRequestUpdate(ctx, opts)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (conn *Conn) ContainerRequestGet(ctx context.Context, opts arvados.GetOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.ContainerRequestGet(ctx, opts)
}

func (conn *Conn) ContainerRequestList(ctx context.Context, opts arvados.ListOptions) (arvados.ContainerRequestList, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.ContainerRequestList(ctx, opts)
}

func (conn *Conn) ContainerRequestDelete(ctx context.Context, opts arvados.DeleteOptions) (arvados.ContainerRequest, error) {
	conn.logActivity(ctx)
	return conn.railsProxy.ContainerRequestDelete(ctx, opts)
}

func (conn *Conn) ContainerRequestContainerStatus(ctx context.Context, opts arvados.GetOptions) (arvados.ContainerStatus, error) {
	conn.logActivity(ctx)
	var ret arvados.ContainerStatus
	cr, err := conn.railsProxy.ContainerRequestGet(ctx, arvados.GetOptions{UUID: opts.UUID, Select: []string{"uuid", "container_uuid", "log_uuid"}})
	if err != nil {
		return ret, err
	}
	if cr.ContainerUUID == "" {
		ret.SchedulingStatus = "No container is assigned."
		return ret, nil
	}
	// We use admin credentials to get the container record so we
	// don't get an error when we're in a race with auto-retry and
	// the container became user-unreadable since we fetched the
	// CR above.
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})
	ctr, err := conn.railsProxy.ContainerGet(ctxRoot, arvados.GetOptions{UUID: cr.ContainerUUID, Select: []string{"uuid", "state", "priority"}})
	if err != nil {
		return ret, err
	}
	ret.UUID = ctr.UUID
	ret.State = ctr.State
	if ctr.State != arvados.ContainerStateQueued && ctr.State != arvados.ContainerStateLocked {
		// Scheduling status is not a thing once the container
		// is in running state.
		return ret, nil
	}
	var lastErr error
	for dispatchurl := range conn.cluster.Services.DispatchCloud.InternalURLs {
		baseurl := url.URL(dispatchurl)
		apiurl, err := baseurl.Parse("/arvados/v1/dispatch/container?container_uuid=" + cr.ContainerUUID)
		if err != nil {
			lastErr = err
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiurl.String(), nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Authorization", "Bearer "+conn.cluster.ManagementToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("error getting status from dispatcher: %w", err)
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			continue
		} else if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("error getting status from dispatcher: %s", resp.Status)
			continue
		}
		var qent scheduler.QueueEnt
		err = json.NewDecoder(resp.Body).Decode(&qent)
		if err != nil {
			lastErr = err
			continue
		}
		ret.State = qent.Container.State // Prefer dispatcher's view of state if not equal to ctr.State
		ret.SchedulingStatus = qent.SchedulingStatus
		return ret, nil
	}
	if lastErr != nil {
		// If we got a non-nil error from a dispatchcloud
		// service, and the container state suggests
		// dispatchcloud should know about it, then we return
		// an error so the client knows to retry.
		return ret, httpserver.ErrorWithStatus(lastErr, http.StatusBadGateway)
	}
	// All running dispatchcloud services confirm they don't have
	// this container (the dispatcher hasn't yet noticed it
	// appearing in the queue) or there are no dispatchcloud
	// services configured. Either way, all we can say is that
	// it's queued.
	if ctr.State == arvados.ContainerStateQueued && ctr.Priority < 1 {
		// If it hasn't been picked up by a dispatcher
		// already, it won't be -- it's just on hold.
		// Scheduling status does not apply.
		return ret, nil
	}
	ret.SchedulingStatus = "Waiting in queue."
	return ret, nil
}
