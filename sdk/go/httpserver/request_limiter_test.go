// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
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

func TestRequestLimiter1(t *testing.T) {
	h := newTestHandler()
	l := NewRequestLimiter(1, h, nil)
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
		t.Fatal("test timed out, probably deadlocked")
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
			t.Fatalf("Unexpected response code %d", resps[i].Code)
		}
	}
	if n200 != 1 || n503 != 9 {
		t.Fatalf("Got %d 200 responses, %d 503 responses (expected 1, 9)", n200, n503)
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
		t.Errorf("Got status %d on 11th request, want 200", resp.Code)
	}
}

func TestRequestLimiter10(t *testing.T) {
	h := newTestHandler()
	l := NewRequestLimiter(10, h, nil)
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
