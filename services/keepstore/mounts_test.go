// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"context"
	"encoding/json"
	"net/http"

	. "gopkg.in/check.v1"
)

func (s *routerSuite) TestMounts(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	router.keepstore.mountsW[0].BlockWrite(context.Background(), fooHash, []byte("foo"))
	router.keepstore.mountsW[1].BlockWrite(context.Background(), barHash, []byte("bar"))

	resp := call(router, "GET", "/mounts", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Log(resp.Body.String())

	var mntList []struct {
		UUID           string          `json:"uuid"`
		DeviceID       string          `json:"device_id"`
		ReadOnly       bool            `json:"read_only"`
		Replication    int             `json:"replication"`
		StorageClasses map[string]bool `json:"storage_classes"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &mntList)
	c.Assert(err, IsNil)
	c.Assert(mntList, HasLen, 2)

	for _, m := range mntList {
		c.Check(len(m.UUID), Equals, 27)
		c.Check(m.UUID[:12], Equals, "zzzzz-nyw5e-")
		c.Check(m.DeviceID, Matches, "0x[0-9a-f]+")
		c.Check(m.ReadOnly, Equals, false)
		c.Check(m.Replication, Equals, 1)
		c.Check(m.StorageClasses, HasLen, 1)
		for k := range m.StorageClasses {
			c.Check(k, Matches, "testclass.*")
		}
	}
	c.Check(mntList[0].UUID, Not(Equals), mntList[1].UUID)

	c.Logf("=== bad auth")
	for _, tok := range []string{"", "xyzzy"} {
		resp = call(router, "GET", "/mounts/"+mntList[1].UUID+"/blocks", tok, nil, nil)
		if tok == "" {
			c.Check(resp.Code, Equals, http.StatusUnauthorized)
			c.Check(resp.Body.String(), Equals, "Unauthorized\n")
		} else {
			c.Check(resp.Code, Equals, http.StatusForbidden)
			c.Check(resp.Body.String(), Equals, "Forbidden\n")
		}
	}

	c.Logf("=== nonexistent mount UUID")
	resp = call(router, "GET", "/mounts/X/blocks", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusNotFound)

	c.Logf("=== complete index of first mount")
	resp = call(router, "GET", "/mounts/"+mntList[0].UUID+"/blocks", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Matches, fooHash+`\+[0-9]+ [0-9]+\n\n`)

	c.Logf("=== partial index of first mount (one block matches prefix)")
	resp = call(router, "GET", "/mounts/"+mntList[0].UUID+"/blocks?prefix="+fooHash[:2], s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Matches, fooHash+`\+[0-9]+ [0-9]+\n\n`)

	c.Logf("=== complete index of second mount (note trailing slash)")
	resp = call(router, "GET", "/mounts/"+mntList[1].UUID+"/blocks/", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Matches, barHash+`\+[0-9]+ [0-9]+\n\n`)

	c.Logf("=== partial index of second mount (no blocks match prefix)")
	resp = call(router, "GET", "/mounts/"+mntList[1].UUID+"/blocks/?prefix="+fooHash[:2], s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Equals, "\n")
}
