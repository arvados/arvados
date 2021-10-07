// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

func (s *HandlerSuite) TestMounts(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	vols := s.handler.volmgr.AllWritable()
	vols[0].Put(context.Background(), TestHash, TestBlock)
	vols[1].Put(context.Background(), TestHash2, TestBlock2)

	resp := s.call("GET", "/mounts", "", nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var mntList []struct {
		UUID           string          `json:"uuid"`
		DeviceID       string          `json:"device_id"`
		ReadOnly       bool            `json:"read_only"`
		Replication    int             `json:"replication"`
		StorageClasses map[string]bool `json:"storage_classes"`
	}
	c.Log(resp.Body.String())
	err := json.Unmarshal(resp.Body.Bytes(), &mntList)
	c.Assert(err, check.IsNil)
	c.Assert(len(mntList), check.Equals, 2)
	for _, m := range mntList {
		c.Check(len(m.UUID), check.Equals, 27)
		c.Check(m.UUID[:12], check.Equals, "zzzzz-nyw5e-")
		c.Check(m.DeviceID, check.Equals, "mock-device-id")
		c.Check(m.ReadOnly, check.Equals, false)
		c.Check(m.Replication, check.Equals, 1)
		c.Check(m.StorageClasses, check.DeepEquals, map[string]bool{"default": true})
	}
	c.Check(mntList[0].UUID, check.Not(check.Equals), mntList[1].UUID)

	// Bad auth
	for _, tok := range []string{"", "xyzzy"} {
		resp = s.call("GET", "/mounts/"+mntList[1].UUID+"/blocks", tok, nil)
		c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
		c.Check(resp.Body.String(), check.Equals, "Unauthorized\n")
	}

	tok := arvadostest.SystemRootToken

	// Nonexistent mount UUID
	resp = s.call("GET", "/mounts/X/blocks", tok, nil)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	c.Check(resp.Body.String(), check.Equals, "mount not found\n")

	// Complete index of first mount
	resp = s.call("GET", "/mounts/"+mntList[0].UUID+"/blocks", tok, nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, TestHash+`\+[0-9]+ [0-9]+\n\n`)

	// Partial index of first mount (one block matches prefix)
	resp = s.call("GET", "/mounts/"+mntList[0].UUID+"/blocks?prefix="+TestHash[:2], tok, nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, TestHash+`\+[0-9]+ [0-9]+\n\n`)

	// Complete index of second mount (note trailing slash)
	resp = s.call("GET", "/mounts/"+mntList[1].UUID+"/blocks/", tok, nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, TestHash2+`\+[0-9]+ [0-9]+\n\n`)

	// Partial index of second mount (no blocks match prefix)
	resp = s.call("GET", "/mounts/"+mntList[1].UUID+"/blocks/?prefix="+TestHash[:2], tok, nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "\n")
}

func (s *HandlerSuite) TestMetrics(c *check.C) {
	reg := prometheus.NewRegistry()
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", reg, testServiceURL), check.IsNil)
	instrumented := httpserver.Instrument(reg, ctxlog.TestLogger(c), s.handler.Handler)
	s.handler.Handler = instrumented.ServeAPI(s.cluster.ManagementToken, instrumented)

	s.call("PUT", "/"+TestHash, "", TestBlock)
	s.call("PUT", "/"+TestHash2, "", TestBlock2)
	resp := s.call("GET", "/metrics.json", "", nil)
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
	resp = s.call("GET", "/metrics.json", "foobar", nil)
	c.Check(resp.Code, check.Equals, http.StatusForbidden)
	resp = s.call("GET", "/metrics.json", arvadostest.ManagementToken, nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
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
				c.Check(m.Summary.SampleCount, check.Equals, "2")
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
		c.Check(ok, check.Equals, true, check.Commentf("checking metric %q", m))
	}
}

func (s *HandlerSuite) call(method, path, tok string, body []byte) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	s.handler.ServeHTTP(resp, req)
	return resp
}
