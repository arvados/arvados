// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	. "gopkg.in/check.v1"
)

func (s *routerSuite) TestTrashList_Clear(c *C) {
	s.cluster.Collections.BlobTrash = false
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	resp := call(router, "PUT", "http://example/trash", s.cluster.SystemRootToken, []byte(`
		[
		 {
		  "locator":"acbd18db4cc2f85cedef654fccc4a4d8+3",
		  "block_mtime":1707249451308502672,
		  "mount_uuid":"zzzzz-nyw5e-000000000000000"
		 }
		]
		`), nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(router.trasher.todo, DeepEquals, []TrashListItem{{
		Locator:    "acbd18db4cc2f85cedef654fccc4a4d8+3",
		BlockMtime: 1707249451308502672,
		MountUUID:  "zzzzz-nyw5e-000000000000000",
	}})

	resp = call(router, "PUT", "http://example/trash", s.cluster.SystemRootToken, []byte("[]"), nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(router.trasher.todo, HasLen, 0)
}

func (s *routerSuite) TestTrashList_Execute(c *C) {
	s.cluster.Collections.BlobTrashConcurrency = 1
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "stub"},
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "stub"},
		"zzzzz-nyw5e-222222222222222": {Replication: 1, Driver: "stub", ReadOnly: true},
		"zzzzz-nyw5e-333333333333333": {Replication: 1, Driver: "stub", ReadOnly: true, AllowTrashWhenReadOnly: true},
	}
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	var mounts []struct {
		UUID     string
		DeviceID string `json:"device_id"`
	}
	resp := call(router, "GET", "http://example/mounts", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	err := json.Unmarshal(resp.Body.Bytes(), &mounts)
	c.Assert(err, IsNil)
	c.Assert(mounts, HasLen, 4)

	// Sort mounts by UUID
	sort.Slice(mounts, func(i, j int) bool {
		return mounts[i].UUID < mounts[j].UUID
	})

	// Make vols (stub volumes) in same order as mounts
	var vols []*stubVolume
	for _, mount := range mounts {
		vols = append(vols, router.keepstore.mounts[mount.UUID].volume.(*stubVolume))
	}

	// The "trial" loop below will construct the trashList which
	// we'll send to trasher via router, plus a slice of checks
	// which we'll run after the trasher has finished executing
	// the list.
	var trashList []TrashListItem
	var checks []func()

	tNew := time.Now().Add(-s.cluster.Collections.BlobSigningTTL.Duration() / 2)
	tOld := time.Now().Add(-s.cluster.Collections.BlobSigningTTL.Duration() - time.Second)

	for _, trial := range []struct {
		comment        string
		storeMtime     []time.Time
		trashListItems []TrashListItem
		expectData     []bool
	}{
		{
			comment:    "timestamp matches, but is not old enough to trash => skip",
			storeMtime: []time.Time{tNew},
			trashListItems: []TrashListItem{
				{
					BlockMtime: tNew.UnixNano(),
					MountUUID:  mounts[0].UUID,
				},
			},
			expectData: []bool{true},
		},
		{
			comment:    "timestamp matches, and is old enough => trash",
			storeMtime: []time.Time{tOld},
			trashListItems: []TrashListItem{
				{
					BlockMtime: tOld.UnixNano(),
					MountUUID:  mounts[0].UUID,
				},
			},
			expectData: []bool{false},
		},
		{
			comment:    "timestamp matches and is old enough on mount 0, but the request specifies mount 1, where timestamp does not match => skip",
			storeMtime: []time.Time{tOld, tOld.Add(-time.Second)},
			trashListItems: []TrashListItem{
				{
					BlockMtime: tOld.UnixNano(),
					MountUUID:  mounts[1].UUID,
				},
			},
			expectData: []bool{true, true},
		},
		{
			comment:    "MountUUID unspecified => trash from any mount where timestamp matches, leave alone elsewhere",
			storeMtime: []time.Time{tOld, tOld.Add(-time.Second)},
			trashListItems: []TrashListItem{
				{
					BlockMtime: tOld.UnixNano(),
				},
			},
			expectData: []bool{false, true},
		},
		{
			comment:    "MountUUID unspecified => trash from multiple mounts if timestamp matches, but skip readonly volumes unless AllowTrashWhenReadOnly",
			storeMtime: []time.Time{tOld, tOld, tOld, tOld},
			trashListItems: []TrashListItem{
				{
					BlockMtime: tOld.UnixNano(),
				},
			},
			expectData: []bool{false, false, true, false},
		},
		{
			comment:    "readonly MountUUID specified => skip",
			storeMtime: []time.Time{tOld, tOld, tOld},
			trashListItems: []TrashListItem{
				{
					BlockMtime: tOld.UnixNano(),
					MountUUID:  mounts[2].UUID,
				},
			},
			expectData: []bool{true, true, true},
		},
	} {
		trial := trial
		data := []byte(fmt.Sprintf("trial %+v", trial))
		hash := fmt.Sprintf("%x", md5.Sum(data))
		for i, t := range trial.storeMtime {
			if t.IsZero() {
				continue
			}
			err := vols[i].BlockWrite(context.Background(), hash, data)
			c.Assert(err, IsNil)
			err = vols[i].blockTouchWithTime(hash, t)
			c.Assert(err, IsNil)
		}
		for _, item := range trial.trashListItems {
			item.Locator = fmt.Sprintf("%s+%d", hash, len(data))
			trashList = append(trashList, item)
		}
		for i, expect := range trial.expectData {
			i, expect := i, expect
			checks = append(checks, func() {
				ent := vols[i].data[hash]
				dataPresent := ent.data != nil && ent.trash.IsZero()
				c.Check(dataPresent, Equals, expect, Commentf("%s mount %d (%s) expect present=%v but got len(ent.data)=%d ent.trash=%v // %s\nlog:\n%s", hash, i, vols[i].params.UUID, expect, len(ent.data), !ent.trash.IsZero(), trial.comment, vols[i].stubLog.String()))
			})
		}
	}

	listjson, err := json.Marshal(trashList)
	resp = call(router, "PUT", "http://example/trash", s.cluster.SystemRootToken, listjson, nil)
	c.Check(resp.Code, Equals, http.StatusOK)

	for {
		router.trasher.cond.L.Lock()
		todolen := len(router.trasher.todo)
		router.trasher.cond.L.Unlock()
		if todolen == 0 && router.trasher.inprogress.Load() == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	for _, check := range checks {
		check()
	}
}
