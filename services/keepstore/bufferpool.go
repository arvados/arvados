// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var bufferPoolBlockSize = BlockSize // modified by tests

type bufferPool struct {
	log logrus.FieldLogger
	// limiter has a "true" placeholder for each in-use buffer.
	limiter chan bool
	// allocated is the number of bytes currently allocated to buffers.
	allocated uint64
	// Pool has unused buffers.
	sync.Pool
}

func newBufferPool(log logrus.FieldLogger, count int, reg *prometheus.Registry) *bufferPool {
	p := bufferPool{log: log}
	p.Pool.New = func() interface{} {
		atomic.AddUint64(&p.allocated, uint64(bufferPoolBlockSize))
		return make([]byte, bufferPoolBlockSize)
	}
	p.limiter = make(chan bool, count)
	if reg != nil {
		reg.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Subsystem: "keepstore",
				Name:      "bufferpool_allocated_bytes",
				Help:      "Number of bytes allocated to buffers",
			},
			func() float64 { return float64(p.Alloc()) },
		))
		reg.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Subsystem: "keepstore",
				Name:      "bufferpool_max_buffers",
				Help:      "Maximum number of buffers allowed",
			},
			func() float64 { return float64(p.Cap()) },
		))
		reg.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Subsystem: "keepstore",
				Name:      "bufferpool_inuse_buffers",
				Help:      "Number of buffers in use",
			},
			func() float64 { return float64(p.Len()) },
		))
	}
	return &p
}

// GetContext gets a buffer from the pool -- but gives up and returns
// ctx.Err() if ctx ends before a buffer is available.
func (p *bufferPool) GetContext(ctx context.Context) ([]byte, error) {
	bufReady := make(chan []byte)
	go func() {
		bufReady <- p.Get()
	}()
	select {
	case buf := <-bufReady:
		return buf, nil
	case <-ctx.Done():
		go func() {
			// Even if closeNotifier happened first, we
			// need to keep waiting for our buf so we can
			// return it to the pool.
			p.Put(<-bufReady)
		}()
		return nil, ctx.Err()
	}
}

func (p *bufferPool) Get() []byte {
	select {
	case p.limiter <- true:
	default:
		t0 := time.Now()
		p.log.Printf("reached max buffers (%d), waiting", cap(p.limiter))
		p.limiter <- true
		p.log.Printf("waited %v for a buffer", time.Since(t0))
	}
	buf := p.Pool.Get().([]byte)
	if len(buf) < bufferPoolBlockSize {
		p.log.Fatalf("bufferPoolBlockSize=%d but cap(buf)=%d", bufferPoolBlockSize, len(buf))
	}
	return buf
}

func (p *bufferPool) Put(buf []byte) {
	p.Pool.Put(buf[:cap(buf)])
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
