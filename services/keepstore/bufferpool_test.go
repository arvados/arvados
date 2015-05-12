package main

import (
	. "gopkg.in/check.v1"
	"testing"
	"time"
)

// Gocheck boilerplate
func TestBufferPool(t *testing.T) {
	TestingT(t)
}
var _ = Suite(&BufferPoolSuite{})
type BufferPoolSuite struct {}

// Initialize a default-sized buffer pool for the benefit of test
// suites that don't run main().
func init() {
	bufs = newBufferPool(maxBuffers, BLOCKSIZE)
}

func (s *BufferPoolSuite) TestBufferPoolBufSize(c *C) {
	bufs := newBufferPool(2, 10)
	b1 := bufs.Get(1)
	bufs.Get(2)
	bufs.Put(b1)
	b3 := bufs.Get(3)
	c.Check(len(b3), Equals, 3)
}

func (s *BufferPoolSuite) TestBufferPoolUnderLimit(c *C) {
	bufs := newBufferPool(3, 10)
	b1 := bufs.Get(10)
	bufs.Get(10)
	testBufferPoolRace(c, bufs, b1, "Get")
}

func (s *BufferPoolSuite) TestBufferPoolAtLimit(c *C) {
	bufs := newBufferPool(2, 10)
	b1 := bufs.Get(10)
	bufs.Get(10)
	testBufferPoolRace(c, bufs, b1, "Put")
}

func testBufferPoolRace(c *C, bufs *bufferPool, unused []byte, expectWin string) {
	race := make(chan string)
	go func() {
		bufs.Get(10)
		time.Sleep(time.Millisecond)
		race <- "Get"
	}()
	go func() {
		time.Sleep(10*time.Millisecond)
		bufs.Put(unused)
		race <- "Put"
	}()
	c.Check(<-race, Equals, expectWin)
	c.Check(<-race, Not(Equals), expectWin)
	close(race)
}

func (s *BufferPoolSuite) TestBufferPoolReuse(c *C) {
	bufs := newBufferPool(2, 10)
	bufs.Get(10)
	last := bufs.Get(10)
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
		next := bufs.Get(10)
		copy(last, []byte("last"))
		copy(next, []byte("next"))
		if last[0] == 'n' {
			reuses++
		}
		last = next
	}
	c.Check(reuses > allocs * 95/100, Equals, true)
}
