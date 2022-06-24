// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	check "gopkg.in/check.v1"
)

func (s *Suite) TestInspect(c *check.C) {
	reg := prometheus.NewRegistry()
	h := newTestHandler()
	mh := Inspect(reg, "abcd", h)
	handlerReturned := make(chan struct{})
	reqctx, reqcancel := context.WithCancel(context.Background())
	longreq := httptest.NewRequest("GET", "/test", nil).WithContext(reqctx)
	go func() {
		mh.ServeHTTP(httptest.NewRecorder(), longreq)
		close(handlerReturned)
	}()
	<-h.inHandler

	resp := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_inspect/requests", nil)
	mh.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
	c.Check(resp.Body.String(), check.Equals, `{"errors":["unauthorized"]}`+"\n")

	resp = httptest.NewRecorder()
	req.Header.Set("Authorization", "Bearer abcde")
	mh.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)

	resp = httptest.NewRecorder()
	req.Header.Set("Authorization", "Bearer abcd")
	mh.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	reqs := []map[string]interface{}{}
	err := json.NewDecoder(resp.Body).Decode(&reqs)
	c.Check(err, check.IsNil)
	c.Check(reqs, check.HasLen, 1)
	c.Check(reqs[0]["URL"], check.Equals, "/test")

	// Request is active, so we should see active request age > 0
	resp = httptest.NewRecorder()
	mreq := httptest.NewRequest("GET", "/metrics", nil)
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(resp, mreq)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*\narvados_max_active_request_age_seconds [0\.]*[1-9][-\d\.e]*\n.*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*\narvados_max_abandoned_request_age_seconds 0\n.*`)

	reqcancel()

	// Request context is canceled but handler hasn't returned, so
	// we should see max abandoned request age > 0
	resp = httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(resp, mreq)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*\narvados_max_active_request_age_seconds 0\n.*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*\narvados_max_abandoned_request_age_seconds [0\.]*[1-9][-\d\.e]*\n.*`)

	h.okToProceed <- struct{}{}
	<-handlerReturned

	// Handler has returned, so we should see max abandoned
	// request age == max active request age == 0
	resp = httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(resp, mreq)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*\narvados_max_active_request_age_seconds 0\n.*`)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*\narvados_max_abandoned_request_age_seconds 0\n.*`)

	// ...and no active requests at the /_monitor endpoint
	resp = httptest.NewRecorder()
	mh.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	reqs = nil
	err = json.NewDecoder(resp.Body).Decode(&reqs)
	c.Check(err, check.IsNil)
	c.Assert(reqs, check.HasLen, 0)
}
