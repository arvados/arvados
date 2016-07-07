package main

import (
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

	collScanned  int
	serviceRoots map[string]string
	errors       []error
	mutex        sync.Mutex
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

	// Start one goroutine for each KeepService: retrieve the
	// index, and add the returned blocks to BlockStateMap.
	for _, srv := range bal.KeepServices {
		wg.Add(1)
		go func(srv *KeepService) {
			defer wg.Done()
			bal.logf("%s: retrieve index", srv)
			idx, err := srv.Index(c, "")
			if err != nil {
				errs <- fmt.Errorf("%s: %v", srv, err)
				return
			}
			bal.logf("%s: add %d replicas to map", srv, len(idx))
			bal.BlockStateMap.AddReplicas(srv, idx)
			bal.logf("%s: done", srv)
		}(srv)
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

	go func() {
		// Send a nil error when all goroutines finish. If
		// this is the first error sent to errs, then
		// everything worked.
		wg.Wait()
		errs <- nil
	}()
	return <-errs
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
	bal.BlockStateMap.IncreaseDesired(repl, blkids)
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
	bal.setupServiceRoots()

	type balanceTask struct {
		blkid arvados.SizedDigest
		blk   *BlockState
	}
	nWorkers := 1 + runtime.NumCPU()
	todo := make(chan balanceTask, nWorkers)
	var wg sync.WaitGroup
	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func() {
			for work := range todo {
				bal.balanceBlock(work.blkid, work.blk)
			}
			wg.Done()
		}()
	}
	bal.BlockStateMap.Apply(func(blkid arvados.SizedDigest, blk *BlockState) {
		todo <- balanceTask{
			blkid: blkid,
			blk:   blk,
		}
	})
	close(todo)
	wg.Wait()
}

func (bal *Balancer) setupServiceRoots() {
	bal.serviceRoots = make(map[string]string)
	for _, srv := range bal.KeepServices {
		bal.serviceRoots[srv.UUID] = srv.UUID
	}
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

// balanceBlock compares current state to desired state for a single
// block, and makes the appropriate ChangeSet calls.
func (bal *Balancer) balanceBlock(blkid arvados.SizedDigest, blk *BlockState) {
	debugf("balanceBlock: %v %+v", blkid, blk)
	uuids := keepclient.NewRootSorter(bal.serviceRoots, string(blkid[:32])).GetSortedRoots()
	hasRepl := make(map[string]Replica, len(bal.serviceRoots))
	for _, repl := range blk.Replicas {
		hasRepl[repl.UUID] = repl
		// TODO: when multiple copies are on one server, use
		// the oldest one that doesn't have a timestamp
		// collision with other replicas.
	}
	// number of replicas already found in positions better than
	// the position we're contemplating now.
	reportedBestRepl := 0
	// To be safe we assume two replicas with the same Mtime are
	// in fact the same replica being reported more than
	// once. len(uniqueBestRepl) is the number of distinct
	// replicas in the best rendezvous positions we've considered
	// so far.
	uniqueBestRepl := make(map[int64]bool, len(bal.serviceRoots))
	// pulls is the number of Pull changes we have already
	// requested. (For purposes of deciding whether to Pull to
	// rendezvous position N, we should assume all pulls we have
	// requested on rendezvous positions M<N will be successful.)
	pulls := 0
	var changes []string
	for _, uuid := range uuids {
		change := changeNone
		srv := bal.KeepServices[uuid]
		// TODO: request a Touch if Mtime is duplicated.
		repl, ok := hasRepl[srv.UUID]
		if ok {
			// This service has a replica. We should
			// delete it if [1] we already have enough
			// distinct replicas in better rendezvous
			// positions and [2] this replica's Mtime is
			// distinct from all of the better replicas'
			// Mtimes.
			if !srv.ReadOnly &&
				repl.Mtime < bal.MinMtime &&
				len(uniqueBestRepl) >= blk.Desired &&
				!uniqueBestRepl[repl.Mtime] {
				srv.AddTrash(Trash{
					SizedDigest: blkid,
					Mtime:       repl.Mtime,
				})
				change = changeTrash
			} else {
				change = changeStay
			}
			uniqueBestRepl[repl.Mtime] = true
			reportedBestRepl++
		} else if pulls+reportedBestRepl < blk.Desired &&
			len(blk.Replicas) > 0 &&
			!srv.ReadOnly {
			// This service doesn't have a replica. We
			// should pull one to this server if we don't
			// already have enough (existing+requested)
			// replicas in better rendezvous positions.
			srv.AddPull(Pull{
				SizedDigest: blkid,
				Source:      blk.Replicas[0].KeepService,
			})
			pulls++
			change = changePull
		}
		if bal.Dumper != nil {
			changes = append(changes, fmt.Sprintf("%s:%d=%s,%d", srv.ServiceHost, srv.ServicePort, changeName[change], repl.Mtime))
		}
	}
	if bal.Dumper != nil {
		bal.Dumper.Printf("%s have=%d want=%d %s", blkid, len(blk.Replicas), blk.Desired, strings.Join(changes, " "))
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
	lost, overrep, unref, garbage, underrep, justright blocksNBytes
	desired, current                                   blocksNBytes
	pulls, trashes                                     int
	replHistogram                                      []int
}

func (bal *Balancer) getStatistics() (s balancerStats) {
	s.replHistogram = make([]int, 2)
	bal.BlockStateMap.Apply(func(blkid arvados.SizedDigest, blk *BlockState) {
		surplus := len(blk.Replicas) - blk.Desired
		bytes := blkid.Size()
		switch {
		case len(blk.Replicas) == 0 && blk.Desired > 0:
			s.lost.replicas -= surplus
			s.lost.blocks++
			s.lost.bytes += bytes * int64(-surplus)
		case len(blk.Replicas) < blk.Desired:
			s.underrep.replicas -= surplus
			s.underrep.blocks++
			s.underrep.bytes += bytes * int64(-surplus)
		case len(blk.Replicas) > 0 && blk.Desired == 0:
			counter := &s.garbage
			for _, r := range blk.Replicas {
				if r.Mtime >= bal.MinMtime {
					counter = &s.unref
					break
				}
			}
			counter.replicas += surplus
			counter.blocks++
			counter.bytes += bytes * int64(surplus)
		case len(blk.Replicas) > blk.Desired:
			s.overrep.replicas += surplus
			s.overrep.blocks++
			s.overrep.bytes += bytes * int64(len(blk.Replicas)-blk.Desired)
		default:
			s.justright.replicas += blk.Desired
			s.justright.blocks++
			s.justright.bytes += bytes * int64(blk.Desired)
		}

		if blk.Desired > 0 {
			s.desired.replicas += blk.Desired
			s.desired.blocks++
			s.desired.bytes += bytes * int64(blk.Desired)
		}
		if len(blk.Replicas) > 0 {
			s.current.replicas += len(blk.Replicas)
			s.current.blocks++
			s.current.bytes += bytes * int64(len(blk.Replicas))
		}

		for len(s.replHistogram) <= len(blk.Replicas) {
			s.replHistogram = append(s.replHistogram, 0)
		}
		s.replHistogram[len(blk.Replicas)]++
	})
	for _, srv := range bal.KeepServices {
		s.pulls += len(srv.ChangeSet.Pulls)
		s.trashes += len(srv.ChangeSet.Trashes)
	}
	return
}

// PrintStatistics writes statistics about the computed changes to
// bal.Logger. It should not be called until ComputeChangeSets has
// finished.
func (bal *Balancer) PrintStatistics() {
	s := bal.getStatistics()
	bal.logf("===")
	bal.logf("%s lost (0=have<want)", s.lost)
	bal.logf("%s underreplicated (0<have<want)", s.underrep)
	bal.logf("%s just right (have=want)", s.justright)
	bal.logf("%s overreplicated (have>want>0)", s.overrep)
	bal.logf("%s unreferenced (have>want=0, new)", s.unref)
	bal.logf("%s garbage (have>want=0, old)", s.garbage)
	bal.logf("===")
	bal.logf("%s total commitment (excluding unreferenced)", s.desired)
	bal.logf("%s total usage", s.current)
	bal.logf("===")
	for _, srv := range bal.KeepServices {
		bal.logf("%s: %v\n", srv, srv.ChangeSet)
	}
	bal.logf("===")
	bal.printHistogram(s, 60)
	bal.logf("===")
}

func (bal *Balancer) printHistogram(s balancerStats, hashColumns int) {
	bal.logf("Replication level distribution (counting N replicas on a single server as N):")
	maxCount := 0
	for _, count := range s.replHistogram {
		if maxCount < count {
			maxCount = count
		}
	}
	hashes := strings.Repeat("#", hashColumns)
	countWidth := 1 + int(math.Log10(float64(maxCount+1)))
	scaleCount := 10 * float64(hashColumns) / math.Floor(1+10*math.Log10(float64(maxCount+1)))
	for repl, count := range s.replHistogram {
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
		if blk.Desired > 0 {
			anyDesired = true
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
