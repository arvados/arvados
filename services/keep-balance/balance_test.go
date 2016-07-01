package main

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"

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
	desired     int
	current     slots
	timestamps  []int64
	shouldPull  slots
	shouldTrash slots
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
		bal.srvs[i] = srv
		bal.KeepServices[srv.UUID] = srv
	}

	bal.MinMtime = time.Now().UnixNano() - bal.signatureTTL*1e9
}

func (bal *balancerSuite) TestPerfect(c *check.C) {
	bal.try(c, tester{
		desired:     2,
		current:     slots{0, 1},
		shouldPull:  nil,
		shouldTrash: nil})
}

func (bal *balancerSuite) TestDecreaseRepl(c *check.C) {
	bal.try(c, tester{
		desired:     2,
		current:     slots{0, 2, 1},
		shouldTrash: slots{2}})
}

func (bal *balancerSuite) TestDecreaseReplToZero(c *check.C) {
	bal.try(c, tester{
		desired:     0,
		current:     slots{0, 1, 3},
		shouldTrash: slots{0, 1, 3}})
}

func (bal *balancerSuite) TestIncreaseRepl(c *check.C) {
	bal.try(c, tester{
		desired:    4,
		current:    slots{0, 1},
		shouldPull: slots{2, 3}})
}

func (bal *balancerSuite) TestSkipReadonly(c *check.C) {
	bal.srvList(0, slots{3})[0].ReadOnly = true
	bal.try(c, tester{
		desired:    4,
		current:    slots{0, 1},
		shouldPull: slots{2, 4}})
}

func (bal *balancerSuite) TestFixUnbalanced(c *check.C) {
	bal.try(c, tester{
		desired:    2,
		current:    slots{2, 0},
		shouldPull: slots{1}})
	bal.try(c, tester{
		desired:    2,
		current:    slots{2, 7},
		shouldPull: slots{0, 1}})
	// if only one of the pulls succeeds, we'll see this next:
	bal.try(c, tester{
		desired:     2,
		current:     slots{2, 1, 7},
		shouldPull:  slots{0},
		shouldTrash: slots{7}})
	// if both pulls succeed, we'll see this next:
	bal.try(c, tester{
		desired:     2,
		current:     slots{2, 0, 1, 7},
		shouldTrash: slots{2, 7}})

	// unbalanced + excessive replication => pull + trash
	bal.try(c, tester{
		desired:     2,
		current:     slots{2, 5, 7},
		shouldPull:  slots{0, 1},
		shouldTrash: slots{7}})
}

func (bal *balancerSuite) TestIncreaseReplTimestampCollision(c *check.C) {
	// For purposes of increasing replication, we assume identical
	// replicas are distinct.
	bal.try(c, tester{
		desired:    4,
		current:    slots{0, 1},
		timestamps: []int64{12345678, 12345678},
		shouldPull: slots{2, 3}})
}

func (bal *balancerSuite) TestDecreaseReplTimestampCollision(c *check.C) {
	// For purposes of decreasing replication, we assume identical
	// replicas are NOT distinct.
	bal.try(c, tester{
		desired:    2,
		current:    slots{0, 1, 2},
		timestamps: []int64{12345678, 12345678, 12345678}})
	bal.try(c, tester{
		desired:    2,
		current:    slots{0, 1, 2},
		timestamps: []int64{12345678, 10000000, 10000000}})
}

func (bal *balancerSuite) TestDecreaseReplBlockTooNew(c *check.C) {
	oldTime := bal.MinMtime - 3600
	newTime := bal.MinMtime + 3600
	// The excess replica is too new to delete.
	bal.try(c, tester{
		desired:    2,
		current:    slots{0, 1, 2},
		timestamps: []int64{oldTime, newTime, newTime + 1}})
	// The best replicas are too new to delete, but the excess
	// replica is old enough.
	bal.try(c, tester{
		desired:     2,
		current:     slots{0, 1, 2},
		timestamps:  []int64{newTime, newTime + 1, oldTime},
		shouldTrash: slots{2}})
}

// Clear all servers' changesets, balance a single block, and verify
// the appropriate changes for that block have been added to the
// changesets.
func (bal *balancerSuite) try(c *check.C, t tester) {
	bal.setupServiceRoots()
	blk := &BlockState{
		Desired:  t.desired,
		Replicas: bal.replList(t.known, t.current)}
	for i, t := range t.timestamps {
		blk.Replicas[i].Mtime = t
	}
	for _, srv := range bal.srvs {
		srv.ChangeSet = &ChangeSet{}
	}
	bal.balanceBlock(knownBlkid(t.known), blk)

	var didPull, didTrash slots
	for i, srv := range bal.srvs {
		var slot int
		for probeOrder, srvNum := range bal.knownRendezvous[t.known] {
			if srvNum == i {
				slot = probeOrder
			}
		}
		for _, pull := range srv.Pulls {
			didPull = append(didPull, slot)
			c.Check(pull.SizedDigest, check.Equals, knownBlkid(t.known))
		}
		for _, trash := range srv.Trashes {
			didTrash = append(didTrash, slot)
			c.Check(trash.SizedDigest, check.Equals, knownBlkid(t.known))
		}
	}

	for _, list := range []slots{didPull, didTrash, t.shouldPull, t.shouldTrash} {
		sort.Sort(sort.IntSlice(list))
	}
	c.Check(didPull, check.DeepEquals, t.shouldPull)
	c.Check(didTrash, check.DeepEquals, t.shouldTrash)
}

// srvList returns the KeepServices, sorted in rendezvous order and
// then selected by idx. For example, srvList(3, 0, 1, 4) returns the
// the first-, second-, and fifth-best servers for storing
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
	mtime := time.Now().UnixNano() - (bal.signatureTTL+86400)*1e9
	for _, srv := range bal.srvList(knownBlockID, order) {
		repls = append(repls, Replica{srv, mtime})
		mtime++
	}
	return
}

// generate the same data hashes that are tested in
// sdk/go/keepclient/root_sorter_test.go
func knownBlkid(i int) arvados.SizedDigest {
	return arvados.SizedDigest(fmt.Sprintf("%x+64", md5.Sum([]byte(fmt.Sprintf("%064x", i)))))
}
