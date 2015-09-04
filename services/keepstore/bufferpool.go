package main

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type bufferPool struct {
	// limiter has a "true" placeholder for each in-use buffer.
	limiter chan bool
	// allocated is the number of bytes currently allocated to buffers.
	allocated uint64
	// Pool has unused buffers.
	sync.Pool
}

func newBufferPool(count int, bufSize int) *bufferPool {
	p := bufferPool{}
	p.New = func() interface{} {
		atomic.AddUint64(&p.allocated, uint64(bufSize))
		return make([]byte, bufSize)
	}
	p.limiter = make(chan bool, count)
	return &p
}

func (p *bufferPool) Get(size int) []byte {
	select {
	case p.limiter <- true:
	default:
		t0 := time.Now()
		log.Printf("reached max buffers (%d), waiting", cap(p.limiter))
		p.limiter <- true
		log.Printf("waited %v for a buffer", time.Since(t0))
	}
	buf := p.Pool.Get().([]byte)
	if cap(buf) < size {
		log.Fatalf("bufferPool Get(size=%d) but max=%d", size, cap(buf))
	}
	return buf[:size]
}

func (p *bufferPool) Put(buf []byte) {
	p.Pool.Put(buf)
	<-p.limiter
}

// Alloc returns the number of bytes allocated to buffers.
func (p *bufferPool) Alloc() uint64 {
	return atomic.LoadUint64(&p.allocated)
}

// Cap returns the maximum number of buffers allowed.
func (p *bufferPool) Cap() int {
	return cap(p.limiter)
}

// Len returns the number of buffers in use right now.
func (p *bufferPool) Len() int {
	return len(p.limiter)
}
