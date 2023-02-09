// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package scheduler uses a resizable worker pool to execute
// containers in priority order.
package scheduler

import (
	"context"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// A Scheduler maps queued containers onto unallocated workers in
// priority order, creating new workers if needed. It locks containers
// that can be mapped onto existing/pending workers, and starts them
// if possible.
//
// A Scheduler unlocks any containers that are locked but can't be
// mapped. (For example, this happens when the cloud provider reaches
// quota/capacity and a previously mappable container's priority is
// surpassed by a newer container.)
//
// If it encounters errors while creating new workers, a Scheduler
// shuts down idle workers, in case they are consuming quota.
type Scheduler struct {
	logger              logrus.FieldLogger
	client              *arvados.Client
	queue               ContainerQueue
	pool                WorkerPool
	reg                 *prometheus.Registry
	staleLockTimeout    time.Duration
	queueUpdateInterval time.Duration

	uuidOp map[string]string // operation in progress: "lock", "cancel", ...
	mtx    sync.Mutex
	wakeup *time.Timer

	runOnce sync.Once
	stop    chan struct{}
	stopped chan struct{}

	last503time    time.Time // last time API responded 503
	maxConcurrency int       // dynamic container limit (0 = unlimited), see runQueue()

	mContainersAllocatedNotStarted   prometheus.Gauge
	mContainersNotAllocatedOverQuota prometheus.Gauge
	mLongestWaitTimeSinceQueue       prometheus.Gauge
}

// New returns a new unstarted Scheduler.
//
// Any given queue and pool should not be used by more than one
// scheduler at a time.
func New(ctx context.Context, client *arvados.Client, queue ContainerQueue, pool WorkerPool, reg *prometheus.Registry, staleLockTimeout, queueUpdateInterval time.Duration) *Scheduler {
	sch := &Scheduler{
		logger:              ctxlog.FromContext(ctx),
		client:              client,
		queue:               queue,
		pool:                pool,
		reg:                 reg,
		staleLockTimeout:    staleLockTimeout,
		queueUpdateInterval: queueUpdateInterval,
		wakeup:              time.NewTimer(time.Second),
		stop:                make(chan struct{}),
		stopped:             make(chan struct{}),
		uuidOp:              map[string]string{},
	}
	sch.registerMetrics(reg)
	return sch
}

func (sch *Scheduler) registerMetrics(reg *prometheus.Registry) {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	sch.mContainersAllocatedNotStarted = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "containers_allocated_not_started",
		Help:      "Number of containers allocated to a worker but not started yet (worker is booting).",
	})
	reg.MustRegister(sch.mContainersAllocatedNotStarted)
	sch.mContainersNotAllocatedOverQuota = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "containers_not_allocated_over_quota",
		Help:      "Number of containers not allocated to a worker because the system has hit a quota.",
	})
	reg.MustRegister(sch.mContainersNotAllocatedOverQuota)
	sch.mLongestWaitTimeSinceQueue = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "containers_longest_wait_time_seconds",
		Help:      "Current longest wait time of any container since queuing, and before the start of crunch-run.",
	})
	reg.MustRegister(sch.mLongestWaitTimeSinceQueue)
}

func (sch *Scheduler) updateMetrics() {
	earliest := time.Time{}
	entries, _ := sch.queue.Entries()
	running := sch.pool.Running()
	for _, ent := range entries {
		if ent.Container.Priority > 0 &&
			(ent.Container.State == arvados.ContainerStateQueued || ent.Container.State == arvados.ContainerStateLocked) {
			// Exclude containers that are preparing to run the payload (i.e.
			// ContainerStateLocked and running on a worker, most likely loading the
			// payload image
			if _, ok := running[ent.Container.UUID]; !ok {
				if ent.Container.CreatedAt.Before(earliest) || earliest.IsZero() {
					earliest = ent.Container.CreatedAt
				}
			}
		}
	}
	if !earliest.IsZero() {
		sch.mLongestWaitTimeSinceQueue.Set(time.Since(earliest).Seconds())
	} else {
		sch.mLongestWaitTimeSinceQueue.Set(0)
	}
}

// Start starts the scheduler.
func (sch *Scheduler) Start() {
	go sch.runOnce.Do(sch.run)
}

// Stop stops the scheduler. No other method should be called after
// Stop.
func (sch *Scheduler) Stop() {
	close(sch.stop)
	<-sch.stopped
}

func (sch *Scheduler) run() {
	defer close(sch.stopped)

	// Ensure the queue is fetched once before attempting anything.
	for err := sch.queue.Update(); err != nil; err = sch.queue.Update() {
		sch.logger.Errorf("error updating queue: %s", err)
		d := sch.queueUpdateInterval / 10
		if d < time.Second {
			d = time.Second
		}
		sch.logger.Infof("waiting %s before retry", d)
		time.Sleep(d)
	}

	// Keep the queue up to date.
	poll := time.NewTicker(sch.queueUpdateInterval)
	defer poll.Stop()
	go func() {
		for range poll.C {
			err := sch.queue.Update()
			if err != nil {
				sch.logger.Errorf("error updating queue: %s", err)
			}
		}
	}()

	t0 := time.Now()
	sch.logger.Infof("FixStaleLocks starting.")
	sch.fixStaleLocks()
	sch.logger.Infof("FixStaleLocks finished (%s), starting scheduling.", time.Since(t0))

	poolNotify := sch.pool.Subscribe()
	defer sch.pool.Unsubscribe(poolNotify)

	queueNotify := sch.queue.Subscribe()
	defer sch.queue.Unsubscribe(queueNotify)

	for {
		sch.runQueue()
		sch.sync()
		sch.updateMetrics()
		select {
		case <-sch.stop:
			return
		case <-queueNotify:
		case <-poolNotify:
		case <-sch.wakeup.C:
		}
	}
}
