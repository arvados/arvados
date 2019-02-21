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

	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
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
	queue               ContainerQueue
	pool                WorkerPool
	staleLockTimeout    time.Duration
	queueUpdateInterval time.Duration

	uuidOp map[string]string // operation in progress: "lock", "cancel", ...
	mtx    sync.Mutex
	wakeup *time.Timer

	runOnce sync.Once
	stop    chan struct{}
	stopped chan struct{}
}

// New returns a new unstarted Scheduler.
//
// Any given queue and pool should not be used by more than one
// scheduler at a time.
func New(ctx context.Context, queue ContainerQueue, pool WorkerPool, staleLockTimeout, queueUpdateInterval time.Duration) *Scheduler {
	return &Scheduler{
		logger:              ctxlog.FromContext(ctx),
		queue:               queue,
		pool:                pool,
		staleLockTimeout:    staleLockTimeout,
		queueUpdateInterval: queueUpdateInterval,
		wakeup:              time.NewTimer(time.Second),
		stop:                make(chan struct{}),
		stopped:             make(chan struct{}),
		uuidOp:              map[string]string{},
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
		select {
		case <-sch.stop:
			return
		case <-queueNotify:
		case <-poolNotify:
		case <-sch.wakeup.C:
		}
	}
}
