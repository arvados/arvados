// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"maps"
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
	Desired  mapPoolEnt
}

var defaultClasses = []string{"default"}

func (bs *BlockState) addReplica(r Replica) {
	bs.Replicas = append(bs.Replicas, r)
	// Free up memory wasted by tracking PDHs that will never be
	// reported (see comment in increaseDesired)
	bs.Refs = nil
}

func (bs *BlockState) increaseDesired(pool *mapPool, pdh string, classes []string, n int) {
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
		bs.Desired = pool.setMinimum(bs.Desired, class, n)
	}
}

// BlockStateMap is a goroutine-safe wrapper around a
// map[arvados.SizedDigest]*BlockState.
type BlockStateMap struct {
	entries map[arvados.SizedDigest]*BlockState
	pool    mapPool
	mutex   sync.Mutex
}

// NewBlockStateMap returns a newly allocated BlockStateMap.
func NewBlockStateMap(maxReplication int) *BlockStateMap {
	return &BlockStateMap{
		entries: make(map[arvados.SizedDigest]*BlockState),
		pool: mapPool{
			Maximum: maxReplication + 1,
		},
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
		bsm.get(blkid).increaseDesired(&bsm.pool, pdh, classes, n)
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
		bs, ok := bsm.entries[blkid]
		if !ok {
			return 0
		}
		for _, r := range bs.Replicas {
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

// mapPool manages a pool of distinct maps of type map[string]int.
// See (*mapPool)setMinimum() and (*BlockState)increaseDesired() for
// usage.
type mapPool struct {
	Maximum int
	next    map[mapPoolTransition]mapPoolEnt
	lock    sync.RWMutex
}

type mapPoolEnt *map[string]int

type mapPoolTransition struct {
	ent     mapPoolEnt
	class   string
	minimum int
}

// setMinimum returns a singleton mapPoolEnt that has
// ent[class]>=minimum and is equivalent to the provided ent for all
// other classes.
//
// The provided ent must be either nil, or a mapPoolEnt previously
// returned by this method.
//
// The provided ent will be returned if it already satisfies
// ent[class]>minimum.
//
// Functionally, it is equivalent to
//
//	if p.Maximum > 0 && minimum > p.Maximum {
//	        minimum = p.Maximum
//	}
//	if ent[class] < minimum {
//	        ent[class] = minimum
//	}
//
// Except that, as long as the mapPool is shared by many
// BlockState callers, it uses far less memory.
//
// The caller should not modify the returned mapPoolEnt.
func (p *mapPool) setMinimum(ent mapPoolEnt, class string, minimum int) mapPoolEnt {
	if ent != nil && (*ent)[class] >= minimum {
		return ent
	}
	if minimum > p.Maximum {
		// Clamp ent.minimum, otherwise p.next can become
		// excessively large when users set
		// replication_desired to unrealistic values.
		minimum = p.Maximum
	}
	transition := mapPoolTransition{
		ent:     ent,
		class:   class,
		minimum: minimum,
	}
	p.lock.RLock()
	next := p.next[transition]
	p.lock.RUnlock()
	if next != nil {
		return next
	}
	var newmap map[string]int
	if ent != nil {
		newmap = maps.Clone(*ent)
		newmap[class] = minimum
	} else {
		newmap = map[string]int{class: minimum}
	}

	p.lock.Lock()
	defer p.lock.Unlock()
	// The following find_matching_ent loop is not especially fast
	// -- O(poolsize*mapsize) at best -- but that's okay because
	// it runs only ~once per distinct pool entry, and the number
	// of pool entries is bounded by configuration (maximum
	// achievable replication ** number of storage classes)
	// regardless of how many blocks are being processed.
	//
	// Typically setMinimum is called millions of times but we
	// arrive at this loop less than 100 times.
find_matching_ent:
	for _, existing := range p.next {
		if len(*existing) != len(newmap) {
			continue find_matching_ent
		}
		for c, d := range *existing {
			if newmap[c] != d {
				continue find_matching_ent
			}
		}
		// reuse existing pool entry, previously added as
		// outcome of a different transition (or by another
		// goroutine between RUnlock and Lock, in which case
		// this is a no-op)
		p.next[transition] = existing
		return ent
	}
	if p.next == nil {
		p.next = make(map[mapPoolTransition]mapPoolEnt)
	}
	p.next[transition] = &newmap
	return &newmap
}
