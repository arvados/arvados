// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&MountsSuite{})

type MountsSuite struct {
	vm  VolumeManager
	rtr http.Handler
}

func (s *MountsSuite) SetUpTest(c *check.C) {
	s.vm = MakeTestVolumeManager(2)
	KeepVM = s.vm
	theConfig = DefaultConfig()
	theConfig.systemAuthToken = arvadostest.DataManagerToken
	theConfig.Start()
	s.rtr = MakeRESTRouter()
}

func (s *MountsSuite) TearDownTest(c *check.C) {
	s.vm.Close()
	KeepVM = nil
	theConfig = DefaultConfig()
	theConfig.Start()
}

func (s *MountsSuite) TestMounts(c *check.C) {
	vols := s.vm.AllWritable()
	vols[0].Put(context.Background(), TestHash, TestBlock)
	vols[1].Put(context.Background(), TestHash2, TestBlock2)

	resp := s.call("GET", "/mounts", "", nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var mntList []struct {
		UUID           string   `json:"uuid"`
		DeviceID       string   `json:"device_id"`
		ReadOnly       bool     `json:"read_only"`
		Replication    int      `json:"replication"`
		StorageClasses []string `json:"storage_classes"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &mntList)
	c.Assert(err, check.IsNil)
	c.Assert(len(mntList), check.Equals, 2)
	for _, m := range mntList {
		c.Check(len(m.UUID), check.Equals, 27)
		c.Check(m.UUID[:12], check.Equals, "zzzzz-ivpuk-")
		c.Check(m.DeviceID, check.Equals, "mock-device-id")
		c.Check(m.ReadOnly, check.Equals, false)
		c.Check(m.Replication, check.Equals, 1)
		c.Check(m.StorageClasses, check.DeepEquals, []string{"default"})
	}
	c.Check(mntList[0].UUID, check.Not(check.Equals), mntList[1].UUID)

	// Bad auth
	for _, tok := range []string{"", "xyzzy"} {
		resp = s.call("GET", "/mounts/"+mntList[1].UUID+"/blocks", tok, nil)
		c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
		c.Check(resp.Body.String(), check.Equals, "Unauthorized\n")
	}

	tok := arvadostest.DataManagerToken

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

func (s *MountsSuite) TestMetrics(c *check.C) {
	s.call("PUT", "/"+TestHash, "", TestBlock)
	s.call("PUT", "/"+TestHash2, "", TestBlock2)
	resp := s.call("GET", "/metrics.json", "", nil)
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
				SampleCount string  `json:"sample_count"`
				SampleSum   float64 `json:"sample_sum"`
				Quantile    []struct {
					Quantile float64
					Value    float64
				}
			}
		}
	}
	json.NewDecoder(resp.Body).Decode(&j)
	found := make(map[string]bool)
	for _, g := range j {
		for _, m := range g.Metric {
			if len(m.Label) == 2 && m.Label[0].Name == "code" && m.Label[0].Value == "200" && m.Label[1].Name == "method" && m.Label[1].Value == "put" {
				c.Check(m.Summary.SampleCount, check.Equals, "2")
				c.Check(len(m.Summary.Quantile), check.Not(check.Equals), 0)
				c.Check(m.Summary.Quantile[0].Value, check.Not(check.Equals), float64(0))
				found[g.Name] = true
			}
		}
	}
	c.Check(found["request_duration_seconds"], check.Equals, true)
	c.Check(found["time_to_status_seconds"], check.Equals, true)
}

func (s *MountsSuite) call(method, path, tok string, body []byte) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewReader(body))
	if tok != "" {
		req.Header.Set("Authorization", "OAuth2 "+tok)
	}
	s.rtr.ServeHTTP(resp, req)
	return resp
}
