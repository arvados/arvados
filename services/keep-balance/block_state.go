// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"sync"

	"git.arvados.org/arvados.git/sdk/go/arvados"
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
	Refs     map[string]bool // pdh => true (only tracked when len(Replicas)==0)
	RefCount int
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
	// Free up memory wasted by tracking PDHs that will never be
	// reported (see comment in increaseDesired)
	bs.Refs = nil
}

func (bs *BlockState) increaseDesired(pdh string, classes []string, n int) {
	if pdh != "" && len(bs.Replicas) == 0 {
		// Note we only track PDHs if there's a possibility
		// that we will report the list of referring PDHs,
		// i.e., if we haven't yet seen a replica.
		if bs.Refs == nil {
			bs.Refs = map[string]bool{}
		}
		bs.Refs[pdh] = true
	}
	bs.RefCount++
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
//
// If pdh is non-empty, it will be tracked and reported in the "lost
// blocks" report.
func (bsm *BlockStateMap) IncreaseDesired(pdh string, classes []string, n int, blocks []arvados.SizedDigest) {
	bsm.mutex.Lock()
	defer bsm.mutex.Unlock()

	for _, blkid := range blocks {
		bsm.get(blkid).increaseDesired(pdh, classes, n)
	}
}

// GetConfirmedReplication returns the replication level of the given
// blocks, considering only the specified storage classes.
//
// If len(classes)==0, returns the replication level without regard to
// storage classes.
//
// Safe to call concurrently with other calls to GetCurrent, but not
// with different BlockStateMap methods.
func (bsm *BlockStateMap) GetConfirmedReplication(blkids []arvados.SizedDigest, classes []string) int {
	defaultClasses := map[string]bool{"default": true}
	min := 0
	for _, blkid := range blkids {
		total := 0
		perclass := make(map[string]int, len(classes))
		for _, c := range classes {
			perclass[c] = 0
		}
		for _, r := range bsm.get(blkid).Replicas {
			total += r.KeepMount.Replication
			mntclasses := r.KeepMount.StorageClasses
			if len(mntclasses) == 0 {
				mntclasses = defaultClasses
			}
			for c := range mntclasses {
				n, ok := perclass[c]
				if !ok {
					// Don't care about this storage class
					continue
				}
				perclass[c] = n + r.KeepMount.Replication
			}
		}
		if total == 0 {
			return 0
		}
		for _, n := range perclass {
			if n == 0 {
				return 0
			}
			if n < min || min == 0 {
				min = n
			}
		}
		if len(perclass) == 0 && (min == 0 || min > total) {
			min = total
		}
	}
	return min
}
