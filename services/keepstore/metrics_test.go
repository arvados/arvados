// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"context"
	"encoding/json"
	"net/http"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/prometheus/client_golang/prometheus"
	. "gopkg.in/check.v1"
)

func (s *routerSuite) TestMetrics(c *C) {
	reg := prometheus.NewRegistry()
	router, cancel := testRouter(c, s.cluster, reg)
	defer cancel()
	instrumented := httpserver.Instrument(reg, ctxlog.TestLogger(c), router)
	handler := instrumented.ServeAPI(s.cluster.ManagementToken, instrumented)

	router.keepstore.BlockWrite(context.Background(), arvados.BlockWriteOptions{
		Hash: fooHash,
		Data: []byte("foo"),
	})
	router.keepstore.BlockWrite(context.Background(), arvados.BlockWriteOptions{
		Hash: barHash,
		Data: []byte("bar"),
	})

	// prime the metrics by doing a no-op request
	resp := call(handler, "GET", "/", "", nil, nil)

	resp = call(handler, "GET", "/metrics.json", "", nil, nil)
	c.Check(resp.Code, Equals, http.StatusUnauthorized)
	resp = call(handler, "GET", "/metrics.json", "foobar", nil, nil)
	c.Check(resp.Code, Equals, http.StatusForbidden)
	resp = call(handler, "GET", "/metrics.json", arvadostest.ManagementToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	var j []struct {
		Name   string
		Help   string
		Type   string
		Metric []struct {
			Label []struct {
				Name  string
				Value string
			}
			Summary struct {
				SampleCount string
				SampleSum   float64
			}
		}
	}
	json.NewDecoder(resp.Body).Decode(&j)
	found := make(map[string]bool)
	names := map[string]bool{}
	for _, g := range j {
		names[g.Name] = true
		for _, m := range g.Metric {
			if len(m.Label) == 2 && m.Label[0].Name == "code" && m.Label[0].Value == "200" && m.Label[1].Name == "method" && m.Label[1].Value == "put" {
				c.Check(m.Summary.SampleCount, Equals, "2")
				found[g.Name] = true
			}
		}
	}

	metricsNames := []string{
		"arvados_keepstore_bufferpool_inuse_buffers",
		"arvados_keepstore_bufferpool_max_buffers",
		"arvados_keepstore_bufferpool_allocated_bytes",
		"arvados_keepstore_pull_queue_inprogress_entries",
		"arvados_keepstore_pull_queue_pending_entries",
		"arvados_keepstore_trash_queue_inprogress_entries",
		"arvados_keepstore_trash_queue_pending_entries",
		"request_duration_seconds",
	}
	for _, m := range metricsNames {
		_, ok := names[m]
		c.Check(ok, Equals, true, Commentf("checking metric %q", m))
	}
}
