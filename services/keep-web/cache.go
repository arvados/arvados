package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"github.com/hashicorp/golang-lru"
)

type cache struct {
	TTL                  arvados.Duration
	MaxCollectionEntries int
	MaxCollectionBytes   int64
	MaxPermissionEntries int
	MaxUUIDEntries       int

	stats       cacheStats
	pdhs        *lru.TwoQueueCache
	collections *lru.TwoQueueCache
	permissions *lru.TwoQueueCache
	setupOnce   sync.Once
}

type cacheStats struct {
	Requests          uint64 `json:"Cache.Requests"`
	CollectionBytes   uint64 `json:"Cache.CollectionBytes"`
	CollectionEntries int    `json:"Cache.CollectionEntries"`
	CollectionHits    uint64 `json:"Cache.CollectionHits"`
	PDHHits           uint64 `json:"Cache.UUIDHits"`
	PermissionHits    uint64 `json:"Cache.PermissionHits"`
	APICalls          uint64 `json:"Cache.APICalls"`
}

type cachedPDH struct {
	expire time.Time
	pdh    string
}

type cachedCollection struct {
	expire     time.Time
	collection map[string]interface{}
}

type cachedPermission struct {
	expire time.Time
}

func (c *cache) setup() {
	var err error
	c.pdhs, err = lru.New2Q(c.MaxUUIDEntries)
	if err != nil {
		panic(err)
	}
	c.collections, err = lru.New2Q(c.MaxCollectionEntries)
	if err != nil {
		panic(err)
	}
	c.permissions, err = lru.New2Q(c.MaxPermissionEntries)
	if err != nil {
		panic(err)
	}
}

var selectPDH = map[string]interface{}{
	"select": []string{"portable_data_hash"},
}

func (c *cache) Stats() cacheStats {
	c.setupOnce.Do(c.setup)
	return cacheStats{
		Requests:          atomic.LoadUint64(&c.stats.Requests),
		CollectionBytes:   c.collectionBytes(),
		CollectionEntries: c.collections.Len(),
		CollectionHits:    atomic.LoadUint64(&c.stats.CollectionHits),
		PDHHits:           atomic.LoadUint64(&c.stats.PDHHits),
		PermissionHits:    atomic.LoadUint64(&c.stats.PermissionHits),
		APICalls:          atomic.LoadUint64(&c.stats.APICalls),
	}
}

func (c *cache) Get(arv *arvadosclient.ArvadosClient, targetID string, forceReload bool) (map[string]interface{}, error) {
	c.setupOnce.Do(c.setup)

	atomic.AddUint64(&c.stats.Requests, 1)

	permOK := false
	permKey := arv.ApiToken + "\000" + targetID
	if forceReload {
	} else if ent, cached := c.permissions.Get(permKey); cached {
		ent := ent.(*cachedPermission)
		if ent.expire.Before(time.Now()) {
			c.permissions.Remove(permKey)
		} else {
			permOK = true
			atomic.AddUint64(&c.stats.PermissionHits, 1)
		}
	}

	var pdh string
	if arvadosclient.PDHMatch(targetID) {
		pdh = targetID
	} else if forceReload {
	} else if ent, cached := c.pdhs.Get(targetID); cached {
		ent := ent.(*cachedPDH)
		if ent.expire.Before(time.Now()) {
			c.pdhs.Remove(targetID)
		} else {
			pdh = ent.pdh
			atomic.AddUint64(&c.stats.PDHHits, 1)
		}
	}

	var collection map[string]interface{}
	if pdh != "" {
		collection = c.lookupCollection(pdh)
	}

	if collection != nil && permOK {
		return collection, nil
	} else if collection != nil {
		// Ask API for current PDH for this targetID. Most
		// likely, the cached PDH is still correct; if so,
		// _and_ the current token has permission, we can
		// use our cached manifest.
		atomic.AddUint64(&c.stats.APICalls, 1)
		var current map[string]interface{}
		err := arv.Get("collections", targetID, selectPDH, &current)
		if err != nil {
			return nil, err
		}
		if checkPDH, ok := current["portable_data_hash"].(string); !ok {
			return nil, fmt.Errorf("API response for %q had no PDH", targetID)
		} else if checkPDH == pdh {
			exp := time.Now().Add(time.Duration(c.TTL))
			c.permissions.Add(permKey, &cachedPermission{
				expire: exp,
			})
			if pdh != targetID {
				c.pdhs.Add(targetID, &cachedPDH{
					expire: exp,
					pdh:    pdh,
				})
			}
			return collection, err
		} else {
			// PDH changed, but now we know we have
			// permission -- and maybe we already have the
			// new PDH in the cache.
			if coll := c.lookupCollection(checkPDH); coll != nil {
				return coll, nil
			}
		}
	}

	// Collection manifest is not cached.
	atomic.AddUint64(&c.stats.APICalls, 1)
	err := arv.Get("collections", targetID, nil, &collection)
	if err != nil {
		return nil, err
	}
	pdh, ok := collection["portable_data_hash"].(string)
	if !ok {
		return nil, fmt.Errorf("API response for %q had no PDH", targetID)
	}
	exp := time.Now().Add(time.Duration(c.TTL))
	c.permissions.Add(permKey, &cachedPermission{
		expire: exp,
	})
	c.pdhs.Add(targetID, &cachedPDH{
		expire: exp,
		pdh:    pdh,
	})
	c.collections.Add(pdh, &cachedCollection{
		expire:     exp,
		collection: collection,
	})
	if int64(len(collection["manifest_text"].(string))) > c.MaxCollectionBytes/int64(c.MaxCollectionEntries) {
		go c.pruneCollections()
	}
	return collection, nil
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
		n := len(ent.collection["manifest_text"].(string))
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
		if size <= c.MaxCollectionBytes {
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

// collectionBytes returns the approximate memory size of the
// collection cache.
func (c *cache) collectionBytes() uint64 {
	var size uint64
	for _, k := range c.collections.Keys() {
		v, ok := c.collections.Peek(k)
		if !ok {
			continue
		}
		size += uint64(len(v.(*cachedCollection).collection["manifest_text"].(string)))
	}
	return size
}

func (c *cache) lookupCollection(pdh string) map[string]interface{} {
	if pdh == "" {
		return nil
	} else if ent, cached := c.collections.Get(pdh); !cached {
		return nil
	} else {
		ent := ent.(*cachedCollection)
		if ent.expire.Before(time.Now()) {
			c.collections.Remove(pdh)
			return nil
		} else {
			atomic.AddUint64(&c.stats.CollectionHits, 1)
			return ent.collection
		}
	}
}
