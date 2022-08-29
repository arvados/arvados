// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
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
	cluster     *arvados.Cluster
	logger      logrus.FieldLogger
	registry    *prometheus.Registry
	metrics     cacheMetrics
	pdhs        *lru.TwoQueueCache
	collections *lru.TwoQueueCache
	sessions    *lru.TwoQueueCache
	setupOnce   sync.Once

	chPruneSessions    chan struct{}
	chPruneCollections chan struct{}
}

type cacheMetrics struct {
	requests          prometheus.Counter
	collectionBytes   prometheus.Gauge
	collectionEntries prometheus.Gauge
	sessionEntries    prometheus.Gauge
	collectionHits    prometheus.Counter
	pdhHits           prometheus.Counter
	sessionHits       prometheus.Counter
	sessionMisses     prometheus.Counter
	apiCalls          prometheus.Counter
}

func (m *cacheMetrics) setup(reg *prometheus.Registry) {
	m.requests = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "requests",
		Help:      "Number of targetID-to-manifest lookups handled.",
	})
	reg.MustRegister(m.requests)
	m.collectionHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "hits",
		Help:      "Number of pdh-to-manifest cache hits.",
	})
	reg.MustRegister(m.collectionHits)
	m.pdhHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "pdh_hits",
		Help:      "Number of uuid-to-pdh cache hits.",
	})
	reg.MustRegister(m.pdhHits)
	m.apiCalls = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "api_calls",
		Help:      "Number of outgoing API calls made by cache.",
	})
	reg.MustRegister(m.apiCalls)
	m.collectionBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_sessions",
		Name:      "cached_collection_bytes",
		Help:      "Total size of all cached manifests and sessions.",
	})
	reg.MustRegister(m.collectionBytes)
	m.collectionEntries = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "keepweb_collectioncache",
		Name:      "cached_manifests",
		Help:      "Number of manifests in cache.",
	})
	reg.MustRegister(m.collectionEntries)
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

type cachedPDH struct {
	expire  time.Time
	refresh time.Time
	pdh     string
}

type cachedCollection struct {
	expire     time.Time
	collection *arvados.Collection
}

type cachedPermission struct {
	expire time.Time
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
	c.pdhs, err = lru.New2Q(c.cluster.Collections.WebDAVCache.MaxUUIDEntries)
	if err != nil {
		panic(err)
	}
	c.collections, err = lru.New2Q(c.cluster.Collections.WebDAVCache.MaxCollectionEntries)
	if err != nil {
		panic(err)
	}
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
	c.chPruneCollections = make(chan struct{}, 1)
	go func() {
		for range c.chPruneCollections {
			c.pruneCollections()
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
	c.metrics.collectionEntries.Set(float64(c.collections.Len()))
	c.metrics.sessionEntries.Set(float64(c.sessions.Len()))
}

var selectPDH = map[string]interface{}{
	"select": []string{"portable_data_hash"},
}

// Update saves a modified version (fs) to an existing collection
// (coll) and, if successful, updates the relevant cache entries so
// subsequent calls to Get() reflect the modifications.
func (c *cache) Update(client *arvados.Client, coll arvados.Collection, fs arvados.CollectionFileSystem) error {
	c.setupOnce.Do(c.setup)

	m, err := fs.MarshalManifest(".")
	if err != nil || m == coll.ManifestText {
		return err
	}
	coll.ManifestText = m
	var updated arvados.Collection
	err = client.RequestAndDecode(&updated, "PATCH", "arvados/v1/collections/"+coll.UUID, nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": coll.ManifestText,
		},
	})
	if err != nil {
		c.pdhs.Remove(coll.UUID)
		return err
	}
	c.collections.Add(client.AuthToken+"\000"+updated.PortableDataHash, &cachedCollection{
		expire:     time.Now().Add(time.Duration(c.cluster.Collections.WebDAVCache.TTL)),
		collection: &updated,
	})
	c.pdhs.Add(coll.UUID, &cachedPDH{
		expire:  time.Now().Add(time.Duration(c.cluster.Collections.WebDAVCache.TTL)),
		refresh: time.Now().Add(time.Duration(c.cluster.Collections.WebDAVCache.UUIDTTL)),
		pdh:     updated.PortableDataHash,
	})
	return nil
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
func (c *cache) GetSession(token string) (arvados.CustomFileSystem, *cachedSession, error) {
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
			return nil, nil, err
		}
		sess.client.AuthToken = token
		sess.arvadosclient, err = arvadosclient.New(sess.client)
		if err != nil {
			return nil, nil, err
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
	if fs != nil && !expired {
		return fs, sess, nil
	}
	fs = sess.client.SiteFileSystem(sess.keepclient)
	fs.ForwardSlashNameSubstitution(c.cluster.Collections.ForwardSlashNameSubstitution)
	sess.fs.Store(fs)
	return fs, sess, nil
}

// Remove all expired session cache entries, then remove more entries
// until approximate remaining size <= maxsize/2
func (c *cache) pruneSessions() {
	now := time.Now()
	var size int64
	keys := c.sessions.Keys()
	for idx, token := range keys {
		ent, ok := c.sessions.Peek(token)
		if !ok {
			continue
		}
		s := ent.(*cachedSession)
		if s.expire.Before(now) {
			c.sessions.Remove(token)
			keys[idx] = ""
			continue
		}
		if fs, ok := s.fs.Load().(arvados.CustomFileSystem); ok {
			size += fs.MemorySize()
		}
	}
	// Remove tokens until reaching size limit, starting with the
	// least frequently used entries (which Keys() returns last).
	for i := len(keys) - 1; i >= 0; i-- {
		token := keys[i]
		if token == "" {
			// removed this session in the loop above;
			// don't prune it, even if it's already been
			// reinserted.
			continue
		}
		if size <= c.cluster.Collections.WebDAVCache.MaxCollectionBytes/2 {
			break
		}
		ent, ok := c.sessions.Peek(token)
		if !ok {
			continue
		}
		s := ent.(*cachedSession)
		fs, _ := s.fs.Load().(arvados.CustomFileSystem)
		if fs == nil {
			continue
		}
		c.sessions.Remove(token)
		size -= fs.MemorySize()
	}
}

func (c *cache) Get(arv *arvadosclient.ArvadosClient, targetID string, forceReload bool) (*arvados.Collection, error) {
	c.setupOnce.Do(c.setup)
	c.metrics.requests.Inc()

	var pdhRefresh bool
	var pdh string
	if arvadosclient.PDHMatch(targetID) {
		pdh = targetID
	} else if ent, cached := c.pdhs.Get(targetID); cached {
		ent := ent.(*cachedPDH)
		if ent.expire.Before(time.Now()) {
			c.pdhs.Remove(targetID)
		} else {
			pdh = ent.pdh
			pdhRefresh = forceReload || time.Now().After(ent.refresh)
			c.metrics.pdhHits.Inc()
		}
	}

	if pdh == "" {
		// UUID->PDH mapping is not cached, might as well get
		// the whole collection record and be done (below).
		c.logger.Debugf("cache(%s): have no pdh", targetID)
	} else if cached := c.lookupCollection(arv.ApiToken + "\000" + pdh); cached == nil {
		// PDH->manifest is not cached, might as well get the
		// whole collection record (below).
		c.logger.Debugf("cache(%s): have pdh %s but manifest is not cached", targetID, pdh)
	} else if !pdhRefresh {
		// We looked up UUID->PDH very recently, and we still
		// have the manifest for that PDH.
		c.logger.Debugf("cache(%s): have pdh %s and refresh not needed", targetID, pdh)
		return cached, nil
	} else {
		// Get current PDH for this UUID (and confirm we still
		// have read permission).  Most likely, the cached PDH
		// is still correct, in which case we can use our
		// cached manifest.
		c.metrics.apiCalls.Inc()
		var current arvados.Collection
		err := arv.Get("collections", targetID, selectPDH, &current)
		if err != nil {
			return nil, err
		}
		if current.PortableDataHash == pdh {
			// PDH has not changed, cached manifest is
			// correct.
			c.logger.Debugf("cache(%s): verified cached pdh %s is still correct", targetID, pdh)
			return cached, nil
		}
		if cached := c.lookupCollection(arv.ApiToken + "\000" + current.PortableDataHash); cached != nil {
			// PDH changed, and we already have the
			// manifest for that new PDH.
			c.logger.Debugf("cache(%s): cached pdh %s was stale, new pdh is %s and manifest is already in cache", targetID, pdh, current.PortableDataHash)
			return cached, nil
		}
	}

	// Either UUID->PDH is not cached, or PDH->manifest is not
	// cached.
	var retrieved arvados.Collection
	c.metrics.apiCalls.Inc()
	err := arv.Get("collections", targetID, nil, &retrieved)
	if err != nil {
		return nil, err
	}
	c.logger.Debugf("cache(%s): retrieved manifest, caching with pdh %s", targetID, retrieved.PortableDataHash)
	exp := time.Now().Add(time.Duration(c.cluster.Collections.WebDAVCache.TTL))
	if targetID != retrieved.PortableDataHash {
		c.pdhs.Add(targetID, &cachedPDH{
			expire:  exp,
			refresh: time.Now().Add(time.Duration(c.cluster.Collections.WebDAVCache.UUIDTTL)),
			pdh:     retrieved.PortableDataHash,
		})
	}
	c.collections.Add(arv.ApiToken+"\000"+retrieved.PortableDataHash, &cachedCollection{
		expire:     exp,
		collection: &retrieved,
	})
	if int64(len(retrieved.ManifestText)) > c.cluster.Collections.WebDAVCache.MaxCollectionBytes/int64(c.cluster.Collections.WebDAVCache.MaxCollectionEntries) {
		select {
		case c.chPruneCollections <- struct{}{}:
		default:
		}
	}
	return &retrieved, nil
}

// pruneCollections checks the total bytes occupied by manifest_text
// in the collection cache and removes old entries as needed to bring
// the total size down to CollectionBytes. It also deletes all expired
// entries.
//
// pruneCollections does not aim to be perfectly correct when there is
// concurrent cache activity.
func (c *cache) pruneCollections() {
	var size int64
	now := time.Now()
	keys := c.collections.Keys()
	entsize := make([]int, len(keys))
	expired := make([]bool, len(keys))
	for i, k := range keys {
		v, ok := c.collections.Peek(k)
		if !ok {
			continue
		}
		ent := v.(*cachedCollection)
		n := len(ent.collection.ManifestText)
		size += int64(n)
		entsize[i] = n
		expired[i] = ent.expire.Before(now)
	}
	for i, k := range keys {
		if expired[i] {
			c.collections.Remove(k)
			size -= int64(entsize[i])
		}
	}
	for i, k := range keys {
		if size <= c.cluster.Collections.WebDAVCache.MaxCollectionBytes/2 {
			break
		}
		if expired[i] {
			// already removed this entry in the previous loop
			continue
		}
		c.collections.Remove(k)
		size -= int64(entsize[i])
	}
}

// collectionBytes returns the approximate combined memory size of the
// collection cache and session filesystem cache.
func (c *cache) collectionBytes() uint64 {
	var size uint64
	for _, k := range c.collections.Keys() {
		v, ok := c.collections.Peek(k)
		if !ok {
			continue
		}
		size += uint64(len(v.(*cachedCollection).collection.ManifestText))
	}
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

func (c *cache) lookupCollection(key string) *arvados.Collection {
	e, cached := c.collections.Get(key)
	if !cached {
		return nil
	}
	ent := e.(*cachedCollection)
	if ent.expire.Before(time.Now()) {
		c.collections.Remove(key)
		return nil
	}
	c.metrics.collectionHits.Inc()
	return ent.collection
}

func (c *cache) GetTokenUser(token string) (*arvados.User, error) {
	// Get and cache user record associated with this
	// token.  We need to know their UUID for logging, and
	// whether they are an admin or not for certain
	// permission checks.

	// Get/create session entry
	_, sess, err := c.GetSession(token)
	if err != nil {
		return nil, err
	}

	// See if the user is already set, and if so, return it
	user, _ := sess.user.Load().(*arvados.User)
	if user != nil {
		return user, nil
	}

	// Fetch the user record
	c.metrics.apiCalls.Inc()
	var current arvados.User

	err = sess.client.RequestAndDecode(&current, "GET", "arvados/v1/users/current", nil, nil)
	if err != nil {
		return nil, err
	}

	// Stash the user record for next time
	sess.user.Store(&current)
	return &current, nil
}
