// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// RequestLimiter wraps http.Handler, limiting the number of
// concurrent requests being handled by the wrapped Handler. Requests
// that arrive when the handler is already at the specified
// concurrency limit are queued and handled in the order indicated by
// the Priority function.
//
// Caller must not modify any RequestLimiter fields after calling its
// methods.
type RequestLimiter struct {
	Handler http.Handler

	// Maximum number of requests being handled at once. Beyond
	// this limit, requests will be queued.
	MaxConcurrent int

	// Maximum number of requests in the queue. Beyond this limit,
	// the lowest priority requests will return 503.
	MaxQueue int

	// Priority determines queue ordering. Requests with higher
	// priority are handled first. Requests with equal priority
	// are handled FIFO. If Priority is nil, all requests are
	// handled FIFO.
	Priority func(req *http.Request, queued time.Time) int64

	// "concurrent_requests", "max_concurrent_requests",
	// "queued_requests", and "max_queued_requests" metrics are
	// registered with Registry, if it is not nil.
	Registry *prometheus.Registry

	setupOnce sync.Once
	mtx       sync.Mutex
	handling  int
	queue     heap
}

type qent struct {
	queued   time.Time
	priority int64
	heappos  int
	ready    chan bool // true = handle now; false = return 503 now
}

type heap []*qent

func (h heap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heappos, h[j].heappos = i, j
}

func (h heap) Less(i, j int) bool {
	pi, pj := h[i].priority, h[j].priority
	return pi > pj || (pi == pj && h[i].queued.Before(h[j].queued))
}

func (h heap) Len() int {
	return len(h)
}

// Move element i to a correct position in the heap. When the heap is
// empty, fix(0) is a no-op (does not crash).
func (h heap) fix(i int) {
	// If the initial position is a leaf (i.e., index is greater
	// than the last node's parent index), we only need to move it
	// up, not down.
	uponly := i > (len(h)-2)/2
	// Move the new entry toward the root until reaching a
	// position where the parent already has higher priority.
	for i > 0 {
		parent := (i - 1) / 2
		if h.Less(i, parent) {
			h.Swap(i, parent)
			i = parent
		} else {
			break
		}
	}
	// Move i away from the root until reaching a position where
	// both children already have lower priority.
	for !uponly {
		child := i*2 + 1
		if child+1 < len(h) && h.Less(child+1, child) {
			// Right child has higher priority than left
			// child. Choose right child.
			child = child + 1
		}
		if child < len(h) && h.Less(child, i) {
			// Chosen child has higher priority than i.
			// Swap and continue down.
			h.Swap(i, child)
			i = child
		} else {
			break
		}
	}
}

func (h *heap) add(ent *qent) {
	ent.heappos = len(*h)
	*h = append(*h, ent)
	h.fix(ent.heappos)
}

func (h *heap) removeMax() *qent {
	ent := (*h)[0]
	if len(*h) == 1 {
		*h = (*h)[:0]
	} else {
		h.Swap(0, len(*h)-1)
		*h = (*h)[:len(*h)-1]
		h.fix(0)
	}
	ent.heappos = -1
	return ent
}

func (h *heap) remove(i int) {
	// Move the last leaf into i's place, then move it to a
	// correct position.
	h.Swap(i, len(*h)-1)
	*h = (*h)[:len(*h)-1]
	if i < len(*h) {
		h.fix(i)
	}
}

func (rl *RequestLimiter) setup() {
	if rl.Registry != nil {
		rl.Registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "concurrent_requests",
				Help:      "Number of requests in progress",
			},
			func() float64 {
				rl.mtx.Lock()
				defer rl.mtx.Unlock()
				return float64(rl.handling)
			},
		))
		rl.Registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "max_concurrent_requests",
				Help:      "Maximum number of concurrent requests",
			},
			func() float64 { return float64(rl.MaxConcurrent) },
		))
		rl.Registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "queued_requests",
				Help:      "Number of requests in queue",
			},
			func() float64 {
				rl.mtx.Lock()
				defer rl.mtx.Unlock()
				return float64(len(rl.queue))
			},
		))
		rl.Registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "max_queued_requests",
				Help:      "Maximum number of queued requests",
			},
			func() float64 { return float64(rl.MaxQueue) },
		))
	}
}

// caller must have lock
func (rl *RequestLimiter) runqueue() {
	// Handle entries from the queue as capacity permits
	for len(rl.queue) > 0 && (rl.MaxConcurrent == 0 || rl.handling < rl.MaxConcurrent) {
		rl.handling++
		ent := rl.queue.removeMax()
		ent.heappos = -1
		ent.ready <- true
	}
}

// If the queue is too full, fail and remove the lowest-priority
// entry. Caller must have lock. Queue must not be empty.
func (rl *RequestLimiter) trimqueue() {
	if len(rl.queue) <= rl.MaxQueue {
		return
	}
	min := 0
	for i := range rl.queue {
		if i == 0 || rl.queue.Less(min, i) {
			min = i
		}
	}
	rl.queue[min].heappos = -1
	rl.queue[min].ready <- false
	rl.queue.remove(min)
}

func (rl *RequestLimiter) enqueue(req *http.Request) *qent {
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	qtime := time.Now()
	var priority int64
	if rl.Priority != nil {
		priority = rl.Priority(req, qtime)
	}
	ent := &qent{
		queued:   qtime,
		priority: priority,
		ready:    make(chan bool, 1),
		heappos:  -1,
	}
	if rl.MaxConcurrent == 0 || rl.MaxConcurrent > rl.handling {
		// fast path, skip the queue
		rl.handling++
		ent.ready <- true
		return ent
	}
	if priority == math.MinInt64 {
		// Priority func is telling us to return 503
		// immediately instead of queueing, regardless of
		// queue size, if we can't handle the request
		// immediately.
		ent.ready <- false
		return ent
	}
	rl.queue.add(ent)
	rl.trimqueue()
	return ent
}

func (rl *RequestLimiter) remove(ent *qent) {
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	if ent.heappos >= 0 {
		rl.queue.remove(ent.heappos)
		ent.heappos = -1
		ent.ready <- false
	}
}

func (rl *RequestLimiter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rl.setupOnce.Do(rl.setup)
	ent := rl.enqueue(req)
	SetResponseLogFields(req.Context(), logrus.Fields{"priority": ent.priority})
	var ok bool
	select {
	case <-req.Context().Done():
		rl.remove(ent)
		// we still need to wait for ent.ready, because
		// sometimes runqueue() will have already decided to
		// send true before our rl.remove() call, and in that
		// case we'll need to decrement rl.handling below.
		ok = <-ent.ready
	case ok = <-ent.ready:
	}
	if !ok {
		resp.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer func() {
		rl.mtx.Lock()
		defer rl.mtx.Unlock()
		rl.handling--
		// unblock the next waiting request
		rl.runqueue()
	}()
	rl.Handler.ServeHTTP(resp, req)
}
