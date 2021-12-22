// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
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
	s.blockStateMap = NewBlockStateMap()
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
