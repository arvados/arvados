// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&limiterSuite{})

type limiterSuite struct{}

func (*limiterSuite) TestLimiter(c *C) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	defer cancel()
	rl := requestLimiter{}

	// unlimited concurrency before first call to Report()
	{
		var wg sync.WaitGroup
		wg.Add(1000)
		for i := 0; i < 1000; i++ {
			go func() {
				rl.Acquire(ctx)
				wg.Done()
			}()
		}
		wg.Wait()
		c.Check(rl.current, Equals, int64(1000))
		wg.Add(1000)
		for i := 0; i < 1000; i++ {
			go func() {
				rl.Release()
				wg.Done()
			}()
		}
		wg.Wait()
		c.Check(rl.current, Equals, int64(0))
	}

	// context cancels while waiting for Acquire
	{
		rl.limit = 1
		rl.Acquire(ctx)
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Millisecond))
		defer cancel()
		rl.Acquire(ctx)
		c.Check(rl.current, Equals, int64(2))
		c.Check(ctx.Err(), NotNil)
		rl.Release()
		rl.Release()
		c.Check(rl.current, Equals, int64(0))
	}

	// Use a short quiet period to make tests faster
	defer func(orig time.Duration) { requestLimiterQuietPeriod = orig }(requestLimiterQuietPeriod)
	requestLimiterQuietPeriod = time.Second / 10

	// Immediately after a 503, limit is decreased, and Acquire()
	// waits for a quiet period
	{
		rl.limit = 0
		for i := 0; i < 5; i++ {
			rl.Acquire(ctx)
		}
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(requestLimiterQuietPeriod/10))
		defer cancel()
		rl.Report(&http.Response{StatusCode: http.StatusServiceUnavailable}, nil)
		c.Check(rl.limit, Equals, int64(3))

		for i := 0; i < 5; i++ {
			rl.Release()
		}

		// Even with all slots released, we can't Acquire in
		// the quiet period.
		acquire := time.Now()
		rl.Acquire(ctx)
		c.Check(ctx.Err(), Equals, context.DeadlineExceeded)
		c.Check(time.Since(acquire) < requestLimiterQuietPeriod/2, Equals, true)
		c.Check(rl.quietUntil.Sub(time.Now()) > requestLimiterQuietPeriod/2, Equals, true)
		rl.Release()
	}

	// Acquire waits for the quiet period to expire.
	{
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(requestLimiterQuietPeriod*2))
		defer cancel()
		acquire := time.Now()
		rl.Acquire(ctx)
		c.Check(time.Since(acquire) > requestLimiterQuietPeriod/10, Equals, true)
		c.Check(time.Since(acquire) < requestLimiterQuietPeriod, Equals, true)
		c.Check(ctx.Err(), IsNil)
		rl.Release()
	}

	// OK to call Report() with nil Response and non-nil error.
	rl.Report(nil, errors.New("network error"))
}
