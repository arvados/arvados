// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
	"net/http"
	"sync"
	"time"
)

var requestLimiterQuietPeriod = time.Second

type requestLimiter struct {
	current    int64
	limit      int64
	lock       sync.Mutex
	cond       *sync.Cond
	quietUntil time.Time
}

// Acquire reserves one request slot, waiting if necessary.
//
// Acquire returns early if ctx cancels before a slot is available. It
// is assumed in this case the caller will immediately notice
// ctx.Err() != nil and call Release().
func (rl *requestLimiter) Acquire(ctx context.Context) {
	rl.lock.Lock()
	if rl.cond == nil {
		// First use of requestLimiter. Initialize.
		rl.cond = sync.NewCond(&rl.lock)
	}
	// Wait out the quiet period(s) immediately following a 503.
	for ctx.Err() == nil {
		delay := rl.quietUntil.Sub(time.Now())
		if delay < 0 {
			break
		}
		// Wait for the end of the quiet period, which started
		// when we last received a 503 response.
		rl.lock.Unlock()
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
		rl.lock.Lock()
	}
	ready := make(chan struct{})
	go func() {
		// close ready when a slot is available _or_ we wake
		// up and find ctx has been canceled (meaning Acquire
		// has already returned, or is about to).
		for rl.limit > 0 && rl.limit <= rl.current && ctx.Err() == nil {
			rl.cond.Wait()
		}
		close(ready)
	}()
	select {
	case <-ready:
		// Wait() returned, so we have the lock.
		rl.current++
		rl.lock.Unlock()
	case <-ctx.Done():
		// When Wait() returns the lock to our goroutine
		// (which might have already happened) we need to
		// release it (if we don't do this now, the following
		// Lock() can deadlock).
		go func() {
			<-ready
			rl.lock.Unlock()
		}()
		// Note we may have current > limit until the caller
		// calls Release().
		rl.lock.Lock()
		rl.current++
		rl.lock.Unlock()
	}
}

// Release releases a slot that has been reserved with Acquire.
func (rl *requestLimiter) Release() {
	rl.lock.Lock()
	rl.current--
	rl.lock.Unlock()
	rl.cond.Signal()
}

// Report uses the return values from (*http.Client)Do() to adjust the
// outgoing request limit (increase on success, decrease on 503).
func (rl *requestLimiter) Report(resp *http.Response, err error) {
	if err != nil {
		return
	}
	rl.lock.Lock()
	defer rl.lock.Unlock()
	if resp.StatusCode == http.StatusServiceUnavailable {
		if rl.limit == 0 {
			// Concurrency was unlimited until now.
			// Calculate new limit based on actual
			// concurrency instead of previous limit.
			rl.limit = rl.current
		}
		if time.Now().After(rl.quietUntil) {
			// Reduce concurrency limit by half.
			rl.limit = (rl.limit + 1) / 2
			// Don't start any new calls (or reduce the
			// limit even further on additional 503s) for
			// a second.
			rl.quietUntil = time.Now().Add(requestLimiterQuietPeriod)
		}
	} else if resp.StatusCode >= 200 && resp.StatusCode < 400 && rl.limit > 0 {
		// After each non-server-error response, increase
		// concurrency limit by at least 10% -- but not beyond
		// 2x the highest concurrency level we've seen without
		// a failure.
		increase := rl.limit / 10
		if increase < 1 {
			increase = 1
		}
		rl.limit += increase
		if max := rl.current * 2; max > rl.limit {
			rl.limit = max
		}
		rl.cond.Broadcast()
	}
}
