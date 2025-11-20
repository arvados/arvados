// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&keepCacheSuite{})

type keepCacheSuite struct {
}

type keepGatewayBlackHole struct {
}

func (*keepGatewayBlackHole) ReadAt(locator string, dst []byte, offset int) (int, error) {
	return 0, errors.New("block not found")
}
func (*keepGatewayBlackHole) BlockRead(ctx context.Context, opts BlockReadOptions) (int, error) {
	return 0, errors.New("block not found")
}
func (*keepGatewayBlackHole) LocalLocator(locator string) (string, error) {
	return locator, nil
}
func (*keepGatewayBlackHole) BlockWrite(ctx context.Context, opts BlockWriteOptions) (BlockWriteResponse, error) {
	h := md5.New()
	var size int64
	if opts.Reader == nil {
		size, _ = io.Copy(h, bytes.NewReader(opts.Data))
	} else {
		size, _ = io.Copy(h, opts.Reader)
	}
	return BlockWriteResponse{Locator: fmt.Sprintf("%x+%d", h.Sum(nil), size), Replicas: 1}, nil
}

type keepGatewayMemoryBacked struct {
	mtx                 sync.RWMutex
	data                map[string][]byte
	pauseBlockReadAfter int
	pauseBlockReadUntil chan error
}

func (k *keepGatewayMemoryBacked) ReadAt(locator string, dst []byte, offset int) (int, error) {
	k.mtx.RLock()
	data := k.data[locator]
	k.mtx.RUnlock()
	if data == nil {
		return 0, errors.New("block not found: " + locator)
	}
	var n int
	if len(data) > offset {
		n = copy(dst, data[offset:])
	}
	if n < len(dst) {
		return n, io.EOF
	}
	return n, nil
}
func (k *keepGatewayMemoryBacked) BlockRead(ctx context.Context, opts BlockReadOptions) (int, error) {
	if opts.CheckCacheOnly {
		return 0, ErrNotCached
	}
	k.mtx.RLock()
	data := k.data[opts.Locator]
	k.mtx.RUnlock()
	if data == nil {
		return 0, errors.New("block not found: " + opts.Locator)
	}
	if k.pauseBlockReadUntil != nil {
		src := bytes.NewReader(data)
		n, err := io.CopyN(opts.WriteTo, src, int64(k.pauseBlockReadAfter))
		if err != nil {
			return int(n), err
		}
		<-k.pauseBlockReadUntil
		n2, err := io.Copy(opts.WriteTo, src)
		return int(n + n2), err
	}
	return opts.WriteTo.Write(data)
}
func (k *keepGatewayMemoryBacked) LocalLocator(locator string) (string, error) {
	return locator, nil
}
func (k *keepGatewayMemoryBacked) BlockWrite(ctx context.Context, opts BlockWriteOptions) (BlockWriteResponse, error) {
	h := md5.New()
	data := bytes.NewBuffer(nil)
	if opts.Reader == nil {
		data.Write(opts.Data)
		h.Write(data.Bytes())
	} else {
		io.Copy(io.MultiWriter(h, data), opts.Reader)
	}
	locator := fmt.Sprintf("%x+%d", h.Sum(nil), data.Len())
	k.mtx.Lock()
	if k.data == nil {
		k.data = map[string][]byte{}
	}
	k.data[locator] = data.Bytes()
	k.mtx.Unlock()
	return BlockWriteResponse{Locator: locator, Replicas: 1}, nil
}

func (s *keepCacheSuite) TestBlockWrite(c *check.C) {
	backend := &keepGatewayMemoryBacked{}
	cache := DiskCache{
		KeepGateway: backend,
		MaxSize:     40000000,
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
		Metrics:     NewKeepClientMetrics(),
	}
	ctx := context.Background()
	real, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, 100000),
	})
	c.Assert(err, check.IsNil)

	// Write different data but supply the same hash. Should be
	// rejected (even though our fake backend doesn't notice).
	_, err = cache.BlockWrite(ctx, BlockWriteOptions{
		Hash: real.Locator[:32],
		Data: make([]byte, 10),
	})
	c.Check(err, check.ErrorMatches, `block hash .+ did not match provided hash .+`)

	// Ensure the bogus write didn't overwrite (or delete) the
	// real cached data associated with that hash.
	delete(backend.data, real.Locator)
	n, err := cache.ReadAt(real.Locator, make([]byte, 100), 0)
	c.Check(n, check.Equals, 100)
	c.Check(err, check.IsNil)
}

func (s *keepCacheSuite) TestMaxSize(c *check.C) {
	backend := &keepGatewayMemoryBacked{}
	cache := &DiskCache{
		KeepGateway: backend,
		MaxSize:     40000000,
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
		Metrics:     NewKeepClientMetrics(),
	}
	ctx := context.Background()
	resp1, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, 44000000),
	})
	c.Check(err, check.IsNil)

	// Wait for tidy to finish, check that it doesn't delete the
	// only block.
	waitTidy(cache)
	c.Check(atomic.LoadInt64(&cache.sizeMeasured), check.Equals, int64(44000000))

	resp2, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, 32000000),
	})
	c.Check(err, check.IsNil)
	delete(backend.data, resp1.Locator)
	delete(backend.data, resp2.Locator)

	// Wait for tidy to finish, check that it deleted the older
	// block.
	waitTidy(cache)
	c.Check(atomic.LoadInt64(&cache.sizeMeasured), check.Equals, int64(32000000))

	n, err := cache.ReadAt(resp1.Locator, make([]byte, 2), 0)
	c.Check(n, check.Equals, 0)
	c.Check(err, check.ErrorMatches, `block not found: .*\+44000000`)

	n, err = cache.ReadAt(resp2.Locator, make([]byte, 2), 0)
	c.Check(n > 0, check.Equals, true)
	c.Check(err, check.IsNil)
}

func (s *keepCacheSuite) TestConcurrentReadersNoRefresh(c *check.C) {
	s.testConcurrentReaders(c, true, false)
}
func (s *keepCacheSuite) TestConcurrentReadersMangleCache(c *check.C) {
	s.testConcurrentReaders(c, false, true)
}
func (s *keepCacheSuite) testConcurrentReaders(c *check.C, cannotRefresh, mangleCache bool) {
	blksize := 64000000
	backend := &keepGatewayMemoryBacked{}
	cache := DiskCache{
		KeepGateway: backend,
		MaxSize:     ByteSizeOrPercent(blksize),
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
		Metrics:     NewKeepClientMetrics(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, blksize),
	})
	c.Check(err, check.IsNil)
	if cannotRefresh {
		// Delete the block from the backing store, to ensure
		// the cache doesn't rely on re-reading a block that
		// it has just written.
		delete(backend.data, resp.Locator)
	}
	if mangleCache {
		// Replace cache files with truncated files (and
		// delete them outright) while the ReadAt loop is
		// running, to ensure the cache can re-fetch from the
		// backend as needed.
		var nRemove, nTrunc int
		defer func() {
			c.Logf("nRemove %d", nRemove)
			c.Logf("nTrunc %d", nTrunc)
		}()
		go func() {
			// Truncate/delete the cache file at various
			// intervals. Readers should re-fetch/recover from
			// this.
			fnm := cache.cacheFile(resp.Locator)
			for ctx.Err() == nil {
				trunclen := rand.Int63() % int64(blksize*2)
				if trunclen > int64(blksize) {
					err := os.Remove(fnm)
					if err == nil {
						nRemove++
					}
				} else if os.WriteFile(fnm+"#", make([]byte, trunclen), 0700) == nil {
					err := os.Rename(fnm+"#", fnm)
					if err == nil {
						nTrunc++
					}
				}
			}
		}()
	}

	failed := false
	var wg sync.WaitGroup
	var slots = make(chan bool, 100) // limit concurrency / memory usage
	for i := 0; i < 20000; i++ {
		offset := (i * 123456) % blksize
		slots <- true
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			buf := make([]byte, 654321)
			if offset+len(buf) > blksize {
				buf = buf[:blksize-offset]
			}
			n, err := cache.ReadAt(resp.Locator, buf, offset)
			if failed {
				// don't fill logs with subsequent errors
				return
			}
			if !c.Check(err, check.IsNil, check.Commentf("offset=%d", offset)) {
				failed = true
			}
			c.Assert(n, check.Equals, len(buf))
		}()
	}
	wg.Wait()
}

func (s *keepCacheSuite) TestBlockRead_CheckCacheOnly(c *check.C) {
	blkCached := make([]byte, 12_000_000)
	blkUncached := make([]byte, 13_000_000)
	backend := &keepGatewayMemoryBacked{}
	cache := DiskCache{
		KeepGateway: backend,
		MaxSize:     ByteSizeOrPercent(len(blkUncached) + len(blkCached)),
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
		Metrics:     NewKeepClientMetrics(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: blkUncached,
	})
	c.Check(err, check.IsNil)
	locUncached := resp.Locator

	resp, err = cache.BlockWrite(ctx, BlockWriteOptions{
		Data: blkCached,
	})
	c.Check(err, check.IsNil)
	locCached := resp.Locator

	os.RemoveAll(filepath.Join(cache.Dir, locUncached[:3]))
	cache.deleteHeldopen(cache.cacheFile(locUncached), nil)
	backend.data = make(map[string][]byte)

	// Do multiple concurrent reads so we have a chance of catching
	// race/locking bugs.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			n, err := cache.BlockRead(ctx, BlockReadOptions{
				Locator:        locUncached,
				WriteTo:        &buf,
				CheckCacheOnly: true})
			c.Check(n, check.Equals, 0)
			c.Check(err, check.Equals, ErrNotCached)
			c.Check(buf.Len(), check.Equals, 0)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			n, err := cache.BlockRead(ctx, BlockReadOptions{
				Locator:        locCached,
				WriteTo:        &buf,
				CheckCacheOnly: true})
			c.Check(n, check.Equals, 0)
			c.Check(err, check.IsNil)
			c.Check(buf.Len(), check.Equals, 0)
		}()
	}
	wg.Wait()
}

func (s *keepCacheSuite) TestStreaming(c *check.C) {
	blksize := 64000000
	backend := &keepGatewayMemoryBacked{
		pauseBlockReadUntil: make(chan error),
		pauseBlockReadAfter: blksize / 8,
	}
	cache := DiskCache{
		KeepGateway: backend,
		MaxSize:     ByteSizeOrPercent(blksize),
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
		Metrics:     NewKeepClientMetrics(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, blksize),
	})
	c.Check(err, check.IsNil)
	os.RemoveAll(filepath.Join(cache.Dir, resp.Locator[:3]))

	// Start a lot of concurrent requests for various ranges of
	// the same block. Our backend will return the first 8MB and
	// then pause. The requests that can be satisfied by the first
	// 8MB of data should return quickly. The rest should wait,
	// and return after we release pauseBlockReadUntil.
	var wgEarly, wgLate sync.WaitGroup
	var doneEarly, doneLate int32
	for i := 0; i < 10000; i++ {
		wgEarly.Add(1)
		go func() {
			offset := int(rand.Int63() % int64(blksize-benchReadSize))
			if offset+benchReadSize > backend.pauseBlockReadAfter {
				wgLate.Add(1)
				defer wgLate.Done()
				wgEarly.Done()
				defer atomic.AddInt32(&doneLate, 1)
			} else {
				defer wgEarly.Done()
				defer atomic.AddInt32(&doneEarly, 1)
			}
			buf := make([]byte, benchReadSize)
			n, err := cache.ReadAt(resp.Locator, buf, offset)
			c.Check(n, check.Equals, len(buf))
			c.Check(err, check.IsNil)
		}()
	}

	// Ensure all early ranges finish while backend request(s) are
	// paused.
	wgEarly.Wait()
	c.Logf("doneEarly = %d", doneEarly)
	c.Check(doneLate, check.Equals, int32(0))

	// Unpause backend request(s).
	close(backend.pauseBlockReadUntil)
	wgLate.Wait()
	c.Logf("doneLate = %d", doneLate)
}

// Check that we empty out the heldopen filehandle cache when it
// exceeds heldopenMax entries.
func (s *keepCacheSuite) TestHeldOpen_RollCache(c *check.C) {
	blksize := 64000
	blkcount := 64
	cache, locators := setupCacheWithBlocks(c, blksize, blkcount)
	cache.maxSize = ByteSizeOrPercent(blksize*blkcount + 1)
	cache.sharedCache.heldopenMax = blkcount + 1
	targetsize := blkcount / 4

	// Exercise the cache until we have more heldopen files than
	// targetsize
	for i := 0; i < 100; i++ {
		doConcurrentReads(c, blkcount, cache, locators, blksize)
		waitTidy(cache)
		cache.tidy()
		if len(cache.sharedCache.heldopen) > targetsize {
			break
		}
	}
	c.Assert(len(cache.sharedCache.heldopen) > targetsize, check.Equals, true)

	// Reduce heldopenMax to make sure we roll the cache in the
	// following ReadAt().
	cache.sharedCache.heldopenMax = targetsize / 2
	cache.deleteHeldopen(cache.cacheFile(locators[0][:32]), nil)
	_, err := cache.ReadAt(locators[0], make([]byte, 1234), 0)
	c.Assert(err, check.IsNil)
	c.Check(len(cache.sharedCache.heldopen), check.Equals, 1)
}

// Check that we close our heldopen files when they are deleted by
// another process.
func (s *keepCacheSuite) TestHeldOpen_CloseDeletedFiles(c *check.C) {
	blksize := 64000
	blkcount := 64
	cache, locators := setupCacheWithBlocks(c, blksize, blkcount)
	cache.maxSize = ByteSizeOrPercent(blksize*blkcount + 1)
	cache.sharedCache.heldopenMax = blkcount + 1
	targetsize := blkcount / 4

	// Exercise the cache until we have more heldopen files than
	// targetsize
	for i := 0; i < 100; i++ {
		doConcurrentReads(c, blkcount, cache, locators, blksize)
		waitTidy(cache)
		cache.tidy()
		if len(cache.sharedCache.heldopen) > targetsize {
			break
		}
	}

	c.Logf("len(cache.sharedCache.heldopen) == %d, targetsize == %d", len(cache.sharedCache.heldopen), targetsize)
	c.Assert(len(cache.sharedCache.heldopen) > targetsize, check.Equals, true)

	for i := targetsize; i < blkcount; i++ {
		os.Remove(cache.cacheFile(locators[i][:32]))
	}
	waitTidy(cache)
	cache.tidy()

	c.Logf("len(cache.sharedCache.heldopen) == %d, targetsize == %d", len(cache.sharedCache.heldopen), targetsize)
	c.Check(len(cache.sharedCache.heldopen) <= targetsize, check.Equals, true)
}

var _ = check.Suite(&keepCacheBenchSuite{})

type keepCacheBenchSuite struct {
	blksize  int
	blkcount int
	cache    *DiskCache
	locators []string
}

func (s *keepCacheBenchSuite) SetUpTest(c *check.C) {
	s.blksize = 64000000
	s.blkcount = 8
	s.cache, s.locators = setupCacheWithBlocks(c, s.blksize, s.blkcount)
}

func (s *keepCacheBenchSuite) BenchmarkConcurrentReads_LowNOFiles(c *check.C) {
	s.cache.sharedCache.heldopenMax = 4
	s.BenchmarkConcurrentReads(c)
}

func (s *keepCacheBenchSuite) BenchmarkConcurrentReads(c *check.C) {
	doConcurrentReads(c, c.N, s.cache, s.locators, s.blksize)
}

func (s *keepCacheBenchSuite) BenchmarkSequentialReads(c *check.C) {
	buf := make([]byte, benchReadSize)
	for i := 0; i < c.N; i++ {
		_, err := s.cache.ReadAt(s.locators[i%s.blkcount], buf, int((int64(i)*1234)%int64(s.blksize-benchReadSize)))
		if err != nil {
			c.Fail()
		}
	}
}

const benchReadSize = 1000

var _ = check.Suite(&fileOpsSuite{})

type fileOpsSuite struct{}

// BenchmarkOpenClose and BenchmarkKeepOpen can be used to measure the
// potential performance improvement of caching filehandles rather
// than opening/closing the cache file for each read.
//
// Results from a development machine indicate a ~3x throughput
// improvement: ~636 MB/s when opening/closing the file for each
// 1000-byte read vs. ~2 GB/s when opening the file once and doing
// concurrent reads using the same file descriptor.
func (s *fileOpsSuite) BenchmarkOpenClose(c *check.C) {
	fnm := c.MkDir() + "/testfile"
	os.WriteFile(fnm, make([]byte, 64000000), 0700)
	var wg sync.WaitGroup
	for i := 0; i < c.N; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := os.OpenFile(fnm, os.O_CREATE|os.O_RDWR, 0700)
			if err != nil {
				c.Fail()
				return
			}
			_, err = f.ReadAt(make([]byte, benchReadSize), (int64(i)*1000000)%63123123)
			if err != nil {
				c.Fail()
				return
			}
			f.Close()
		}()
	}
	wg.Wait()
}

func (s *fileOpsSuite) BenchmarkKeepOpen(c *check.C) {
	fnm := c.MkDir() + "/testfile"
	os.WriteFile(fnm, make([]byte, 64000000), 0700)
	f, err := os.OpenFile(fnm, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		c.Fail()
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < c.N; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err = f.ReadAt(make([]byte, benchReadSize), (int64(i)*1000000)%63123123)
			if err != nil {
				c.Fail()
				return
			}
		}()
	}
	wg.Wait()
	f.Close()
}

func setupCacheWithBlocks(c *check.C, blksize, blkcount int) (cache *DiskCache, locators []string) {
	backend := &keepGatewayMemoryBacked{}
	cache = &DiskCache{
		KeepGateway: backend,
		MaxSize:     ByteSizeOrPercent(blksize),
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
		Metrics:     NewKeepClientMetrics(),
	}
	locators = make([]string, blkcount)
	data := make([]byte, blksize)
	for b := 0; b < blkcount; b++ {
		for i := range data {
			data[i] = byte(b)
		}
		resp, err := cache.BlockWrite(context.Background(), BlockWriteOptions{
			Data: data,
		})
		c.Assert(err, check.IsNil)
		locators[b] = resp.Locator
	}
	return
}

func doConcurrentReads(c *check.C, N int, cache *DiskCache, locators []string, blksize int) {
	blkcount := len(locators)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, benchReadSize)
			_, err := cache.ReadAt(locators[i%blkcount], buf, int((int64(i)*1234)%int64(blksize-benchReadSize)))
			if err != nil {
				c.Fail()
			}
		}()
	}
	wg.Wait()
}

func waitTidy(cache *DiskCache) {
	time.Sleep(time.Millisecond)
	for atomic.LoadInt32(&cache.tidying) > 0 {
		time.Sleep(time.Millisecond)
	}
}
