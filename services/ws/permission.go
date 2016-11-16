package main

import (
	"net/http"
	"net/url"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

const (
	maxPermCacheAge = time.Hour
	minPermCacheAge = 5 * time.Minute
)

type permChecker interface {
	SetToken(token string)
	Check(uuid string) (bool, error)
}

func NewPermChecker(ac arvados.Client) permChecker {
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
}

func (pc *cachingPermChecker) SetToken(token string) {
	pc.Client.AuthToken = token
}

func (pc *cachingPermChecker) Check(uuid string) (bool, error) {
	pc.tidy()
	now := time.Now()
	if perm, ok := pc.cache[uuid]; ok && now.Sub(perm.Time) < maxPermCacheAge {
		debugLogf("perm (cached): %+q %+q => %s", pc.Client.AuthToken, uuid, perm.allowed)
		return perm.allowed, nil
	}
	var buf map[string]interface{}
	path, err := pc.PathForUUID("get", uuid)
	if err != nil {
		return false, err
	}
	err = pc.RequestAndDecode(&buf, "GET", path, nil, url.Values{
		"select": {`["uuid"]`},
	})

	var allowed bool
	if err == nil {
		allowed = true
	} else if txErr, ok := err.(arvados.TransactionError); ok && txErr.StatusCode == http.StatusNotFound {
		allowed = false
	} else {
		errorLogf("perm err: %+q %+q: %s", pc.Client.AuthToken, uuid, err)
		return false, err
	}
	debugLogf("perm: %+q %+q => %s", pc.Client.AuthToken, uuid, allowed)
	pc.cache[uuid] = cacheEnt{Time: now, allowed: allowed}
	return allowed, nil
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
