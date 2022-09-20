// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"net/http"
	"sync"
	"sync/atomic"
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
	expire        time.Time
	fs            atomic.Value
	client        *arvados.Client
	arvadosclient *arvadosclient.ArvadosClient
	keepclient    *keepclient.KeepClient
	user          atomic.Value
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

// ResetSession unloads any potentially stale state. Should be called
// after write operations, so subsequent reads don't return stale
// data.
func (c *cache) ResetSession(token string) {
	c.setupOnce.Do(c.setup)
	c.sessions.Remove(token)
}

// Get a long-lived CustomFileSystem suitable for doing a read operation
// with the given token.
func (c *cache) GetSession(token string) (arvados.CustomFileSystem, *cachedSession, *arvados.User, error) {
	c.setupOnce.Do(c.setup)
	now := time.Now()
	ent, _ := c.sessions.Get(token)
	sess, _ := ent.(*cachedSession)
	expired := false
	if sess == nil {
		c.metrics.sessionMisses.Inc()
		sess = &cachedSession{
			expire: now.Add(c.cluster.Collections.WebDAVCache.TTL.Duration()),
		}
		var err error
		sess.client, err = arvados.NewClientFromConfig(c.cluster)
		if err != nil {
			return nil, nil, nil, err
		}
		sess.client.AuthToken = token
		sess.arvadosclient, err = arvadosclient.New(sess.client)
		if err != nil {
			return nil, nil, nil, err
		}
		sess.keepclient = keepclient.New(sess.arvadosclient)
		c.sessions.Add(token, sess)
	} else if sess.expire.Before(now) {
		c.metrics.sessionMisses.Inc()
		expired = true
	} else {
		c.metrics.sessionHits.Inc()
	}
	select {
	case c.chPruneSessions <- struct{}{}:
	default:
	}

	fs, _ := sess.fs.Load().(arvados.CustomFileSystem)
	if fs == nil || expired {
		fs = sess.client.SiteFileSystem(sess.keepclient)
		fs.ForwardSlashNameSubstitution(c.cluster.Collections.ForwardSlashNameSubstitution)
		sess.fs.Store(fs)
	}

	user, _ := sess.user.Load().(*arvados.User)
	if user == nil || expired {
		user = new(arvados.User)
		err := sess.client.RequestAndDecode(user, "GET", "/arvados/v1/users/current", nil, nil)
		if statusErr, ok := err.(interface{ HTTPStatus() int }); ok && statusErr.HTTPStatus() == http.StatusForbidden {
			// token is OK, but "get user id" api is out
			// of scope -- return nil, signifying unknown
			// user
		} else if err != nil {
			return nil, nil, nil, err
		}
		sess.user.Store(user)
	}

	return fs, sess, user, nil
}

// Remove all expired session cache entries, then remove more entries
// until approximate remaining size <= maxsize/2
func (c *cache) pruneSessions() {
	now := time.Now()
	keys := c.sessions.Keys()
	sizes := make([]int64, len(keys))
	var size int64
	for i, token := range keys {
		ent, ok := c.sessions.Peek(token)
		if !ok {
			continue
		}
		s := ent.(*cachedSession)
		if s.expire.Before(now) {
			c.sessions.Remove(token)
			continue
		}
		if fs, ok := s.fs.Load().(arvados.CustomFileSystem); ok {
			sizes[i] = fs.MemorySize()
			size += sizes[i]
		}
	}
	// Remove tokens until reaching size limit, starting with the
	// least frequently used entries (which Keys() returns last).
	for i := len(keys) - 1; i >= 0 && size > c.cluster.Collections.WebDAVCache.MaxCollectionBytes; i-- {
		if sizes[i] > 0 {
			c.sessions.Remove(keys[i])
			size -= sizes[i]
		}
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
		if fs, ok := ent.(*cachedSession).fs.Load().(arvados.CustomFileSystem); ok {
			size += uint64(fs.MemorySize())
		}
	}
	return size
}
