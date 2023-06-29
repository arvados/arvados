// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	lru "github.com/hashicorp/golang-lru"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const metricsUpdateInterval = time.Second / 10

type cache struct {
	cluster   *arvados.Cluster
	logger    logrus.FieldLogger
	registry  *prometheus.Registry
	metrics   cacheMetrics
	sessions  *lru.TwoQueueCache
	setupOnce sync.Once
	mtx       sync.Mutex

	chPruneSessions chan struct{}
}

type cacheMetrics struct {
	requests        prometheus.Counter
	collectionBytes prometheus.Gauge
	sessionEntries  prometheus.Gauge
	sessionHits     prometheus.Counter
	sessionMisses   prometheus.Counter
}

func (m *cacheMetrics) setup(reg *prometheus.Registry) {
	m.collectionBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_sessions",
		Name:      "cached_session_bytes",
		Help:      "Total size of all cached sessions.",
	})
	reg.MustRegister(m.collectionBytes)
	m.sessionEntries = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_sessions",
		Name:      "active",
		Help:      "Number of active token sessions.",
	})
	reg.MustRegister(m.sessionEntries)
	m.sessionHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_sessions",
		Name:      "hits",
		Help:      "Number of token session cache hits.",
	})
	reg.MustRegister(m.sessionHits)
	m.sessionMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_sessions",
		Name:      "misses",
		Help:      "Number of token session cache misses.",
	})
	reg.MustRegister(m.sessionMisses)
}

type cachedSession struct {
	cache         *cache
	expire        time.Time
	client        *arvados.Client
	arvadosclient *arvadosclient.ArvadosClient
	keepclient    *keepclient.KeepClient

	// Each session uses a system of three mutexes (plus the
	// cache-wide mutex) to enable the following semantics:
	//
	// - There are never multiple sessions in use for a given
	// token.
	//
	// - If the cached in-memory filesystems/user records are
	// older than the configured cache TTL when a request starts,
	// the request will use new ones.
	//
	// - Unused sessions are garbage-collected.
	//
	// In particular, when it is necessary to reset a session's
	// filesystem/user record (to save memory or respect the
	// configured cache TTL), any operations that are already
	// using the existing filesystem/user record are allowed to
	// finish before the new filesystem is constructed.
	//
	// The locks must be acquired in the following order:
	// cache.mtx, session.mtx, session.refresh, session.inuse.

	// mtx is RLocked while session is not safe to evict from
	// cache -- i.e., a checkout() has decided to use it, and its
	// caller is not finished with it. When locking or rlocking
	// this mtx, the cache mtx MUST already be held.
	//
	// This mutex enables pruneSessions to detect when it is safe
	// to completely remove the session entry from the cache.
	mtx sync.RWMutex
	// refresh must be locked in order to read or write the
	// fs/user/userLoaded fields. This mutex enables GetSession
	// and pruneSessions to remove/replace fs and user values
	// safely.
	refresh sync.Mutex
	// inuse must be RLocked while the session is in use by a
	// caller. This mutex enables pruneSessions() to wait for all
	// existing usage to finish by calling inuse.Lock().
	inuse sync.RWMutex

	fs         arvados.CustomFileSystem
	user       arvados.User
	userLoaded bool
}

func (sess *cachedSession) Release() {
	sess.inuse.RUnlock()
	sess.mtx.RUnlock()
	select {
	case sess.cache.chPruneSessions <- struct{}{}:
	default:
	}
}

func (c *cache) setup() {
	var err error
	c.sessions, err = lru.New2Q(c.cluster.Collections.WebDAVCache.MaxSessions)
	if err != nil {
		panic(err)
	}

	reg := c.registry
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	c.metrics.setup(reg)
	go func() {
		for range time.Tick(metricsUpdateInterval) {
			c.updateGauges()
		}
	}()
	c.chPruneSessions = make(chan struct{}, 1)
	go func() {
		for range c.chPruneSessions {
			c.pruneSessions()
		}
	}()
}

func (c *cache) updateGauges() {
	c.metrics.collectionBytes.Set(float64(c.collectionBytes()))
	c.metrics.sessionEntries.Set(float64(c.sessions.Len()))
}

var selectPDH = map[string]interface{}{
	"select": []string{"portable_data_hash"},
}

func (c *cache) checkout(token string) (*cachedSession, error) {
	c.setupOnce.Do(c.setup)
	c.mtx.Lock()
	defer c.mtx.Unlock()
	ent, _ := c.sessions.Get(token)
	sess, _ := ent.(*cachedSession)
	if sess == nil {
		client, err := arvados.NewClientFromConfig(c.cluster)
		if err != nil {
			return nil, err
		}
		client.AuthToken = token
		client.Timeout = time.Minute
		// A non-empty origin header tells controller to
		// prioritize our traffic as interactive, which is
		// true most of the time.
		origin := c.cluster.Services.WebDAVDownload.ExternalURL
		client.SendHeader = http.Header{"Origin": {origin.Scheme + "://" + origin.Host}}
		arvadosclient, err := arvadosclient.New(client)
		if err != nil {
			return nil, err
		}
		sess = &cachedSession{
			cache:         c,
			client:        client,
			arvadosclient: arvadosclient,
			keepclient:    keepclient.New(arvadosclient),
		}
		c.sessions.Add(token, sess)
	}
	sess.mtx.RLock()
	return sess, nil
}

// Get a long-lived CustomFileSystem suitable for doing a read or
// write operation with the given token.
//
// If the returned error is nil, the caller must call Release() on the
// returned session when finished using it.
func (c *cache) GetSession(token string) (arvados.CustomFileSystem, *cachedSession, *arvados.User, error) {
	sess, err := c.checkout(token)
	if err != nil {
		return nil, nil, nil, err
	}
	sess.refresh.Lock()
	defer sess.refresh.Unlock()
	now := time.Now()
	refresh := sess.expire.Before(now)
	if sess.fs == nil || !sess.userLoaded || refresh {
		// Wait for all active users to finish (otherwise they
		// might make changes to an old fs after we start
		// using the new fs).
		sess.inuse.Lock()
		if !sess.userLoaded || refresh {
			err := sess.client.RequestAndDecode(&sess.user, "GET", "/arvados/v1/users/current", nil, nil)
			if he := errorWithHTTPStatus(nil); errors.As(err, &he) && he.HTTPStatus() == http.StatusForbidden {
				// token is OK, but "get user id" api is out
				// of scope -- use existing/expired info if
				// any, or leave empty for unknown user
			} else if err != nil {
				sess.inuse.Unlock()
				sess.mtx.RUnlock()
				return nil, nil, nil, err
			}
			sess.userLoaded = true
		}

		if sess.fs == nil || refresh {
			sess.fs = sess.client.SiteFileSystem(sess.keepclient)
			sess.fs.ForwardSlashNameSubstitution(c.cluster.Collections.ForwardSlashNameSubstitution)
			sess.expire = now.Add(c.cluster.Collections.WebDAVCache.TTL.Duration())
			c.metrics.sessionMisses.Inc()
		} else {
			c.metrics.sessionHits.Inc()
		}
		sess.inuse.Unlock()
	} else {
		c.metrics.sessionHits.Inc()
	}
	sess.inuse.RLock()
	return sess.fs, sess, &sess.user, nil
}

// Remove all expired idle session cache entries, and remove in-memory
// filesystems until approximate remaining size <= maxsize/2
func (c *cache) pruneSessions() {
	now := time.Now()
	keys := c.sessions.Keys()
	sizes := make([]int64, len(keys))
	prune := []string(nil)
	var size int64
	for i, token := range keys {
		token := token.(string)
		ent, ok := c.sessions.Peek(token)
		if !ok {
			continue
		}
		sess := ent.(*cachedSession)
		sess.refresh.Lock()
		expired := sess.expire.Before(now)
		fs := sess.fs
		sess.refresh.Unlock()
		if expired {
			prune = append(prune, token)
		}
		if fs != nil {
			sizes[i] = fs.MemorySize()
			size += sizes[i]
		}
	}
	// Remove tokens until reaching size limit, starting with the
	// least frequently used entries (which Keys() returns last).
	for i := len(keys) - 1; i >= 0 && size > c.cluster.Collections.WebDAVCache.MaxCollectionBytes; i-- {
		if sizes[i] > 0 {
			prune = append(prune, keys[i].(string))
			size -= sizes[i]
		}
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()
	for _, token := range prune {
		ent, ok := c.sessions.Peek(token)
		if !ok {
			continue
		}
		sess := ent.(*cachedSession)
		if sess.mtx.TryLock() {
			c.sessions.Remove(token)
			continue
		}
		// We can't remove a session that's been checked out
		// -- that would allow another session to be created
		// for the same token using a different in-memory
		// filesystem. Instead, we wait for active requests to
		// finish and then "unload" it. After this, either the
		// next GetSession will reload fs/user, or a
		// subsequent pruneSessions will remove the session.
		go func() {
			// Ensure nobody is mid-GetSession() (note we
			// already know nobody is mid-checkout()
			// because we have c.mtx locked)
			sess.refresh.Lock()
			defer sess.refresh.Unlock()
			// Wait for current usage to finish (i.e.,
			// anyone who has decided to use the current
			// values of sess.fs and sess.user, and hasn't
			// called Release() yet)
			sess.inuse.Lock()
			defer sess.inuse.Unlock()
			// Release memory
			sess.fs = nil
			if sess.expire.Before(now) {
				// Mark user data as stale
				sess.userLoaded = false
			}
			// Next GetSession will make a new fs
		}()
	}
}

// collectionBytes returns the approximate combined memory size of the
// collection cache and session filesystem cache.
func (c *cache) collectionBytes() uint64 {
	var size uint64
	for _, token := range c.sessions.Keys() {
		ent, ok := c.sessions.Peek(token)
		if !ok {
			continue
		}
		sess := ent.(*cachedSession)
		sess.refresh.Lock()
		fs := sess.fs
		sess.refresh.Unlock()
		if fs != nil {
			size += uint64(fs.MemorySize())
		}
	}
	return size
}
