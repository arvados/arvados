// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// Replica is a file on disk (or object in an S3 bucket, or blob in an
// Azure storage container, etc.) as reported in a keepstore index
// response.
type Replica struct {
	*KeepMount
	Mtime int64
}

// BlockState indicates the desired storage class and number of
// replicas (according to the collections we know about) and the
// replicas actually stored (according to the keepstore indexes we
// know about).
type BlockState struct {
	Replicas []Replica
	Desired  map[string]int
	// TODO: Support combinations of classes ("private + durable")
	// by replacing the map[string]int with a map[*[]string]int
	// here, where the map keys come from a pool of semantically
	// distinct class combinations.
	//
	// TODO: Use a pool of semantically distinct Desired maps to
	// conserve memory (typically there are far more BlockState
	// objects in memory than distinct Desired profiles).
}

var defaultClasses = []string{"default"}

func (bs *BlockState) addReplica(r Replica) {
	bs.Replicas = append(bs.Replicas, r)
}

func (bs *BlockState) increaseDesired(classes []string, n int) {
	if len(classes) == 0 {
		classes = defaultClasses
	}
	for _, class := range classes {
		if bs.Desired == nil {
			bs.Desired = map[string]int{class: n}
		} else if d, ok := bs.Desired[class]; !ok || d < n {
			bs.Desired[class] = n
		}
	}
}

// BlockStateMap is a goroutine-safe wrapper around a
// map[arvados.SizedDigest]*BlockState.
type BlockStateMap struct {
	entries map[arvados.SizedDigest]*BlockState
	mutex   sync.Mutex
}

// NewBlockStateMap returns a newly allocated BlockStateMap.
func NewBlockStateMap() *BlockStateMap {
	return &BlockStateMap{
		entries: make(map[arvados.SizedDigest]*BlockState),
	}
}

// return a BlockState entry, allocating a new one if needed. (Private
// method: not goroutine-safe.)
func (bsm *BlockStateMap) get(blkid arvados.SizedDigest) *BlockState {
	// TODO? Allocate BlockState structs a slice at a time,
	// instead of one at a time.
	blk := bsm.entries[blkid]
	if blk == nil {
		blk = &BlockState{}
		bsm.entries[blkid] = blk
	}
	return blk
}

// Apply runs f on each entry in the map.
func (bsm *BlockStateMap) Apply(f func(arvados.SizedDigest, *BlockState)) {
	bsm.mutex.Lock()
	defer bsm.mutex.Unlock()

	for blkid, blk := range bsm.entries {
		f(blkid, blk)
	}
}

// AddReplicas updates the map to indicate that mnt has a replica of
// each block in idx.
func (bsm *BlockStateMap) AddReplicas(mnt *KeepMount, idx []arvados.KeepServiceIndexEntry) {
	bsm.mutex.Lock()
	defer bsm.mutex.Unlock()

	for _, ent := range idx {
		bsm.get(ent.SizedDigest).addReplica(Replica{
			KeepMount: mnt,
			Mtime:     ent.Mtime,
		})
	}
}

// IncreaseDesired updates the map to indicate the desired replication
// for the given blocks in the given storage class is at least n.
func (bsm *BlockStateMap) IncreaseDesired(classes []string, n int, blocks []arvados.SizedDigest) {
	bsm.mutex.Lock()
	defer bsm.mutex.Unlock()

	for _, blkid := range blocks {
		bsm.get(blkid).increaseDesired(classes, n)
	}
}
