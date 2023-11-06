// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"container/heap"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const MinPriority = math.MinInt64

// Prometheus typically polls every 10 seconds, but it doesn't cost us
// much to also accommodate higher frequency collection by updating
// internal stats more frequently. (This limits time resolution only
// for the metrics that aren't generated on the fly.)
const metricsUpdateInterval = time.Second

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

	// Return 503 for any request for which Priority() returns
	// MinPriority if it spends longer than this in the queue
	// before starting processing.
	MaxQueueTimeForMinPriority time.Duration

	// "concurrent_requests", "max_concurrent_requests",
	// "queued_requests", and "max_queued_requests" metrics are
	// registered with Registry, if it is not nil.
	Registry *prometheus.Registry

	setupOnce     sync.Once
	mQueueDelay   *prometheus.SummaryVec
	mQueueTimeout *prometheus.SummaryVec
	mQueueUsage   *prometheus.GaugeVec
	mtx           sync.Mutex
	handling      int
	queue         queue
}

type qent struct {
	queued   time.Time
	priority int64
	heappos  int
	ready    chan bool // true = handle now; false = return 503 now
}

type queue []*qent

func (h queue) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heappos, h[j].heappos = i, j
}

func (h queue) Less(i, j int) bool {
	pi, pj := h[i].priority, h[j].priority
	return pi > pj || (pi == pj && h[i].queued.Before(h[j].queued))
}

func (h queue) Len() int {
	return len(h)
}

func (h *queue) Push(x interface{}) {
	n := len(*h)
	ent := x.(*qent)
	ent.heappos = n
	*h = append(*h, ent)
}

func (h *queue) Pop() interface{} {
	n := len(*h)
	ent := (*h)[n-1]
	ent.heappos = -1
	(*h)[n-1] = nil
	*h = (*h)[0 : n-1]
	return ent
}

func (h *queue) add(ent *qent) {
	ent.heappos = h.Len()
	h.Push(ent)
}

func (h *queue) removeMax() *qent {
	return heap.Pop(h).(*qent)
}

func (h *queue) remove(i int) {
	heap.Remove(h, i)
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
		rl.mQueueUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "arvados",
			Name:      "queued_requests",
			Help:      "Number of requests in queue",
		}, []string{"priority"})
		rl.Registry.MustRegister(rl.mQueueUsage)
		rl.Registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "max_queued_requests",
				Help:      "Maximum number of queued requests",
			},
			func() float64 { return float64(rl.MaxQueue) },
		))
		rl.mQueueDelay = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "arvados",
			Name:       "queue_delay_seconds",
			Help:       "Time spent in the incoming request queue before start of processing",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		}, []string{"priority"})
		rl.Registry.MustRegister(rl.mQueueDelay)
		rl.mQueueTimeout = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "arvados",
			Name:       "queue_timeout_seconds",
			Help:       "Time spent in the incoming request queue before client timed out or disconnected",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		}, []string{"priority"})
		rl.Registry.MustRegister(rl.mQueueTimeout)
		go func() {
			for range time.NewTicker(metricsUpdateInterval).C {
				var low, normal, high int
				rl.mtx.Lock()
				for _, ent := range rl.queue {
					switch {
					case ent.priority < 0:
						low++
					case ent.priority > 0:
						high++
					default:
						normal++
					}
				}
				rl.mtx.Unlock()
				rl.mQueueUsage.WithLabelValues("low").Set(float64(low))
				rl.mQueueUsage.WithLabelValues("normal").Set(float64(normal))
				rl.mQueueUsage.WithLabelValues("high").Set(float64(high))
			}
		}()
	}
}

// caller must have lock
func (rl *RequestLimiter) runqueue() {
	// Handle entries from the queue as capacity permits
	for len(rl.queue) > 0 && (rl.MaxConcurrent == 0 || rl.handling < rl.MaxConcurrent) {
		rl.handling++
		ent := rl.queue.removeMax()
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
	rl.queue.add(ent)
	rl.trimqueue()
	return ent
}

func (rl *RequestLimiter) remove(ent *qent) {
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	if ent.heappos >= 0 {
		rl.queue.remove(ent.heappos)
		ent.ready <- false
	}
}

func (rl *RequestLimiter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rl.setupOnce.Do(rl.setup)
	ent := rl.enqueue(req)
	SetResponseLogFields(req.Context(), logrus.Fields{"priority": ent.priority})
	if ent.priority == MinPriority {
		// Note that MaxQueueTime==0 does not cancel a req
		// that skips the queue, because in that case
		// rl.enqueue() has already fired ready<-true and
		// rl.remove() is a no-op.
		go func() {
			time.Sleep(rl.MaxQueueTimeForMinPriority)
			rl.remove(ent)
		}()
	}
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

	// Report time spent in queue in the appropriate bucket:
	// mQueueDelay if the request actually got processed,
	// mQueueTimeout if it was abandoned or cancelled before
	// getting a processing slot.
	var series *prometheus.SummaryVec
	if ok {
		series = rl.mQueueDelay
	} else {
		series = rl.mQueueTimeout
	}
	if series != nil {
		var qlabel string
		switch {
		case ent.priority < 0:
			qlabel = "low"
		case ent.priority > 0:
			qlabel = "high"
		default:
			qlabel = "normal"
		}
		series.WithLabelValues(qlabel).Observe(time.Now().Sub(ent.queued).Seconds())
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
