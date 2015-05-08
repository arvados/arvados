package main

import (
	"log"
	"sync"
	"time"
)

type bufferPool struct {
	// limiter has a "true" placeholder for each in-use buffer.
	limiter chan bool
	// Pool has unused buffers.
	sync.Pool
}

func newBufferPool(count int, bufSize int) *bufferPool {
	p := bufferPool{}
	p.New = func() interface{} {
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
