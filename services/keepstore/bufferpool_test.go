// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	. "gopkg.in/check.v1"
)

var _ = Suite(&BufferPoolSuite{})

var bufferPoolTestSize = 10

type BufferPoolSuite struct{}

func (s *BufferPoolSuite) SetUpTest(c *C) {
	bufferPoolBlockSize = bufferPoolTestSize
}

func (s *BufferPoolSuite) TearDownTest(c *C) {
	bufferPoolBlockSize = BlockSize
}

func (s *BufferPoolSuite) TestBufferPoolBufSize(c *C) {
	bufs := newBufferPool(ctxlog.TestLogger(c), 2, prometheus.NewRegistry())
	b1 := bufs.Get()
	bufs.Get()
	bufs.Put(b1)
	b3 := bufs.Get()
	c.Check(len(b3), Equals, bufferPoolTestSize)
}

func (s *BufferPoolSuite) TestBufferPoolUnderLimit(c *C) {
	bufs := newBufferPool(ctxlog.TestLogger(c), 3, prometheus.NewRegistry())
	b1 := bufs.Get()
	bufs.Get()
	testBufferPoolRace(c, bufs, b1, "Get")
}

func (s *BufferPoolSuite) TestBufferPoolAtLimit(c *C) {
	bufs := newBufferPool(ctxlog.TestLogger(c), 2, prometheus.NewRegistry())
	b1 := bufs.Get()
	bufs.Get()
	testBufferPoolRace(c, bufs, b1, "Put")
}

func testBufferPoolRace(c *C, bufs *bufferPool, unused []byte, expectWin string) {
	race := make(chan string)
	go func() {
		bufs.Get()
		time.Sleep(time.Millisecond)
		race <- "Get"
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		bufs.Put(unused)
		race <- "Put"
	}()
	c.Check(<-race, Equals, expectWin)
	c.Check(<-race, Not(Equals), expectWin)
	close(race)
}

func (s *BufferPoolSuite) TestBufferPoolReuse(c *C) {
	bufs := newBufferPool(ctxlog.TestLogger(c), 2, prometheus.NewRegistry())
	bufs.Get()
	last := bufs.Get()
	// The buffer pool is allowed to throw away unused buffers
	// (e.g., during sync.Pool's garbage collection hook, in the
	// the current implementation). However, if unused buffers are
	// getting thrown away and reallocated more than {arbitrary
	// frequency threshold} during a busy loop, it's not acting
	// much like a buffer pool.
	allocs := 1000
	reuses := 0
	for i := 0; i < allocs; i++ {
		bufs.Put(last)
		next := bufs.Get()
		copy(last, []byte("last"))
		copy(next, []byte("next"))
		if last[0] == 'n' {
			reuses++
		}
		last = next
	}
	c.Check(reuses > allocs*95/100, Equals, true)
}
