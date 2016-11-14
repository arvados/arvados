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
		cache:      make(map[string]time.Time),
		maxCurrent: 16,
	}
}

type cachingPermChecker struct {
	*arvados.Client
	cache      map[string]time.Time
	maxCurrent int
}

func (pc *cachingPermChecker) SetToken(token string) {
	pc.Client.AuthToken = token
}

func (pc *cachingPermChecker) Check(uuid string) (bool, error) {
	pc.tidy()
	if t, ok := pc.cache[uuid]; ok && time.Now().Sub(t) < maxPermCacheAge {
		debugLogf("perm ok (cached): %+q %+q", pc.Client.AuthToken, uuid)
		return true, nil
	}
	var buf map[string]interface{}
	path, err := pc.PathForUUID("get", uuid)
	if err != nil {
		return false, err
	}
	err = pc.RequestAndDecode(&buf, "GET", path, nil, url.Values{
		"select": {`["uuid"]`},
	})
	if err, ok := err.(arvados.TransactionError); ok && err.StatusCode == http.StatusNotFound {
		debugLogf("perm err: %+q %+q: %s", pc.Client.AuthToken, uuid, err)
		return false, nil
	}
	if err != nil {
		debugLogf("perm !ok: %+q %+q", pc.Client.AuthToken, uuid)
		return false, err
	}
	debugLogf("perm ok: %+q %+q", pc.Client.AuthToken, uuid)
	pc.cache[uuid] = time.Now()
	return true, nil
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
