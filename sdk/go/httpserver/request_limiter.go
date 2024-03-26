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

	// Queue determines which queue a request is assigned to.
	Queue func(req *http.Request) *RequestQueue

	// Priority determines queue ordering. Requests with higher
	// priority are handled first. Requests with equal priority
	// are handled FIFO. If Priority is nil, all requests are
	// handled FIFO.
	Priority func(req *http.Request, queued time.Time) int64

	// "concurrent_requests", "max_concurrent_requests",
	// "queued_requests", and "max_queued_requests" metrics are
	// registered with Registry, if it is not nil.
	Registry *prometheus.Registry

	setupOnce     sync.Once
	mQueueDelay   *prometheus.SummaryVec
	mQueueTimeout *prometheus.SummaryVec
	mQueueUsage   *prometheus.GaugeVec
	mtx           sync.Mutex
	rqs           map[*RequestQueue]bool // all RequestQueues in use
}

type RequestQueue struct {
	// Label for metrics. No two queues should have the same label.
	Label string

	// Maximum number of requests being handled at once. Beyond
	// this limit, requests will be queued.
	MaxConcurrent int

	// Maximum number of requests in the queue. Beyond this limit,
	// the lowest priority requests will return 503.
	MaxQueue int

	// Return 503 for any request for which Priority() returns
	// MinPriority if it spends longer than this in the queue
	// before starting processing.
	MaxQueueTimeForMinPriority time.Duration

	queue    queue
	handling int
}

type qent struct {
	rq       *RequestQueue
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
		mCurrentReqs := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "arvados",
			Name:      "concurrent_requests",
			Help:      "Number of requests in progress",
		}, []string{"queue"})
		rl.Registry.MustRegister(mCurrentReqs)
		mMaxReqs := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "arvados",
			Name:      "max_concurrent_requests",
			Help:      "Maximum number of concurrent requests",
		}, []string{"queue"})
		rl.Registry.MustRegister(mMaxReqs)
		mMaxQueue := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "arvados",
			Name:      "max_queued_requests",
			Help:      "Maximum number of queued requests",
		}, []string{"queue"})
		rl.Registry.MustRegister(mMaxQueue)
		rl.mQueueUsage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "arvados",
			Name:      "queued_requests",
			Help:      "Number of requests in queue",
		}, []string{"queue", "priority"})
		rl.Registry.MustRegister(rl.mQueueUsage)
		rl.mQueueDelay = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "arvados",
			Name:       "queue_delay_seconds",
			Help:       "Time spent in the incoming request queue before start of processing",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		}, []string{"queue", "priority"})
		rl.Registry.MustRegister(rl.mQueueDelay)
		rl.mQueueTimeout = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  "arvados",
			Name:       "queue_timeout_seconds",
			Help:       "Time spent in the incoming request queue before client timed out or disconnected",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		}, []string{"queue", "priority"})
		rl.Registry.MustRegister(rl.mQueueTimeout)
		go func() {
			for range time.NewTicker(metricsUpdateInterval).C {
				rl.mtx.Lock()
				for rq := range rl.rqs {
					var low, normal, high int
					for _, ent := range rq.queue {
						switch {
						case ent.priority < 0:
							low++
						case ent.priority > 0:
							high++
						default:
							normal++
						}
					}
					mCurrentReqs.WithLabelValues(rq.Label).Set(float64(rq.handling))
					mMaxReqs.WithLabelValues(rq.Label).Set(float64(rq.MaxConcurrent))
					mMaxQueue.WithLabelValues(rq.Label).Set(float64(rq.MaxQueue))
					rl.mQueueUsage.WithLabelValues(rq.Label, "low").Set(float64(low))
					rl.mQueueUsage.WithLabelValues(rq.Label, "normal").Set(float64(normal))
					rl.mQueueUsage.WithLabelValues(rq.Label, "high").Set(float64(high))
				}
				rl.mtx.Unlock()
			}
		}()
	}
}

// caller must have lock
func (rq *RequestQueue) runqueue() {
	// Handle entries from the queue as capacity permits
	for len(rq.queue) > 0 && (rq.MaxConcurrent == 0 || rq.handling < rq.MaxConcurrent) {
		rq.handling++
		ent := rq.queue.removeMax()
		ent.ready <- true
	}
}

// If the queue is too full, fail and remove the lowest-priority
// entry. Caller must have lock. Queue must not be empty.
func (rq *RequestQueue) trimqueue() {
	if len(rq.queue) <= rq.MaxQueue {
		return
	}
	min := 0
	for i := range rq.queue {
		if i == 0 || rq.queue.Less(min, i) {
			min = i
		}
	}
	rq.queue[min].ready <- false
	rq.queue.remove(min)
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
		rq:       rl.Queue(req),
		queued:   qtime,
		priority: priority,
		ready:    make(chan bool, 1),
		heappos:  -1,
	}
	if rl.rqs == nil {
		rl.rqs = map[*RequestQueue]bool{}
	}
	rl.rqs[ent.rq] = true
	if ent.rq.MaxConcurrent == 0 || ent.rq.MaxConcurrent > ent.rq.handling {
		// fast path, skip the queue
		ent.rq.handling++
		ent.ready <- true
		return ent
	}
	ent.rq.queue.add(ent)
	ent.rq.trimqueue()
	return ent
}

func (rl *RequestLimiter) remove(ent *qent) {
	rl.mtx.Lock()
	defer rl.mtx.Unlock()
	if ent.heappos >= 0 {
		ent.rq.queue.remove(ent.heappos)
		ent.ready <- false
	}
}

func (rl *RequestLimiter) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rl.setupOnce.Do(rl.setup)
	ent := rl.enqueue(req)
	SetResponseLogFields(req.Context(), logrus.Fields{"priority": ent.priority, "queue": ent.rq.Label})
	if ent.priority == MinPriority {
		// Note that MaxQueueTime==0 does not cancel a req
		// that skips the queue, because in that case
		// rl.enqueue() has already fired ready<-true and
		// rl.remove() is a no-op.
		go func() {
			time.Sleep(ent.rq.MaxQueueTimeForMinPriority)
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
		// case we'll need to decrement ent.rq.handling below.
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
		series.WithLabelValues(ent.rq.Label, qlabel).Observe(time.Now().Sub(ent.queued).Seconds())
	}

	if !ok {
		resp.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer func() {
		rl.mtx.Lock()
		defer rl.mtx.Unlock()
		ent.rq.handling--
		// unblock the next waiting request
		ent.rq.runqueue()
	}()
	rl.Handler.ServeHTTP(resp, req)
}
