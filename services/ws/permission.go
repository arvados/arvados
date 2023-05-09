// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

const (
	maxPermCacheAge = time.Hour
	minPermCacheAge = 5 * time.Minute
)

type permChecker interface {
	SetToken(token string)
	Check(ctx context.Context, uuid string) (bool, error)
}

func newPermChecker(ac arvados.Client) permChecker {
	ac.AuthToken = ""
	return &cachingPermChecker{
		Client:     &ac,
		cache:      make(map[string]cacheEnt),
		maxCurrent: 16,
	}
}

type cacheEnt struct {
	time.Time
	allowed bool
}

type cachingPermChecker struct {
	*arvados.Client
	cache      map[string]cacheEnt
	maxCurrent int

	nChecks  uint64
	nMisses  uint64
	nInvalid uint64
}

func (pc *cachingPermChecker) SetToken(token string) {
	if pc.Client.AuthToken == token {
		return
	}
	pc.Client.AuthToken = token
	pc.cache = make(map[string]cacheEnt)
}

func (pc *cachingPermChecker) Check(ctx context.Context, uuid string) (bool, error) {
	pc.nChecks++
	logger := ctxlog.FromContext(ctx).
		WithField("token", pc.Client.AuthToken).
		WithField("uuid", uuid)
	pc.tidy()
	now := time.Now()
	if perm, ok := pc.cache[uuid]; ok && now.Sub(perm.Time) < maxPermCacheAge {
		logger.WithField("allowed", perm.allowed).Debug("cache hit")
		return perm.allowed, nil
	}
	var buf map[string]interface{}
	path, err := pc.PathForUUID("get", uuid)
	if err != nil {
		pc.nInvalid++
		return false, err
	}

	pc.nMisses++
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	defer cancel()
	err = pc.RequestAndDecodeContext(ctx, &buf, "GET", path, nil, url.Values{
		"include_trash": {"true"},
		"select":        {`["uuid"]`},
	})

	var allowed bool
	if err == nil {
		allowed = true
	} else if txErr, ok := err.(*arvados.TransactionError); ok && pc.isNotAllowed(txErr.StatusCode) {
		allowed = false
	} else {
		logger.WithError(err).Error("lookup error")
		return false, err
	}
	logger.WithField("allowed", allowed).Debug("cache miss")
	pc.cache[uuid] = cacheEnt{Time: now, allowed: allowed}
	return allowed, nil
}

func (pc *cachingPermChecker) isNotAllowed(status int) bool {
	switch status {
	case http.StatusForbidden, http.StatusUnauthorized, http.StatusNotFound:
		return true
	default:
		return false
	}
}

func (pc *cachingPermChecker) tidy() {
	if len(pc.cache) <= pc.maxCurrent*2 {
		return
	}
	tooOld := time.Now().Add(-minPermCacheAge)
	for uuid, t := range pc.cache {
		if t.Before(tooOld) {
			delete(pc.cache, uuid)
		}
	}
	pc.maxCurrent = len(pc.cache)
}
