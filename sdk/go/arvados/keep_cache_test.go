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
	"sync"
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
	mtx  sync.RWMutex
	data map[string][]byte
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
	k.mtx.RLock()
	data := k.data[opts.Locator]
	k.mtx.RUnlock()
	if data == nil {
		return 0, errors.New("block not found: " + opts.Locator)
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

func (s *keepCacheSuite) TestMaxSize(c *check.C) {
	backend := &keepGatewayMemoryBacked{}
	cache := DiskCache{
		KeepGateway: backend,
		MaxSize:     40000000,
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
	}
	ctx := context.Background()
	resp1, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, 44000000),
	})
	c.Check(err, check.IsNil)
	time.Sleep(time.Millisecond)
	resp2, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, 32000000),
	})
	c.Check(err, check.IsNil)
	delete(backend.data, resp1.Locator)
	delete(backend.data, resp2.Locator)
	cache.tidyHoldUntil = time.Time{}
	cache.tidy()

	n, err := cache.ReadAt(resp1.Locator, make([]byte, 2), 0)
	c.Check(n, check.Equals, 0)
	c.Check(err, check.ErrorMatches, `block not found: .*\+44000000`)

	n, err = cache.ReadAt(resp2.Locator, make([]byte, 2), 0)
	c.Check(n > 0, check.Equals, true)
	c.Check(err, check.IsNil)
}

func (s *keepCacheSuite) TestConcurrentReaders(c *check.C) {
	blksize := 64000000
	backend := &keepGatewayMemoryBacked{}
	cache := DiskCache{
		KeepGateway: backend,
		MaxSize:     int64(blksize),
		Dir:         c.MkDir(),
		Logger:      ctxlog.TestLogger(c),
	}
	ctx := context.Background()
	resp, err := cache.BlockWrite(ctx, BlockWriteOptions{
		Data: make([]byte, blksize),
	})
	c.Check(err, check.IsNil)
	delete(backend.data, resp.Locator)

	failed := false
	var wg sync.WaitGroup
	for offset := 0; offset < blksize; offset += 123456 {
		offset := offset
		wg.Add(1)
		go func() {
			defer wg.Done()
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
