// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

var DefaultBlockCache = &BlockCache{}

type BlockCache struct {
	// Maximum number of blocks to keep in the cache. If 0, a
	// default size (currently 4) is used instead.
	MaxBlocks int

	cache map[string]*cacheBlock
	mtx   sync.Mutex
}

const defaultMaxBlocks = 4

// Sweep deletes the least recently used blocks from the cache until
// there are no more than MaxBlocks left.
func (c *BlockCache) Sweep() {
	max := c.MaxBlocks
	if max == 0 {
		max = defaultMaxBlocks
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if len(c.cache) <= max {
		return
	}
	lru := make([]time.Time, 0, len(c.cache))
	for _, b := range c.cache {
		lru = append(lru, b.lastUse)
	}
	sort.Sort(sort.Reverse(timeSlice(lru)))
	threshold := lru[max]
	for loc, b := range c.cache {
		if !b.lastUse.After(threshold) {
			delete(c.cache, loc)
		}
	}
}

// ReadAt returns data from the cache, first retrieving it from Keep if
// necessary.
func (c *BlockCache) ReadAt(upstream arvados.KeepGateway, locator string, p []byte, off int) (int, error) {
	buf, err := c.get(upstream, locator)
	if err != nil {
		return 0, err
	}
	if off > len(buf) {
		return 0, io.ErrUnexpectedEOF
	}
	return copy(p, buf[off:]), nil
}

// Get a block from the cache, first retrieving it from Keep if
// necessary.
func (c *BlockCache) get(upstream arvados.KeepGateway, locator string) ([]byte, error) {
	cacheKey := locator[:32]
	bufsize := BLOCKSIZE
	if parts := strings.SplitN(locator, "+", 3); len(parts) >= 2 {
		datasize, err := strconv.ParseInt(parts[1], 10, 32)
		if err == nil && datasize >= 0 {
			bufsize = int(datasize)
		}
	}
	c.mtx.Lock()
	if c.cache == nil {
		c.cache = make(map[string]*cacheBlock)
	}
	b, ok := c.cache[cacheKey]
	if !ok || b.err != nil {
		b = &cacheBlock{
			fetched: make(chan struct{}),
			lastUse: time.Now(),
		}
		c.cache[cacheKey] = b
		go func() {
			buf := bytes.NewBuffer(make([]byte, 0, bufsize))
			_, err := upstream.BlockRead(context.Background(), arvados.BlockReadOptions{Locator: locator, WriteTo: buf})
			c.mtx.Lock()
			b.data, b.err = buf.Bytes(), err
			c.mtx.Unlock()
			close(b.fetched)
			go c.Sweep()
		}()
	}
	c.mtx.Unlock()

	// Wait (with mtx unlocked) for the fetch goroutine to finish,
	// in case it hasn't already.
	<-b.fetched

	c.mtx.Lock()
	b.lastUse = time.Now()
	c.mtx.Unlock()
	return b.data, b.err
}

func (c *BlockCache) Clear() {
	c.mtx.Lock()
	c.cache = nil
	c.mtx.Unlock()
}

type timeSlice []time.Time

func (ts timeSlice) Len() int { return len(ts) }

func (ts timeSlice) Less(i, j int) bool { return ts[i].Before(ts[j]) }

func (ts timeSlice) Swap(i, j int) { ts[i], ts[j] = ts[j], ts[i] }

type cacheBlock struct {
	data    []byte
	err     error
	fetched chan struct{}
	lastUse time.Time
}
