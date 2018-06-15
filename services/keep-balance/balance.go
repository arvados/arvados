// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

// CheckConfig returns an error if anything is wrong with the given
// config and runOptions.
func CheckConfig(config Config, runOptions RunOptions) error {
	if len(config.KeepServiceList.Items) > 0 && config.KeepServiceTypes != nil {
		return fmt.Errorf("cannot specify both KeepServiceList and KeepServiceTypes in config")
	}
	if !runOptions.Once && config.RunPeriod == arvados.Duration(0) {
		return fmt.Errorf("you must either use the -once flag, or specify RunPeriod in config")
	}
	return nil
}

// Balancer compares the contents of keepstore servers with the
// collections stored in Arvados, and issues pull/trash requests
// needed to get (closer to) the optimal data layout.
//
// In the optimal data layout: every data block referenced by a
// collection is replicated at least as many times as desired by the
// collection; there are no unreferenced data blocks older than
// BlobSignatureTTL; and all N existing replicas of a given data block
// are in the N best positions in rendezvous probe order.
type Balancer struct {
	*BlockStateMap
	KeepServices       map[string]*KeepService
	DefaultReplication int
	Logger             *log.Logger
	Dumper             *log.Logger
	MinMtime           int64

	classes       []string
	mounts        int
	mountsByClass map[string]map[*KeepMount]bool
	collScanned   int
	serviceRoots  map[string]string
	errors        []error
	stats         balancerStats
	mutex         sync.Mutex
}

// Run performs a balance operation using the given config and
// runOptions, and returns RunOptions suitable for passing to a
// subsequent balance operation.
//
// Run should only be called once on a given Balancer object.
//
// Typical usage:
//
//   runOptions, err = (&Balancer{}).Run(config, runOptions)
func (bal *Balancer) Run(config Config, runOptions RunOptions) (nextRunOptions RunOptions, err error) {
	nextRunOptions = runOptions

	bal.Dumper = runOptions.Dumper
	bal.Logger = runOptions.Logger
	if bal.Logger == nil {
		bal.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	defer timeMe(bal.Logger, "Run")()

	if len(config.KeepServiceList.Items) > 0 {
		err = bal.SetKeepServices(config.KeepServiceList)
	} else {
		err = bal.DiscoverKeepServices(&config.Client, config.KeepServiceTypes)
	}
	if err != nil {
		return
	}

	for _, srv := range bal.KeepServices {
		err = srv.discoverMounts(&config.Client)
		if err != nil {
			return
		}
	}
	bal.cleanupMounts()

	if err = bal.CheckSanityEarly(&config.Client); err != nil {
		return
	}
	rs := bal.rendezvousState()
	if runOptions.CommitTrash && rs != runOptions.SafeRendezvousState {
		if runOptions.SafeRendezvousState != "" {
			bal.logf("notice: KeepServices list has changed since last run")
		}
		bal.logf("clearing existing trash lists, in case the new rendezvous order differs from previous run")
		if err = bal.ClearTrashLists(&config.Client); err != nil {
			return
		}
		// The current rendezvous state becomes "safe" (i.e.,
		// OK to compute changes for that state without
		// clearing existing trash lists) only now, after we
		// succeed in clearing existing trash lists.
		nextRunOptions.SafeRendezvousState = rs
	}
	if err = bal.GetCurrentState(&config.Client, config.CollectionBatchSize, config.CollectionBuffers); err != nil {
		return
	}
	bal.ComputeChangeSets()
	bal.PrintStatistics()
	if err = bal.CheckSanityLate(); err != nil {
		return
	}
	if runOptions.CommitPulls {
		err = bal.CommitPulls(&config.Client)
		if err != nil {
			// Skip trash if we can't pull. (Too cautious?)
			return
		}
	}
	if runOptions.CommitTrash {
		err = bal.CommitTrash(&config.Client)
	}
	return
}

// SetKeepServices sets the list of KeepServices to operate on.
func (bal *Balancer) SetKeepServices(srvList arvados.KeepServiceList) error {
	bal.KeepServices = make(map[string]*KeepService)
	for _, srv := range srvList.Items {
		bal.KeepServices[srv.UUID] = &KeepService{
			KeepService: srv,
			ChangeSet:   &ChangeSet{},
		}
	}
	return nil
}

// DiscoverKeepServices sets the list of KeepServices by calling the
// API to get a list of all services, and selecting the ones whose
// ServiceType is in okTypes.
func (bal *Balancer) DiscoverKeepServices(c *arvados.Client, okTypes []string) error {
	bal.KeepServices = make(map[string]*KeepService)
	ok := make(map[string]bool)
	for _, t := range okTypes {
		ok[t] = true
	}
	return c.EachKeepService(func(srv arvados.KeepService) error {
		if ok[srv.ServiceType] {
			bal.KeepServices[srv.UUID] = &KeepService{
				KeepService: srv,
				ChangeSet:   &ChangeSet{},
			}
		} else {
			bal.logf("skipping %v with service type %q", srv.UUID, srv.ServiceType)
		}
		return nil
	})
}

func (bal *Balancer) cleanupMounts() {
	rwdev := map[string]*KeepService{}
	for _, srv := range bal.KeepServices {
		for _, mnt := range srv.mounts {
			if !mnt.ReadOnly && mnt.DeviceID != "" {
				rwdev[mnt.DeviceID] = srv
			}
		}
	}
	// Drop the readonly mounts whose device is mounted RW
	// elsewhere.
	for _, srv := range bal.KeepServices {
		var dedup []*KeepMount
		for _, mnt := range srv.mounts {
			if mnt.ReadOnly && rwdev[mnt.DeviceID] != nil {
				bal.logf("skipping srv %s readonly mount %q because same device %q is mounted read-write on srv %s", srv, mnt.UUID, mnt.DeviceID, rwdev[mnt.DeviceID])
			} else {
				dedup = append(dedup, mnt)
			}
		}
		srv.mounts = dedup
	}
	for _, srv := range bal.KeepServices {
		for _, mnt := range srv.mounts {
			if mnt.Replication <= 0 {
				log.Printf("%s: mount %s reports replication=%d, using replication=1", srv, mnt.UUID, mnt.Replication)
				mnt.Replication = 1
			}
		}
	}
}

// CheckSanityEarly checks for configuration and runtime errors that
// can be detected before GetCurrentState() and ComputeChangeSets()
// are called.
//
// If it returns an error, it is pointless to run GetCurrentState or
// ComputeChangeSets: after doing so, the statistics would be
// meaningless and it would be dangerous to run any Commit methods.
func (bal *Balancer) CheckSanityEarly(c *arvados.Client) error {
	u, err := c.CurrentUser()
	if err != nil {
		return fmt.Errorf("CurrentUser(): %v", err)
	}
	if !u.IsActive || !u.IsAdmin {
		return fmt.Errorf("current user (%s) is not an active admin user", u.UUID)
	}
	for _, srv := range bal.KeepServices {
		if srv.ServiceType == "proxy" {
			return fmt.Errorf("config error: %s: proxy servers cannot be balanced", srv)
		}
	}
	return nil
}

// rendezvousState returns a fingerprint (e.g., a sorted list of
// UUID+host+port) of the current set of keep services.
func (bal *Balancer) rendezvousState() string {
	srvs := make([]string, 0, len(bal.KeepServices))
	for _, srv := range bal.KeepServices {
		srvs = append(srvs, srv.String())
	}
	sort.Strings(srvs)
	return strings.Join(srvs, "; ")
}

// ClearTrashLists sends an empty trash list to each keep
// service. Calling this before GetCurrentState avoids races.
//
// When a block appears in an index, we assume that replica will still
// exist after we delete other replicas on other servers. However,
// it's possible that a previous rebalancing operation made different
// decisions (e.g., servers were added/removed, and rendezvous order
// changed). In this case, the replica might already be on that
// server's trash list, and it might be deleted before we send a
// replacement trash list.
//
// We avoid this problem if we clear all trash lists before getting
// indexes. (We also assume there is only one rebalancing process
// running at a time.)
func (bal *Balancer) ClearTrashLists(c *arvados.Client) error {
	for _, srv := range bal.KeepServices {
		srv.ChangeSet = &ChangeSet{}
	}
	return bal.CommitTrash(c)
}

// GetCurrentState determines the current replication state, and the
// desired replication level, for every block that is either
// retrievable or referenced.
//
// It determines the current replication state by reading the block index
// from every known Keep service.
//
// It determines the desired replication level by retrieving all
// collection manifests in the database (API server).
//
// It encodes the resulting information in BlockStateMap.
func (bal *Balancer) GetCurrentState(c *arvados.Client, pageSize, bufs int) error {
	defer timeMe(bal.Logger, "GetCurrentState")()
	bal.BlockStateMap = NewBlockStateMap()

	dd, err := c.DiscoveryDocument()
	if err != nil {
		return err
	}
	bal.DefaultReplication = dd.DefaultCollectionReplication
	bal.MinMtime = time.Now().UnixNano() - dd.BlobSignatureTTL*1e9

	errs := make(chan error, 2+len(bal.KeepServices))
	wg := sync.WaitGroup{}

	// When a device is mounted more than once, we will get its
	// index only once, and call AddReplicas on all of the mounts.
	// equivMount keys are the mounts that will be indexed, and
	// each value is a list of mounts to apply the received index
	// to.
	equivMount := map[*KeepMount][]*KeepMount{}
	// deviceMount maps each device ID to the one mount that will
	// be indexed for that device.
	deviceMount := map[string]*KeepMount{}
	for _, srv := range bal.KeepServices {
		for _, mnt := range srv.mounts {
			equiv := deviceMount[mnt.DeviceID]
			if equiv == nil {
				equiv = mnt
				if mnt.DeviceID != "" {
					deviceMount[mnt.DeviceID] = equiv
				}
			}
			equivMount[equiv] = append(equivMount[equiv], mnt)
		}
	}

	// Start one goroutine for each (non-redundant) mount:
	// retrieve the index, and add the returned blocks to
	// BlockStateMap.
	for _, mounts := range equivMount {
		wg.Add(1)
		go func(mounts []*KeepMount) {
			defer wg.Done()
			bal.logf("mount %s: retrieve index from %s", mounts[0], mounts[0].KeepService)
			idx, err := mounts[0].KeepService.IndexMount(c, mounts[0].UUID, "")
			if err != nil {
				errs <- fmt.Errorf("%s: retrieve index: %v", mounts[0], err)
				return
			}
			if len(errs) > 0 {
				// Some other goroutine encountered an
				// error -- any further effort here
				// will be wasted.
				return
			}
			for _, mount := range mounts {
				bal.logf("%s: add %d replicas to map", mount, len(idx))
				bal.BlockStateMap.AddReplicas(mount, idx)
				bal.logf("%s: added %d replicas", mount, len(idx))
			}
			bal.logf("mount %s: index done", mounts[0])
		}(mounts)
	}

	// collQ buffers incoming collections so we can start fetching
	// the next page without waiting for the current page to
	// finish processing.
	collQ := make(chan arvados.Collection, bufs)

	// Start a goroutine to process collections. (We could use a
	// worker pool here, but even with a single worker we already
	// process collections much faster than we can retrieve them.)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for coll := range collQ {
			err := bal.addCollection(coll)
			if err != nil {
				errs <- err
				for range collQ {
				}
				return
			}
			bal.collScanned++
		}
	}()

	// Start a goroutine to retrieve all collections from the
	// Arvados database and send them to collQ for processing.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = EachCollection(c, pageSize,
			func(coll arvados.Collection) error {
				collQ <- coll
				if len(errs) > 0 {
					// some other GetCurrentState
					// error happened: no point
					// getting any more
					// collections.
					return fmt.Errorf("")
				}
				return nil
			}, func(done, total int) {
				bal.logf("collections: %d/%d", done, total)
			})
		close(collQ)
		if err != nil {
			errs <- err
		}
	}()

	wg.Wait()
	if len(errs) > 0 {
		return <-errs
	}
	return nil
}

func (bal *Balancer) addCollection(coll arvados.Collection) error {
	blkids, err := coll.SizedDigests()
	if err != nil {
		bal.mutex.Lock()
		bal.errors = append(bal.errors, fmt.Errorf("%v: %v", coll.UUID, err))
		bal.mutex.Unlock()
		return nil
	}
	repl := bal.DefaultReplication
	if coll.ReplicationDesired != nil {
		repl = *coll.ReplicationDesired
	}
	debugf("%v: %d block x%d", coll.UUID, len(blkids), repl)
	bal.BlockStateMap.IncreaseDesired(coll.StorageClassesDesired, repl, blkids)
	return nil
}

// ComputeChangeSets compares, for each known block, the current and
// desired replication states. If it is possible to get closer to the
// desired state by copying or deleting blocks, it adds those changes
// to the relevant KeepServices' ChangeSets.
//
// It does not actually apply any of the computed changes.
func (bal *Balancer) ComputeChangeSets() {
	// This just calls balanceBlock() once for each block, using a
	// pool of worker goroutines.
	defer timeMe(bal.Logger, "ComputeChangeSets")()
	bal.setupLookupTables()

	type balanceTask struct {
		blkid arvados.SizedDigest
		blk   *BlockState
	}
	workers := runtime.GOMAXPROCS(-1)
	todo := make(chan balanceTask, workers)
	go func() {
		bal.BlockStateMap.Apply(func(blkid arvados.SizedDigest, blk *BlockState) {
			todo <- balanceTask{
				blkid: blkid,
				blk:   blk,
			}
		})
		close(todo)
	}()
	results := make(chan balanceResult, workers)
	go func() {
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				for work := range todo {
					results <- bal.balanceBlock(work.blkid, work.blk)
				}
				wg.Done()
			}()
		}
		wg.Wait()
		close(results)
	}()
	bal.collectStatistics(results)
}

func (bal *Balancer) setupLookupTables() {
	bal.serviceRoots = make(map[string]string)
	bal.classes = []string{"default"}
	bal.mountsByClass = map[string]map[*KeepMount]bool{"default": {}}
	bal.mounts = 0
	for _, srv := range bal.KeepServices {
		bal.serviceRoots[srv.UUID] = srv.UUID
		for _, mnt := range srv.mounts {
			bal.mounts++

			// All mounts on a read-only service are
			// effectively read-only.
			mnt.ReadOnly = mnt.ReadOnly || srv.ReadOnly

			if len(mnt.StorageClasses) == 0 {
				bal.mountsByClass["default"][mnt] = true
				continue
			}
			for _, class := range mnt.StorageClasses {
				if mbc := bal.mountsByClass[class]; mbc == nil {
					bal.classes = append(bal.classes, class)
					bal.mountsByClass[class] = map[*KeepMount]bool{mnt: true}
				} else {
					mbc[mnt] = true
				}
			}
		}
	}
	// Consider classes in lexicographic order to avoid flapping
	// between balancing runs.  The outcome of the "prefer a mount
	// we're already planning to use for a different storage
	// class" case in balanceBlock depends on the order classes
	// are considered.
	sort.Strings(bal.classes)
}

const (
	changeStay = iota
	changePull
	changeTrash
	changeNone
)

var changeName = map[int]string{
	changeStay:  "stay",
	changePull:  "pull",
	changeTrash: "trash",
	changeNone:  "none",
}

type balanceResult struct {
	blk        *BlockState
	blkid      arvados.SizedDigest
	have       int
	want       int
	classState map[string]balancedBlockState
}

// balanceBlock compares current state to desired state for a single
// block, and makes the appropriate ChangeSet calls.
func (bal *Balancer) balanceBlock(blkid arvados.SizedDigest, blk *BlockState) balanceResult {
	debugf("balanceBlock: %v %+v", blkid, blk)

	type slot struct {
		mnt  *KeepMount // never nil
		repl *Replica   // replica already stored here (or nil)
		want bool       // we should pull/leave a replica here
	}

	// Build a list of all slots (one per mounted volume).
	slots := make([]slot, 0, bal.mounts)
	for _, srv := range bal.KeepServices {
		for _, mnt := range srv.mounts {
			var repl *Replica
			for r := range blk.Replicas {
				if blk.Replicas[r].KeepMount == mnt {
					repl = &blk.Replicas[r]
				}
			}
			// Initial value of "want" is "have, and can't
			// delete". These untrashable replicas get
			// prioritized when sorting slots: otherwise,
			// non-optimal readonly copies would cause us
			// to overreplicate.
			slots = append(slots, slot{
				mnt:  mnt,
				repl: repl,
				want: repl != nil && (mnt.ReadOnly || repl.Mtime >= bal.MinMtime),
			})
		}
	}

	uuids := keepclient.NewRootSorter(bal.serviceRoots, string(blkid[:32])).GetSortedRoots()
	srvRendezvous := make(map[*KeepService]int, len(uuids))
	for i, uuid := range uuids {
		srv := bal.KeepServices[uuid]
		srvRendezvous[srv] = i
	}

	// Below we set underreplicated=true if we find any storage
	// class that's currently underreplicated -- in that case we
	// won't want to trash any replicas.
	underreplicated := false

	classState := make(map[string]balancedBlockState, len(bal.classes))
	unsafeToDelete := make(map[int64]bool, len(slots))
	for _, class := range bal.classes {
		desired := blk.Desired[class]

		countedDev := map[string]bool{}
		have := 0
		for _, slot := range slots {
			if slot.repl != nil && bal.mountsByClass[class][slot.mnt] && !countedDev[slot.mnt.DeviceID] {
				have += slot.mnt.Replication
				if slot.mnt.DeviceID != "" {
					countedDev[slot.mnt.DeviceID] = true
				}
			}
		}
		classState[class] = balancedBlockState{
			desired: desired,
			surplus: have - desired,
		}

		if desired == 0 {
			continue
		}

		// Sort the slots by desirability.
		sort.Slice(slots, func(i, j int) bool {
			si, sj := slots[i], slots[j]
			if classi, classj := bal.mountsByClass[class][si.mnt], bal.mountsByClass[class][sj.mnt]; classi != classj {
				// Prefer a mount that satisfies the
				// desired class.
				return bal.mountsByClass[class][si.mnt]
			} else if wanti, wantj := si.want, si.want; wanti != wantj {
				// Prefer a mount that will have a
				// replica no matter what we do here
				// -- either because it already has an
				// untrashable replica, or because we
				// already need it to satisfy a
				// different storage class.
				return slots[i].want
			} else if orderi, orderj := srvRendezvous[si.mnt.KeepService], srvRendezvous[sj.mnt.KeepService]; orderi != orderj {
				// Prefer a better rendezvous
				// position.
				return orderi < orderj
			} else if repli, replj := si.repl != nil, sj.repl != nil; repli != replj {
				// Prefer a mount that already has a
				// replica.
				return repli
			} else {
				// If pull/trash turns out to be
				// needed, distribute the
				// new/remaining replicas uniformly
				// across qualifying mounts on a given
				// server.
				return rendezvousLess(si.mnt.DeviceID, sj.mnt.DeviceID, blkid)
			}
		})

		// Servers/mounts/devices (with or without existing
		// replicas) that are part of the best achievable
		// layout for this storage class.
		wantSrv := map[*KeepService]bool{}
		wantMnt := map[*KeepMount]bool{}
		wantDev := map[string]bool{}
		// Positions (with existing replicas) that have been
		// protected (via unsafeToDelete) to ensure we don't
		// reduce replication below desired level when
		// trashing replicas that aren't optimal positions for
		// any storage class.
		protMnt := map[*KeepMount]bool{}
		// Replication planned so far (corresponds to wantMnt).
		replWant := 0
		// Protected replication (corresponds to protMnt).
		replProt := 0

		// trySlot tries using a slot to meet requirements,
		// and returns true if all requirements are met.
		trySlot := func(i int) bool {
			slot := slots[i]
			if wantMnt[slot.mnt] || wantDev[slot.mnt.DeviceID] {
				// Already allocated a replica to this
				// backend device, possibly on a
				// different server.
				return false
			}
			if replProt < desired && slot.repl != nil && !protMnt[slot.mnt] {
				unsafeToDelete[slot.repl.Mtime] = true
				protMnt[slot.mnt] = true
				replProt += slot.mnt.Replication
			}
			if replWant < desired && (slot.repl != nil || !slot.mnt.ReadOnly) {
				slots[i].want = true
				wantSrv[slot.mnt.KeepService] = true
				wantMnt[slot.mnt] = true
				if slot.mnt.DeviceID != "" {
					wantDev[slot.mnt.DeviceID] = true
				}
				replWant += slot.mnt.Replication
			}
			return replProt >= desired && replWant >= desired
		}

		// First try to achieve desired replication without
		// using the same server twice.
		done := false
		for i := 0; i < len(slots) && !done; i++ {
			if !wantSrv[slots[i].mnt.KeepService] {
				done = trySlot(i)
			}
		}

		// If that didn't suffice, do another pass without the
		// "distinct services" restriction. (Achieving the
		// desired volume replication on fewer than the
		// desired number of services is better than
		// underreplicating.)
		for i := 0; i < len(slots) && !done; i++ {
			done = trySlot(i)
		}

		if !underreplicated {
			safe := 0
			for _, slot := range slots {
				if slot.repl == nil || !bal.mountsByClass[class][slot.mnt] {
					continue
				}
				if safe += slot.mnt.Replication; safe >= desired {
					break
				}
			}
			underreplicated = safe < desired
		}

		// set the unachievable flag if there aren't enough
		// slots offering the relevant storage class. (This is
		// as easy as checking slots[desired] because we
		// already sorted the qualifying slots to the front.)
		if desired >= len(slots) || !bal.mountsByClass[class][slots[desired].mnt] {
			cs := classState[class]
			cs.unachievable = true
			classState[class] = cs
		}

		// Avoid deleting wanted replicas from devices that
		// are mounted on multiple servers -- even if they
		// haven't already been added to unsafeToDelete
		// because the servers report different Mtimes.
		for _, slot := range slots {
			if slot.repl != nil && wantDev[slot.mnt.DeviceID] {
				unsafeToDelete[slot.repl.Mtime] = true
			}
		}
	}

	// TODO: If multiple replicas are trashable, prefer the oldest
	// replica that doesn't have a timestamp collision with
	// others.

	countedDev := map[string]bool{}
	var have, want int
	for _, slot := range slots {
		if countedDev[slot.mnt.DeviceID] {
			continue
		}
		if slot.want {
			want += slot.mnt.Replication
		}
		if slot.repl != nil {
			have += slot.mnt.Replication
		}
		if slot.mnt.DeviceID != "" {
			countedDev[slot.mnt.DeviceID] = true
		}
	}

	var changes []string
	for _, slot := range slots {
		// TODO: request a Touch if Mtime is duplicated.
		var change int
		switch {
		case !underreplicated && slot.repl != nil && !slot.want && !unsafeToDelete[slot.repl.Mtime]:
			slot.mnt.KeepService.AddTrash(Trash{
				SizedDigest: blkid,
				Mtime:       slot.repl.Mtime,
				From:        slot.mnt,
			})
			change = changeTrash
		case len(blk.Replicas) == 0:
			change = changeNone
		case slot.repl == nil && slot.want && !slot.mnt.ReadOnly:
			slot.mnt.KeepService.AddPull(Pull{
				SizedDigest: blkid,
				From:        blk.Replicas[0].KeepMount.KeepService,
				To:          slot.mnt,
			})
			change = changePull
		default:
			change = changeStay
		}
		if bal.Dumper != nil {
			var mtime int64
			if slot.repl != nil {
				mtime = slot.repl.Mtime
			}
			srv := slot.mnt.KeepService
			changes = append(changes, fmt.Sprintf("%s:%d/%s=%s,%d", srv.ServiceHost, srv.ServicePort, slot.mnt.UUID, changeName[change], mtime))
		}
	}
	if bal.Dumper != nil {
		bal.Dumper.Printf("%s have=%d want=%v %s", blkid, have, want, strings.Join(changes, " "))
	}
	return balanceResult{
		blk:        blk,
		blkid:      blkid,
		have:       have,
		want:       want,
		classState: classState,
	}
}

type blocksNBytes struct {
	replicas int
	blocks   int
	bytes    int64
}

func (bb blocksNBytes) String() string {
	return fmt.Sprintf("%d replicas (%d blocks, %d bytes)", bb.replicas, bb.blocks, bb.bytes)
}

type balancerStats struct {
	lost          blocksNBytes
	overrep       blocksNBytes
	unref         blocksNBytes
	garbage       blocksNBytes
	underrep      blocksNBytes
	unachievable  blocksNBytes
	justright     blocksNBytes
	desired       blocksNBytes
	current       blocksNBytes
	pulls         int
	trashes       int
	replHistogram []int
	classStats    map[string]replicationStats
}

type replicationStats struct {
	desired      blocksNBytes
	surplus      blocksNBytes
	short        blocksNBytes
	unachievable blocksNBytes
}

type balancedBlockState struct {
	desired      int
	surplus      int
	unachievable bool
}

func (bal *Balancer) collectStatistics(results <-chan balanceResult) {
	var s balancerStats
	s.replHistogram = make([]int, 2)
	s.classStats = make(map[string]replicationStats, len(bal.classes))
	for result := range results {
		surplus := result.have - result.want
		bytes := result.blkid.Size()

		for class, state := range result.classState {
			cs := s.classStats[class]
			if state.unachievable {
				cs.unachievable.blocks++
				cs.unachievable.bytes += bytes
			}
			if state.desired > 0 {
				cs.desired.replicas += state.desired
				cs.desired.blocks++
				cs.desired.bytes += bytes * int64(state.desired)
			}
			if state.surplus > 0 {
				cs.surplus.replicas += state.surplus
				cs.surplus.blocks++
				cs.surplus.bytes += bytes * int64(state.surplus)
			} else if state.surplus < 0 {
				cs.short.replicas += -state.surplus
				cs.short.blocks++
				cs.short.bytes += bytes * int64(-state.surplus)
			}
			s.classStats[class] = cs
		}

		switch {
		case result.have == 0 && result.want > 0:
			s.lost.replicas -= surplus
			s.lost.blocks++
			s.lost.bytes += bytes * int64(-surplus)
		case surplus < 0:
			s.underrep.replicas -= surplus
			s.underrep.blocks++
			s.underrep.bytes += bytes * int64(-surplus)
		case surplus > 0 && result.want == 0:
			counter := &s.garbage
			for _, r := range result.blk.Replicas {
				if r.Mtime >= bal.MinMtime {
					counter = &s.unref
					break
				}
			}
			counter.replicas += surplus
			counter.blocks++
			counter.bytes += bytes * int64(surplus)
		case surplus > 0:
			s.overrep.replicas += surplus
			s.overrep.blocks++
			s.overrep.bytes += bytes * int64(result.have-result.want)
		default:
			s.justright.replicas += result.want
			s.justright.blocks++
			s.justright.bytes += bytes * int64(result.want)
		}

		if result.want > 0 {
			s.desired.replicas += result.want
			s.desired.blocks++
			s.desired.bytes += bytes * int64(result.want)
		}
		if result.have > 0 {
			s.current.replicas += result.have
			s.current.blocks++
			s.current.bytes += bytes * int64(result.have)
		}

		for len(s.replHistogram) <= result.have {
			s.replHistogram = append(s.replHistogram, 0)
		}
		s.replHistogram[result.have]++
	}
	for _, srv := range bal.KeepServices {
		s.pulls += len(srv.ChangeSet.Pulls)
		s.trashes += len(srv.ChangeSet.Trashes)
	}
	bal.stats = s
}

// PrintStatistics writes statistics about the computed changes to
// bal.Logger. It should not be called until ComputeChangeSets has
// finished.
func (bal *Balancer) PrintStatistics() {
	bal.logf("===")
	bal.logf("%s lost (0=have<want)", bal.stats.lost)
	bal.logf("%s underreplicated (0<have<want)", bal.stats.underrep)
	bal.logf("%s just right (have=want)", bal.stats.justright)
	bal.logf("%s overreplicated (have>want>0)", bal.stats.overrep)
	bal.logf("%s unreferenced (have>want=0, new)", bal.stats.unref)
	bal.logf("%s garbage (have>want=0, old)", bal.stats.garbage)
	for _, class := range bal.classes {
		cs := bal.stats.classStats[class]
		bal.logf("===")
		bal.logf("storage class %q: %s desired", class, cs.desired)
		bal.logf("storage class %q: %s short", class, cs.short)
		bal.logf("storage class %q: %s surplus", class, cs.surplus)
		bal.logf("storage class %q: %s unachievable", class, cs.unachievable)
	}
	bal.logf("===")
	bal.logf("%s total commitment (excluding unreferenced)", bal.stats.desired)
	bal.logf("%s total usage", bal.stats.current)
	bal.logf("===")
	for _, srv := range bal.KeepServices {
		bal.logf("%s: %v\n", srv, srv.ChangeSet)
	}
	bal.logf("===")
	bal.printHistogram(60)
	bal.logf("===")
}

func (bal *Balancer) printHistogram(hashColumns int) {
	bal.logf("Replication level distribution (counting N replicas on a single server as N):")
	maxCount := 0
	for _, count := range bal.stats.replHistogram {
		if maxCount < count {
			maxCount = count
		}
	}
	hashes := strings.Repeat("#", hashColumns)
	countWidth := 1 + int(math.Log10(float64(maxCount+1)))
	scaleCount := 10 * float64(hashColumns) / math.Floor(1+10*math.Log10(float64(maxCount+1)))
	for repl, count := range bal.stats.replHistogram {
		nHashes := int(scaleCount * math.Log10(float64(count+1)))
		bal.logf("%2d: %*d %s", repl, countWidth, count, hashes[:nHashes])
	}
}

// CheckSanityLate checks for configuration and runtime errors after
// GetCurrentState() and ComputeChangeSets() have finished.
//
// If it returns an error, it is dangerous to run any Commit methods.
func (bal *Balancer) CheckSanityLate() error {
	if bal.errors != nil {
		for _, err := range bal.errors {
			bal.logf("deferred error: %v", err)
		}
		return fmt.Errorf("cannot proceed safely after deferred errors")
	}

	if bal.collScanned == 0 {
		return fmt.Errorf("received zero collections")
	}

	anyDesired := false
	bal.BlockStateMap.Apply(func(_ arvados.SizedDigest, blk *BlockState) {
		for _, desired := range blk.Desired {
			if desired > 0 {
				anyDesired = true
				break
			}
		}
	})
	if !anyDesired {
		return fmt.Errorf("zero blocks have desired replication>0")
	}

	if dr := bal.DefaultReplication; dr < 1 {
		return fmt.Errorf("Default replication (%d) is less than 1", dr)
	}

	// TODO: no two services have identical indexes
	// TODO: no collisions (same md5, different size)
	return nil
}

// CommitPulls sends the computed lists of pull requests to the
// keepstore servers. This has the effect of increasing replication of
// existing blocks that are either underreplicated or poorly
// distributed according to rendezvous hashing.
func (bal *Balancer) CommitPulls(c *arvados.Client) error {
	return bal.commitAsync(c, "send pull list",
		func(srv *KeepService) error {
			return srv.CommitPulls(c)
		})
}

// CommitTrash sends the computed lists of trash requests to the
// keepstore servers. This has the effect of deleting blocks that are
// overreplicated or unreferenced.
func (bal *Balancer) CommitTrash(c *arvados.Client) error {
	return bal.commitAsync(c, "send trash list",
		func(srv *KeepService) error {
			return srv.CommitTrash(c)
		})
}

func (bal *Balancer) commitAsync(c *arvados.Client, label string, f func(srv *KeepService) error) error {
	errs := make(chan error)
	for _, srv := range bal.KeepServices {
		go func(srv *KeepService) {
			var err error
			defer func() { errs <- err }()
			label := fmt.Sprintf("%s: %v", srv, label)
			defer timeMe(bal.Logger, label)()
			err = f(srv)
			if err != nil {
				err = fmt.Errorf("%s: %v", label, err)
			}
		}(srv)
	}
	var lastErr error
	for range bal.KeepServices {
		if err := <-errs; err != nil {
			bal.logf("%v", err)
			lastErr = err
		}
	}
	close(errs)
	return lastErr
}

func (bal *Balancer) logf(f string, args ...interface{}) {
	if bal.Logger != nil {
		bal.Logger.Printf(f, args...)
	}
}

// Rendezvous hash sort function. Less efficient than sorting on
// precomputed rendezvous hashes, but also rarely used.
func rendezvousLess(i, j string, blkid arvados.SizedDigest) bool {
	a := md5.Sum([]byte(string(blkid[:32]) + i))
	b := md5.Sum([]byte(string(blkid[:32]) + j))
	return bytes.Compare(a[:], b[:]) < 0
}
