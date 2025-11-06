// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&confirmedReplicationSuite{})

type confirmedReplicationSuite struct {
	blockStateMap *BlockStateMap
	mtime         int64
}

func (s *confirmedReplicationSuite) SetUpTest(c *check.C) {
	t, _ := time.Parse(time.RFC3339Nano, time.RFC3339Nano)
	s.mtime = t.UnixNano()
	s.blockStateMap = NewBlockStateMap(8)
	s.blockStateMap.AddReplicas(&KeepMount{KeepMount: arvados.KeepMount{
		Replication:    1,
		StorageClasses: map[string]bool{"default": true},
	}}, []arvados.KeepServiceIndexEntry{
		{SizedDigest: knownBlkid(10), Mtime: s.mtime},
	})
	s.blockStateMap.AddReplicas(&KeepMount{KeepMount: arvados.KeepMount{
		Replication:    2,
		StorageClasses: map[string]bool{"default": true},
	}}, []arvados.KeepServiceIndexEntry{
		{SizedDigest: knownBlkid(20), Mtime: s.mtime},
	})
}

func (s *confirmedReplicationSuite) TestZeroReplication(c *check.C) {
	n := s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(404), knownBlkid(409)}, []string{"default"})
	c.Check(n, check.Equals, 0)
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(10), knownBlkid(404)}, []string{"default"})
	c.Check(n, check.Equals, 0)
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(10), knownBlkid(404)}, nil)
	c.Check(n, check.Equals, 0)
}

func (s *confirmedReplicationSuite) TestBlocksWithDifferentReplication(c *check.C) {
	n := s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(10), knownBlkid(20)}, []string{"default"})
	c.Check(n, check.Equals, 1)
}

func (s *confirmedReplicationSuite) TestBlocksInDifferentClasses(c *check.C) {
	s.blockStateMap.AddReplicas(&KeepMount{KeepMount: arvados.KeepMount{
		Replication:    3,
		StorageClasses: map[string]bool{"three": true},
	}}, []arvados.KeepServiceIndexEntry{
		{SizedDigest: knownBlkid(30), Mtime: s.mtime},
	})

	n := s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(30)}, []string{"three"})
	c.Check(n, check.Equals, 3)
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(20), knownBlkid(30)}, []string{"default"})
	c.Check(n, check.Equals, 0) // block 30 has repl 0 @ "default"
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(20), knownBlkid(30)}, []string{"three"})
	c.Check(n, check.Equals, 0) // block 20 has repl 0 @ "three"
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(20), knownBlkid(30)}, nil)
	c.Check(n, check.Equals, 2)
}

func (s *confirmedReplicationSuite) TestBlocksOnMultipleMounts(c *check.C) {
	s.blockStateMap.AddReplicas(&KeepMount{KeepMount: arvados.KeepMount{
		Replication:    2,
		StorageClasses: map[string]bool{"default": true, "four": true},
	}}, []arvados.KeepServiceIndexEntry{
		{SizedDigest: knownBlkid(40), Mtime: s.mtime},
		{SizedDigest: knownBlkid(41), Mtime: s.mtime},
	})
	s.blockStateMap.AddReplicas(&KeepMount{KeepMount: arvados.KeepMount{
		Replication:    2,
		StorageClasses: map[string]bool{"four": true},
	}}, []arvados.KeepServiceIndexEntry{
		{SizedDigest: knownBlkid(40), Mtime: s.mtime},
		{SizedDigest: knownBlkid(41), Mtime: s.mtime},
	})
	n := s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(40), knownBlkid(41)}, []string{"default"})
	c.Check(n, check.Equals, 2)
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(40), knownBlkid(41)}, []string{"four"})
	c.Check(n, check.Equals, 4)
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(40), knownBlkid(41)}, []string{"default", "four"})
	c.Check(n, check.Equals, 2)
	n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(40), knownBlkid(41)}, nil)
	c.Check(n, check.Equals, 4)
}

func (s *confirmedReplicationSuite) TestConcurrency(c *check.C) {
	var wg sync.WaitGroup
	for i := 1000; i < 1256; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			n := s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(i), knownBlkid(i)}, []string{"default"})
			c.Check(n, check.Equals, 0)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			n := s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(10)}, []string{"default"})
			c.Check(n, check.Equals, 1)
			n = s.blockStateMap.GetConfirmedReplication([]arvados.SizedDigest{knownBlkid(20)}, []string{"default"})
			c.Check(n, check.Equals, 2)
		}()
	}
	wg.Wait()
}

var _ = check.Suite(&mapPoolSuite{})

type mapPoolSuite struct{}

func (s *mapPoolSuite) TestMapPool(c *check.C) {
	maxPoolReplication := 8
	maxDesired := 1000 // unrealistically high replication_desired
	nblocks := 10000
	classes := []string{"class_one", "class_two", "class_three"}
	bsm := NewBlockStateMap(maxPoolReplication)
	var wg sync.WaitGroup
	for i := 0; i < nblocks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bsm.IncreaseDesired("", classes, rand.Int()%maxDesired, []arvados.SizedDigest{knownBlkid(i)})
		}()
	}
	wg.Wait()

	// Check that the mapPool's "next" transition map does not get
	// too large, even with unrealistically high
	// replication_desired values.
	c.Logf("blocks==%d len(classes)==%d --> len(pool)==%d", nblocks, len(classes), len(bsm.pool.next))
	c.Check(len(bsm.pool.next) <= int(math.Pow(float64(maxPoolReplication+1), float64(len(classes)))), check.Equals, true)

	// Check that all pool entries are unique, i.e., if ent1 !=
	// ent2, then maps *ent1 and *ent2 have different content.
	ents := map[mapPoolEnt]bool{}
	for transition, ent := range bsm.pool.next {
		ents[ent] = true
		ents[transition.ent] = true
	}
	seen := map[string]bool{}
	for ent := range ents {
		var txt string
		if ent != nil {
			var classes []string
			for class := range *ent {
				classes = append(classes, class)
			}
			sort.Strings(classes)
			for _, class := range classes {
				txt += fmt.Sprintf("%s %d ", class, (*ent)[class])
			}
		}
		c.Check(seen[txt], check.Equals, false, check.Commentf("seen twice: %s", txt))
		seen[txt] = true
	}
	c.Assert(seen, check.HasLen, len(ents))
}
