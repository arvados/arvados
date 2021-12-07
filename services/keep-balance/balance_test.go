// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

// Test with Gocheck
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&balancerSuite{})

type balancerSuite struct {
	Balancer
	srvs            []*KeepService
	blks            map[string]tester
	knownRendezvous [][]int
	signatureTTL    int64
}

const (
	// index into knownRendezvous
	known0 = 0
)

type slots []int

type tester struct {
	known       int
	desired     map[string]int
	current     slots
	timestamps  []int64
	shouldPull  slots
	shouldTrash slots

	shouldPullMounts  []string
	shouldTrashMounts []string

	expectBlockState *balancedBlockState
	expectClassState map[string]balancedBlockState
}

func (bal *balancerSuite) SetUpSuite(c *check.C) {
	bal.knownRendezvous = nil
	for _, str := range []string{
		"3eab2d5fc9681074",
		"097dba52e648f1c3",
		"c5b4e023f8a7d691",
		"9d81c02e76a3bf54",
	} {
		var slots []int
		for _, c := range []byte(str) {
			pos, _ := strconv.ParseUint(string(c), 16, 4)
			slots = append(slots, int(pos))
		}
		bal.knownRendezvous = append(bal.knownRendezvous, slots)
	}

	bal.signatureTTL = 3600
	bal.Logger = ctxlog.TestLogger(c)
}

func (bal *balancerSuite) SetUpTest(c *check.C) {
	bal.srvs = make([]*KeepService, 16)
	bal.KeepServices = make(map[string]*KeepService)
	for i := range bal.srvs {
		srv := &KeepService{
			KeepService: arvados.KeepService{
				UUID: fmt.Sprintf("zzzzz-bi6l4-%015x", i),
			},
		}
		srv.mounts = []*KeepMount{{
			KeepMount: arvados.KeepMount{
				UUID:           fmt.Sprintf("zzzzz-mount-%015x", i),
				StorageClasses: map[string]bool{"default": true},
			},
			KeepService: srv,
		}}
		bal.srvs[i] = srv
		bal.KeepServices[srv.UUID] = srv
	}

	bal.MinMtime = time.Now().UnixNano() - bal.signatureTTL*1e9
	bal.cleanupMounts()
}

func (bal *balancerSuite) TestPerfect(c *check.C) {
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{0, 1},
		shouldPull:  nil,
		shouldTrash: nil,
		expectBlockState: &balancedBlockState{
			needed: 2,
		}})
}

func (bal *balancerSuite) TestDecreaseRepl(c *check.C) {
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{0, 2, 1},
		shouldTrash: slots{2},
		expectBlockState: &balancedBlockState{
			needed:   2,
			unneeded: 1,
		}})
}

func (bal *balancerSuite) TestDecreaseReplToZero(c *check.C) {
	bal.try(c, tester{
		desired:     map[string]int{"default": 0},
		current:     slots{0, 1, 3},
		shouldTrash: slots{0, 1, 3},
		expectBlockState: &balancedBlockState{
			unneeded: 3,
		}})
}

func (bal *balancerSuite) TestIncreaseRepl(c *check.C) {
	bal.try(c, tester{
		desired:    map[string]int{"default": 4},
		current:    slots{0, 1},
		shouldPull: slots{2, 3},
		expectBlockState: &balancedBlockState{
			needed:  2,
			pulling: 2,
		}})
}

func (bal *balancerSuite) TestSkipReadonly(c *check.C) {
	bal.srvList(0, slots{3})[0].ReadOnly = true
	bal.try(c, tester{
		desired:    map[string]int{"default": 4},
		current:    slots{0, 1},
		shouldPull: slots{2, 4},
		expectBlockState: &balancedBlockState{
			needed:  2,
			pulling: 2,
		}})
}

func (bal *balancerSuite) TestMultipleViewsReadOnly(c *check.C) {
	bal.testMultipleViews(c, true)
}

func (bal *balancerSuite) TestMultipleViews(c *check.C) {
	bal.testMultipleViews(c, false)
}

func (bal *balancerSuite) testMultipleViews(c *check.C, readonly bool) {
	for i, srv := range bal.srvs {
		// Add a mount to each service
		srv.mounts[0].KeepMount.DeviceID = fmt.Sprintf("writable-by-srv-%x", i)
		srv.mounts = append(srv.mounts, &KeepMount{
			KeepMount: arvados.KeepMount{
				DeviceID:       bal.srvs[(i+1)%len(bal.srvs)].mounts[0].KeepMount.DeviceID,
				UUID:           bal.srvs[(i+1)%len(bal.srvs)].mounts[0].KeepMount.UUID,
				ReadOnly:       readonly,
				Replication:    1,
				StorageClasses: map[string]bool{"default": true},
			},
			KeepService: srv,
		})
	}
	for i := 1; i < len(bal.srvs); i++ {
		c.Logf("i=%d", i)
		if i == 4 {
			// Timestamps are all different, but one of
			// the mounts on srv[4] has the same device ID
			// where the non-deletable replica is stored
			// on srv[3], so only one replica is safe to
			// trash.
			bal.try(c, tester{
				desired:     map[string]int{"default": 1},
				current:     slots{0, i, i},
				shouldTrash: slots{i}})
		} else if readonly {
			// Timestamps are all different, and the third
			// replica can't be trashed because it's on a
			// read-only mount, so the first two replicas
			// should be trashed.
			bal.try(c, tester{
				desired:     map[string]int{"default": 1},
				current:     slots{0, i, i},
				shouldTrash: slots{0, i}})
		} else {
			// Timestamps are all different, so both
			// replicas on the non-optimal server should
			// be trashed.
			bal.try(c, tester{
				desired:     map[string]int{"default": 1},
				current:     slots{0, i, i},
				shouldTrash: slots{i, i}})
		}
		// If the three replicas have identical timestamps,
		// none of them can be trashed safely.
		bal.try(c, tester{
			desired:    map[string]int{"default": 1},
			current:    slots{0, i, i},
			timestamps: []int64{12345678, 12345678, 12345678}})
		// If the first and third replicas have identical
		// timestamps, only the second replica should be
		// trashed.
		bal.try(c, tester{
			desired:     map[string]int{"default": 1},
			current:     slots{0, i, i},
			timestamps:  []int64{12345678, 12345679, 12345678},
			shouldTrash: slots{i}})
	}
}

func (bal *balancerSuite) TestFixUnbalanced(c *check.C) {
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{2, 0},
		shouldPull: slots{1}})
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{2, 7},
		shouldPull: slots{0, 1}})
	// if only one of the pulls succeeds, we'll see this next:
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{2, 1, 7},
		shouldPull:  slots{0},
		shouldTrash: slots{7}})
	// if both pulls succeed, we'll see this next:
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{2, 0, 1, 7},
		shouldTrash: slots{2, 7}})

	// unbalanced + excessive replication => pull + trash
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{2, 5, 7},
		shouldPull:  slots{0, 1},
		shouldTrash: slots{7}})
}

func (bal *balancerSuite) TestMultipleReplicasPerService(c *check.C) {
	for s, srv := range bal.srvs {
		for i := 0; i < 3; i++ {
			m := *(srv.mounts[0])
			m.UUID = fmt.Sprintf("zzzzz-mount-%015x", (s<<10)+i)
			srv.mounts = append(srv.mounts, &m)
		}
	}
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{0, 0},
		shouldPull: slots{1}})
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{2, 2},
		shouldPull: slots{0, 1}})
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{0, 0, 1},
		shouldTrash: slots{0}})
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{1, 1, 0},
		shouldTrash: slots{1}})
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{1, 0, 1, 0, 2},
		shouldTrash: slots{0, 1, 2}})
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{1, 1, 1, 0, 2},
		shouldTrash: slots{1, 1, 2}})
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{1, 1, 2},
		shouldPull:  slots{0},
		shouldTrash: slots{1}})
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{1, 1, 0},
		timestamps:  []int64{12345678, 12345678, 12345679},
		shouldTrash: nil})
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{1, 1},
		shouldPull: slots{0}})
}

func (bal *balancerSuite) TestIncreaseReplTimestampCollision(c *check.C) {
	// For purposes of increasing replication, we assume identical
	// replicas are distinct.
	bal.try(c, tester{
		desired:    map[string]int{"default": 4},
		current:    slots{0, 1},
		timestamps: []int64{12345678, 12345678},
		shouldPull: slots{2, 3}})
}

func (bal *balancerSuite) TestDecreaseReplTimestampCollision(c *check.C) {
	// For purposes of decreasing replication, we assume identical
	// replicas are NOT distinct.
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{0, 1, 2},
		timestamps: []int64{12345678, 12345678, 12345678}})
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{0, 1, 2},
		timestamps: []int64{12345678, 10000000, 10000000}})
}

func (bal *balancerSuite) TestDecreaseReplBlockTooNew(c *check.C) {
	oldTime := bal.MinMtime - 3600
	newTime := bal.MinMtime + 3600
	// The excess replica is too new to delete.
	bal.try(c, tester{
		desired:    map[string]int{"default": 2},
		current:    slots{0, 1, 2},
		timestamps: []int64{oldTime, newTime, newTime + 1},
		expectBlockState: &balancedBlockState{
			needed:   2,
			unneeded: 1,
		}})
	// The best replicas are too new to delete, but the excess
	// replica is old enough.
	bal.try(c, tester{
		desired:     map[string]int{"default": 2},
		current:     slots{0, 1, 2},
		timestamps:  []int64{newTime, newTime + 1, oldTime},
		shouldTrash: slots{2}})
}

func (bal *balancerSuite) TestCleanupMounts(c *check.C) {
	bal.srvs[3].mounts[0].KeepMount.ReadOnly = true
	bal.srvs[3].mounts[0].KeepMount.DeviceID = "abcdef"
	bal.srvs[14].mounts[0].KeepMount.UUID = bal.srvs[3].mounts[0].KeepMount.UUID
	bal.srvs[14].mounts[0].KeepMount.DeviceID = "abcdef"
	c.Check(len(bal.srvs[3].mounts), check.Equals, 1)
	bal.cleanupMounts()
	c.Check(len(bal.srvs[3].mounts), check.Equals, 0)
	bal.try(c, tester{
		known:      0,
		desired:    map[string]int{"default": 2},
		current:    slots{1},
		shouldPull: slots{2}})
}

func (bal *balancerSuite) TestVolumeReplication(c *check.C) {
	bal.srvs[0].mounts[0].KeepMount.Replication = 2  // srv 0
	bal.srvs[14].mounts[0].KeepMount.Replication = 2 // srv e
	bal.cleanupMounts()
	// block 0 rendezvous is 3,e,a -- so slot 1 has repl=2
	bal.try(c, tester{
		known:      0,
		desired:    map[string]int{"default": 2},
		current:    slots{1},
		shouldPull: slots{0},
		expectBlockState: &balancedBlockState{
			needed:  1,
			pulling: 1,
		}})
	bal.try(c, tester{
		known:      0,
		desired:    map[string]int{"default": 2},
		current:    slots{0, 1},
		shouldPull: nil,
		expectBlockState: &balancedBlockState{
			needed: 2,
		}})
	bal.try(c, tester{
		known:       0,
		desired:     map[string]int{"default": 2},
		current:     slots{0, 1, 2},
		shouldTrash: slots{2},
		expectBlockState: &balancedBlockState{
			needed:   2,
			unneeded: 1,
		}})
	bal.try(c, tester{
		known:       0,
		desired:     map[string]int{"default": 3},
		current:     slots{0, 2, 3, 4},
		shouldPull:  slots{1},
		shouldTrash: slots{4},
		expectBlockState: &balancedBlockState{
			needed:   3,
			unneeded: 1,
			pulling:  1,
		}})
	bal.try(c, tester{
		known:       0,
		desired:     map[string]int{"default": 3},
		current:     slots{0, 1, 2, 3, 4},
		shouldTrash: slots{2, 3, 4},
		expectBlockState: &balancedBlockState{
			needed:   2,
			unneeded: 3,
		}})
	bal.try(c, tester{
		known:       0,
		desired:     map[string]int{"default": 4},
		current:     slots{0, 1, 2, 3, 4},
		shouldTrash: slots{3, 4},
		expectBlockState: &balancedBlockState{
			needed:   3,
			unneeded: 2,
		}})
	// block 1 rendezvous is 0,9,7 -- so slot 0 has repl=2
	bal.try(c, tester{
		known:   1,
		desired: map[string]int{"default": 2},
		current: slots{0},
		expectBlockState: &balancedBlockState{
			needed: 1,
		}})
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"default": 3},
		current:    slots{0},
		shouldPull: slots{1},
		expectBlockState: &balancedBlockState{
			needed:  1,
			pulling: 1,
		}})
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"default": 4},
		current:    slots{0},
		shouldPull: slots{1, 2},
		expectBlockState: &balancedBlockState{
			needed:  1,
			pulling: 2,
		}})
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"default": 4},
		current:    slots{2},
		shouldPull: slots{0, 1},
		expectBlockState: &balancedBlockState{
			needed:  1,
			pulling: 2,
		}})
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"default": 4},
		current:    slots{7},
		shouldPull: slots{0, 1, 2},
		expectBlockState: &balancedBlockState{
			needed:  1,
			pulling: 3,
		}})
	bal.try(c, tester{
		known:       1,
		desired:     map[string]int{"default": 2},
		current:     slots{1, 2, 3, 4},
		shouldPull:  slots{0},
		shouldTrash: slots{3, 4},
		expectBlockState: &balancedBlockState{
			needed:   2,
			unneeded: 2,
			pulling:  1,
		}})
	bal.try(c, tester{
		known:       1,
		desired:     map[string]int{"default": 2},
		current:     slots{0, 1, 2},
		shouldTrash: slots{1, 2},
		expectBlockState: &balancedBlockState{
			needed:   1,
			unneeded: 2,
		}})
}

func (bal *balancerSuite) TestDeviceRWMountedByMultipleServers(c *check.C) {
	dupUUID := bal.srvs[0].mounts[0].KeepMount.UUID
	bal.srvs[9].mounts[0].KeepMount.UUID = dupUUID
	bal.srvs[14].mounts[0].KeepMount.UUID = dupUUID
	// block 0 belongs on servers 3 and e, which have different
	// UUIDs.
	bal.try(c, tester{
		known:      0,
		desired:    map[string]int{"default": 2},
		current:    slots{1},
		shouldPull: slots{0}})
	// block 1 belongs on servers 0 and 9, which both report
	// having a replica, but the replicas are on the same volume
	// -- so we should pull to the third position (7).
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"default": 2},
		current:    slots{0, 1},
		shouldPull: slots{2}})
	// block 1 can be pulled to the doubly-mounted volume, but the
	// pull should only be done on the first of the two servers.
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"default": 2},
		current:    slots{2},
		shouldPull: slots{0}})
	// block 0 has one replica on a single volume mounted on two
	// servers (e,9 at positions 1,9). Trashing the replica on 9
	// would lose the block.
	bal.try(c, tester{
		known:      0,
		desired:    map[string]int{"default": 2},
		current:    slots{1, 9},
		shouldPull: slots{0},
		expectBlockState: &balancedBlockState{
			needed:  1,
			pulling: 1,
		}})
	// block 0 is overreplicated, but the second and third
	// replicas are the same replica according to volume UUID
	// (despite different Mtimes). Don't trash the third replica.
	bal.try(c, tester{
		known:   0,
		desired: map[string]int{"default": 2},
		current: slots{0, 1, 9},
		expectBlockState: &balancedBlockState{
			needed: 2,
		}})
	// block 0 is overreplicated; the third and fifth replicas are
	// extra, but the fourth is another view of the second and
	// shouldn't be trashed.
	bal.try(c, tester{
		known:       0,
		desired:     map[string]int{"default": 2},
		current:     slots{0, 1, 5, 9, 12},
		shouldTrash: slots{5, 12},
		expectBlockState: &balancedBlockState{
			needed:   2,
			unneeded: 2,
		}})
}

func (bal *balancerSuite) TestChangeStorageClasses(c *check.C) {
	// For known blocks 0/1/2/3, server 9 is slot 9/1/14/0 in
	// probe order. For these tests we give it two mounts, one
	// with classes=[special], one with
	// classes=[special,special2].
	bal.srvs[9].mounts = []*KeepMount{{
		KeepMount: arvados.KeepMount{
			Replication:    1,
			StorageClasses: map[string]bool{"special": true},
			UUID:           "zzzzz-mount-special00000009",
			DeviceID:       "9-special",
		},
		KeepService: bal.srvs[9],
	}, {
		KeepMount: arvados.KeepMount{
			Replication:    1,
			StorageClasses: map[string]bool{"special": true, "special2": true},
			UUID:           "zzzzz-mount-special20000009",
			DeviceID:       "9-special-and-special2",
		},
		KeepService: bal.srvs[9],
	}}
	// For known blocks 0/1/2/3, server 13 (d) is slot 5/3/11/1 in
	// probe order. We give it two mounts, one with
	// classes=[special3], one with classes=[default].
	bal.srvs[13].mounts = []*KeepMount{{
		KeepMount: arvados.KeepMount{
			Replication:    1,
			StorageClasses: map[string]bool{"special2": true},
			UUID:           "zzzzz-mount-special2000000d",
			DeviceID:       "13-special2",
		},
		KeepService: bal.srvs[13],
	}, {
		KeepMount: arvados.KeepMount{
			Replication:    1,
			StorageClasses: map[string]bool{"default": true},
			UUID:           "zzzzz-mount-00000000000000d",
			DeviceID:       "13-default",
		},
		KeepService: bal.srvs[13],
	}}
	// Pull to slot 9 because that's the only server with the
	// desired class "special".
	bal.try(c, tester{
		known:            0,
		desired:          map[string]int{"default": 2, "special": 1},
		current:          slots{0, 1},
		shouldPull:       slots{9},
		shouldPullMounts: []string{"zzzzz-mount-special20000009"}})
	// If some storage classes are not satisfied, don't trash any
	// excess replicas. (E.g., if someone desires repl=1 on
	// class=durable, and we have two copies on class=volatile, we
	// should wait for pull to succeed before trashing anything).
	bal.try(c, tester{
		known:            0,
		desired:          map[string]int{"special": 1},
		current:          slots{0, 1},
		shouldPull:       slots{9},
		shouldPullMounts: []string{"zzzzz-mount-special20000009"}})
	// Once storage classes are satisfied, trash excess replicas
	// that appear earlier in probe order but aren't needed to
	// satisfy the desired classes.
	bal.try(c, tester{
		known:       0,
		desired:     map[string]int{"special": 1},
		current:     slots{0, 1, 9},
		shouldTrash: slots{0, 1}})
	// Pull to slot 5, the best server with class "special2".
	bal.try(c, tester{
		known:            0,
		desired:          map[string]int{"special2": 1},
		current:          slots{0, 1},
		shouldPull:       slots{5},
		shouldPullMounts: []string{"zzzzz-mount-special2000000d"}})
	// Pull to slot 5 and 9 to get replication 2 in desired class
	// "special2".
	bal.try(c, tester{
		known:            0,
		desired:          map[string]int{"special2": 2},
		current:          slots{0, 1},
		shouldPull:       slots{5, 9},
		shouldPullMounts: []string{"zzzzz-mount-special20000009", "zzzzz-mount-special2000000d"}})
	// Slot 0 has a replica in "default", slot 1 has a replica
	// in "special"; we need another replica in "default", i.e.,
	// on slot 2.
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"default": 2, "special": 1},
		current:    slots{0, 1},
		shouldPull: slots{2}})
	// Pull to best probe position 0 (despite wrong storage class)
	// if it's impossible to achieve desired replication in the
	// desired class (only slots 1 and 3 have special2).
	bal.try(c, tester{
		known:      1,
		desired:    map[string]int{"special2": 3},
		current:    slots{3},
		shouldPull: slots{0, 1}})
	// Trash excess replica.
	bal.try(c, tester{
		known:       3,
		desired:     map[string]int{"special": 1},
		current:     slots{0, 1},
		shouldTrash: slots{1}})
	// Leave one copy on slot 1 because slot 0 (server 9) only
	// gives us repl=1.
	bal.try(c, tester{
		known:   3,
		desired: map[string]int{"special": 2},
		current: slots{0, 1}})
}

// Clear all servers' changesets, balance a single block, and verify
// the appropriate changes for that block have been added to the
// changesets.
func (bal *balancerSuite) try(c *check.C, t tester) {
	bal.setupLookupTables()
	blk := &BlockState{
		Replicas: bal.replList(t.known, t.current),
		Desired:  t.desired,
	}
	for i, t := range t.timestamps {
		blk.Replicas[i].Mtime = t
	}
	for _, srv := range bal.srvs {
		srv.ChangeSet = &ChangeSet{}
	}
	result := bal.balanceBlock(knownBlkid(t.known), blk)

	var didPull, didTrash slots
	var didPullMounts, didTrashMounts []string
	for i, srv := range bal.srvs {
		var slot int
		for probeOrder, srvNum := range bal.knownRendezvous[t.known] {
			if srvNum == i {
				slot = probeOrder
			}
		}
		for _, pull := range srv.Pulls {
			didPull = append(didPull, slot)
			didPullMounts = append(didPullMounts, pull.To.UUID)
			c.Check(pull.SizedDigest, check.Equals, knownBlkid(t.known))
		}
		for _, trash := range srv.Trashes {
			didTrash = append(didTrash, slot)
			didTrashMounts = append(didTrashMounts, trash.From.UUID)
			c.Check(trash.SizedDigest, check.Equals, knownBlkid(t.known))
		}
	}

	for _, list := range []slots{didPull, didTrash, t.shouldPull, t.shouldTrash} {
		sort.Sort(sort.IntSlice(list))
	}
	c.Check(didPull, check.DeepEquals, t.shouldPull)
	c.Check(didTrash, check.DeepEquals, t.shouldTrash)
	if t.shouldPullMounts != nil {
		sort.Strings(didPullMounts)
		c.Check(didPullMounts, check.DeepEquals, t.shouldPullMounts)
	}
	if t.shouldTrashMounts != nil {
		sort.Strings(didTrashMounts)
		c.Check(didTrashMounts, check.DeepEquals, t.shouldTrashMounts)
	}
	if t.expectBlockState != nil {
		c.Check(result.blockState, check.Equals, *t.expectBlockState)
	}
	if t.expectClassState != nil {
		c.Check(result.classState, check.DeepEquals, t.expectClassState)
	}
}

// srvList returns the KeepServices, sorted in rendezvous order and
// then selected by idx. For example, srvList(3, slots{0, 1, 4})
// returns the first-, second-, and fifth-best servers for storing
// bal.knownBlkid(3).
func (bal *balancerSuite) srvList(knownBlockID int, order slots) (srvs []*KeepService) {
	for _, i := range order {
		srvs = append(srvs, bal.srvs[bal.knownRendezvous[knownBlockID][i]])
	}
	return
}

// replList is like srvList but returns an "existing replicas" slice,
// suitable for a BlockState test fixture.
func (bal *balancerSuite) replList(knownBlockID int, order slots) (repls []Replica) {
	nextMnt := map[*KeepService]int{}
	mtime := time.Now().UnixNano() - (bal.signatureTTL+86400)*1e9
	for _, srv := range bal.srvList(knownBlockID, order) {
		// round-robin repls onto each srv's mounts
		n := nextMnt[srv]
		nextMnt[srv] = (n + 1) % len(srv.mounts)

		repls = append(repls, Replica{srv.mounts[n], mtime})
		mtime++
	}
	return
}

// generate the same data hashes that are tested in
// sdk/go/keepclient/root_sorter_test.go
func knownBlkid(i int) arvados.SizedDigest {
	return arvados.SizedDigest(fmt.Sprintf("%x+64", md5.Sum([]byte(fmt.Sprintf("%064x", i)))))
}
