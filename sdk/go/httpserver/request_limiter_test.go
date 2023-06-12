// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"time"

	check "gopkg.in/check.v1"
)

type testHandler struct {
	inHandler   chan struct{}
	okToProceed chan struct{}
}

func (h *testHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	h.inHandler <- struct{}{}
	<-h.okToProceed
}

func newTestHandler() *testHandler {
	return &testHandler{
		inHandler:   make(chan struct{}),
		okToProceed: make(chan struct{}),
	}
}

func (s *Suite) TestRequestLimiter1(c *check.C) {
	h := newTestHandler()
	l := RequestLimiter{MaxConcurrent: 1, Handler: h}
	var wg sync.WaitGroup
	resps := make([]*httptest.ResponseRecorder, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		resps[i] = httptest.NewRecorder()
		go func(i int) {
			l.ServeHTTP(resps[i], &http.Request{})
			wg.Done()
		}(i)
	}
	done := make(chan struct{})
	go func() {
		// Make sure one request has entered the handler
		<-h.inHandler
		// Make sure all unsuccessful requests finish (but don't wait
		// for the one that's still waiting for okToProceed)
		wg.Add(-1)
		wg.Wait()
		// Wait for the last goroutine
		wg.Add(1)
		h.okToProceed <- struct{}{}
		wg.Wait()
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		c.Fatal("test timed out, probably deadlocked")
	}
	n200 := 0
	n503 := 0
	for i := 0; i < 10; i++ {
		switch resps[i].Code {
		case 200:
			n200++
		case 503:
			n503++
		default:
			c.Fatalf("Unexpected response code %d", resps[i].Code)
		}
	}
	if n200 != 1 || n503 != 9 {
		c.Fatalf("Got %d 200 responses, %d 503 responses (expected 1, 9)", n200, n503)
	}
	// Now that all 10 are finished, an 11th request should
	// succeed.
	go func() {
		<-h.inHandler
		h.okToProceed <- struct{}{}
	}()
	resp := httptest.NewRecorder()
	l.ServeHTTP(resp, &http.Request{})
	if resp.Code != 200 {
		c.Errorf("Got status %d on 11th request, want 200", resp.Code)
	}
}

func (*Suite) TestRequestLimiter10(c *check.C) {
	h := newTestHandler()
	l := RequestLimiter{MaxConcurrent: 10, Handler: h}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			l.ServeHTTP(httptest.NewRecorder(), &http.Request{})
			wg.Done()
		}()
		// Make sure the handler starts before we initiate the
		// next request, but don't let it finish yet.
		<-h.inHandler
	}
	for i := 0; i < 10; i++ {
		h.okToProceed <- struct{}{}
	}
	wg.Wait()
}

func (*Suite) TestRequestLimiterQueuePriority(c *check.C) {
	h := newTestHandler()
	rl := RequestLimiter{
		MaxConcurrent: 1000,
		MaxQueue:      200,
		Handler:       h,
		Priority: func(r *http.Request, _ time.Time) int64 {
			p, _ := strconv.ParseInt(r.Header.Get("Priority"), 10, 64)
			return p
		}}

	c.Logf("starting initial requests")
	for i := 0; i < rl.MaxConcurrent; i++ {
		go func() {
			rl.ServeHTTP(httptest.NewRecorder(), &http.Request{Header: http.Header{"No-Priority": {"x"}}})
		}()
	}
	c.Logf("waiting for initial requests to consume all MaxConcurrent slots")
	for i := 0; i < rl.MaxConcurrent; i++ {
		<-h.inHandler
	}

	c.Logf("starting %d priority=IneligibleForQueuePriority requests (should respond 503 immediately)", rl.MaxQueue)
	var wgX sync.WaitGroup
	for i := 0; i < rl.MaxQueue; i++ {
		wgX.Add(1)
		go func() {
			defer wgX.Done()
			resp := httptest.NewRecorder()
			rl.ServeHTTP(resp, &http.Request{Header: http.Header{"Priority": {fmt.Sprintf("%d", IneligibleForQueuePriority)}}})
			c.Check(resp.Code, check.Equals, http.StatusServiceUnavailable)
		}()
	}
	wgX.Wait()

	c.Logf("starting %d priority=1 and %d priority=1 requests", rl.MaxQueue, rl.MaxQueue)
	var wg1, wg2 sync.WaitGroup
	wg1.Add(rl.MaxQueue)
	wg2.Add(rl.MaxQueue)
	for i := 0; i < rl.MaxQueue*2; i++ {
		i := i
		go func() {
			pri := (i & 1) + 1
			resp := httptest.NewRecorder()
			rl.ServeHTTP(resp, &http.Request{Header: http.Header{"Priority": {fmt.Sprintf("%d", pri)}}})
			if pri == 1 {
				c.Check(resp.Code, check.Equals, http.StatusServiceUnavailable)
				wg1.Done()
			} else {
				c.Check(resp.Code, check.Equals, http.StatusOK)
				wg2.Done()
			}
		}()
	}

	c.Logf("waiting for queued priority=1 requests to fail")
	wg1.Wait()

	c.Logf("allowing initial requests to proceed")
	for i := 0; i < rl.MaxConcurrent; i++ {
		h.okToProceed <- struct{}{}
	}

	c.Logf("allowing queued priority=2 requests to proceed")
	for i := 0; i < rl.MaxQueue; i++ {
		<-h.inHandler
		h.okToProceed <- struct{}{}
	}
	c.Logf("waiting for queued priority=2 requests to succeed")
	wg2.Wait()
}
