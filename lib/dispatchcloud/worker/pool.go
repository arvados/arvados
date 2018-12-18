// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	tagKeyInstanceType = "InstanceType"
	tagKeyHold         = "Hold"
)

// An InstanceView shows a worker's current state and recent activity.
type InstanceView struct {
	Instance             string
	Price                float64
	ArvadosInstanceType  string
	ProviderInstanceType string
	LastContainerUUID    string
	LastBusy             time.Time
	WorkerState          string
}

// An Executor executes shell commands on a remote host.
type Executor interface {
	// Run cmd on the current target.
	Execute(cmd string, stdin io.Reader) (stdout, stderr []byte, err error)

	// Use the given target for subsequent operations. The new
	// target is the same host as the previous target, but it
	// might return a different address and verify a different
	// host key.
	//
	// SetTarget is called frequently, and in most cases the new
	// target will behave exactly the same as the old one. An
	// implementation should optimize accordingly.
	//
	// SetTarget must not block on concurrent Execute calls.
	SetTarget(cloud.ExecutorTarget)

	Close()
}

const (
	defaultSyncInterval       = time.Minute
	defaultProbeInterval      = time.Second * 10
	defaultMaxProbesPerSecond = 10
	defaultTimeoutIdle        = time.Minute
	defaultTimeoutBooting     = time.Minute * 10
	defaultTimeoutProbe       = time.Minute * 10
	defaultTimeoutShutdown    = time.Second * 10
)

func duration(conf arvados.Duration, def time.Duration) time.Duration {
	if conf > 0 {
		return time.Duration(conf)
	} else {
		return def
	}
}

// NewPool creates a Pool of workers backed by instanceSet.
//
// New instances are configured and set up according to the given
// cluster configuration.
func NewPool(logger logrus.FieldLogger, reg *prometheus.Registry, instanceSet cloud.InstanceSet, newExecutor func(cloud.Instance) Executor, cluster *arvados.Cluster) *Pool {
	wp := &Pool{
		logger:             logger,
		instanceSet:        instanceSet,
		newExecutor:        newExecutor,
		bootProbeCommand:   cluster.CloudVMs.BootProbeCommand,
		imageID:            cloud.ImageID(cluster.CloudVMs.ImageID),
		instanceTypes:      cluster.InstanceTypes,
		maxProbesPerSecond: cluster.Dispatch.MaxProbesPerSecond,
		probeInterval:      duration(cluster.Dispatch.ProbeInterval, defaultProbeInterval),
		syncInterval:       duration(cluster.CloudVMs.SyncInterval, defaultSyncInterval),
		timeoutIdle:        duration(cluster.CloudVMs.TimeoutIdle, defaultTimeoutIdle),
		timeoutBooting:     duration(cluster.CloudVMs.TimeoutBooting, defaultTimeoutBooting),
		timeoutProbe:       duration(cluster.CloudVMs.TimeoutProbe, defaultTimeoutProbe),
		timeoutShutdown:    duration(cluster.CloudVMs.TimeoutShutdown, defaultTimeoutShutdown),
	}
	wp.registerMetrics(reg)
	go func() {
		wp.setupOnce.Do(wp.setup)
		go wp.runMetrics()
		go wp.runProbes()
		go wp.runSync()
	}()
	return wp
}

// Pool is a resizable worker pool backed by a cloud.InstanceSet. A
// zero Pool should not be used. Call NewPool to create a new Pool.
type Pool struct {
	// configuration
	logger             logrus.FieldLogger
	instanceSet        cloud.InstanceSet
	newExecutor        func(cloud.Instance) Executor
	bootProbeCommand   string
	imageID            cloud.ImageID
	instanceTypes      map[string]arvados.InstanceType
	syncInterval       time.Duration
	probeInterval      time.Duration
	maxProbesPerSecond int
	timeoutIdle        time.Duration
	timeoutBooting     time.Duration
	timeoutProbe       time.Duration
	timeoutShutdown    time.Duration

	// private state
	subscribers  map[<-chan struct{}]chan<- struct{}
	creating     map[arvados.InstanceType][]time.Time // start times of unfinished (InstanceSet)Create calls
	workers      map[cloud.InstanceID]*worker
	loaded       bool                 // loaded list of instances from InstanceSet at least once
	exited       map[string]time.Time // containers whose crunch-run proc has exited, but KillContainer has not been called
	atQuotaUntil time.Time
	atQuotaErr   cloud.QuotaError
	stop         chan bool
	mtx          sync.RWMutex
	setupOnce    sync.Once

	mInstances         prometheus.Gauge
	mContainersRunning prometheus.Gauge
	mVCPUs             prometheus.Gauge
	mVCPUsInuse        prometheus.Gauge
	mMemory            prometheus.Gauge
	mMemoryInuse       prometheus.Gauge
}

// Subscribe returns a channel that becomes ready whenever a worker's
// state changes.
//
// Example:
//
//	ch := wp.Subscribe()
//	defer wp.Unsubscribe(ch)
//	for range ch {
//		// ...try scheduling some work...
//		if done {
//			break
//		}
//	}
func (wp *Pool) Subscribe() <-chan struct{} {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	ch := make(chan struct{}, 1)
	wp.subscribers[ch] = ch
	return ch
}

// Unsubscribe stops sending updates to the given channel.
func (wp *Pool) Unsubscribe(ch <-chan struct{}) {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	delete(wp.subscribers, ch)
}

// Unallocated returns the number of unallocated (creating + booting +
// idle + unknown) workers for each instance type.
func (wp *Pool) Unallocated() map[arvados.InstanceType]int {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.RLock()
	defer wp.mtx.RUnlock()
	unalloc := map[arvados.InstanceType]int{}
	creating := map[arvados.InstanceType]int{}
	for it, times := range wp.creating {
		creating[it] = len(times)
	}
	for _, wkr := range wp.workers {
		if !(wkr.state == StateIdle || wkr.state == StateBooting || wkr.state == StateUnknown) {
			continue
		}
		it := wkr.instType
		unalloc[it]++
		if wkr.state == StateUnknown && creating[it] > 0 && wkr.appeared.After(wp.creating[it][0]) {
			// If up to N new workers appear in
			// Instances() while we are waiting for N
			// Create() calls to complete, we assume we're
			// just seeing a race between Instances() and
			// Create() responses.
			//
			// The other common reason why nodes have
			// state==Unknown is that they appeared at
			// startup, before any Create calls. They
			// don't match the above timing condition, so
			// we never mistakenly attribute them to
			// pending Create calls.
			creating[it]--
		}
	}
	for it, c := range creating {
		unalloc[it] += c
	}
	return unalloc
}

// Create a new instance with the given type, and add it to the worker
// pool. The worker is added immediately; instance creation runs in
// the background.
func (wp *Pool) Create(it arvados.InstanceType) error {
	logger := wp.logger.WithField("InstanceType", it.Name)
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	if time.Now().Before(wp.atQuotaUntil) {
		return wp.atQuotaErr
	}
	tags := cloud.InstanceTags{tagKeyInstanceType: it.Name}
	now := time.Now()
	wp.creating[it] = append(wp.creating[it], now)
	go func() {
		defer wp.notify()
		inst, err := wp.instanceSet.Create(it, wp.imageID, tags, nil)
		wp.mtx.Lock()
		defer wp.mtx.Unlock()
		// Remove our timestamp marker from wp.creating
		for i, t := range wp.creating[it] {
			if t == now {
				copy(wp.creating[it][i:], wp.creating[it][i+1:])
				wp.creating[it] = wp.creating[it][:len(wp.creating[it])-1]
				break
			}
		}
		if err, ok := err.(cloud.QuotaError); ok && err.IsQuotaError() {
			wp.atQuotaErr = err
			wp.atQuotaUntil = time.Now().Add(time.Minute)
		}
		if err != nil {
			logger.WithError(err).Error("create failed")
			return
		}
		wp.updateWorker(inst, it, StateBooting)
	}()
	return nil
}

// AtQuota returns true if Create is not expected to work at the
// moment.
func (wp *Pool) AtQuota() bool {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	return time.Now().Before(wp.atQuotaUntil)
}

// Add or update worker attached to the given instance. Use
// initialState if a new worker is created.
//
// The second return value is true if a new worker is created.
//
// Caller must have lock.
func (wp *Pool) updateWorker(inst cloud.Instance, it arvados.InstanceType, initialState State) (*worker, bool) {
	id := inst.ID()
	if wkr := wp.workers[id]; wkr != nil {
		wkr.executor.SetTarget(inst)
		wkr.instance = inst
		wkr.updated = time.Now()
		if initialState == StateBooting && wkr.state == StateUnknown {
			wkr.state = StateBooting
		}
		return wkr, false
	}
	if initialState == StateUnknown && inst.Tags()[tagKeyHold] != "" {
		initialState = StateHold
	}
	logger := wp.logger.WithFields(logrus.Fields{
		"InstanceType": it.Name,
		"Instance":     inst,
	})
	logger.WithField("State", initialState).Infof("instance appeared in cloud")
	now := time.Now()
	wkr := &worker{
		mtx:      &wp.mtx,
		wp:       wp,
		logger:   logger,
		executor: wp.newExecutor(inst),
		state:    initialState,
		instance: inst,
		instType: it,
		appeared: now,
		probed:   now,
		busy:     now,
		updated:  now,
		running:  make(map[string]struct{}),
		starting: make(map[string]struct{}),
		probing:  make(chan struct{}, 1),
	}
	wp.workers[id] = wkr
	return wkr, true
}

// caller must have lock.
func (wp *Pool) notifyExited(uuid string, t time.Time) {
	wp.exited[uuid] = t
}

// Shutdown shuts down a worker with the given type, or returns false
// if all workers with the given type are busy.
func (wp *Pool) Shutdown(it arvados.InstanceType) bool {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	logger := wp.logger.WithField("InstanceType", it.Name)
	logger.Info("shutdown requested")
	for _, tryState := range []State{StateBooting, StateIdle} {
		// TODO: shutdown the worker with the longest idle
		// time (Idle) or the earliest create time (Booting)
		for _, wkr := range wp.workers {
			if wkr.state == tryState && wkr.instType == it {
				logger.WithField("Instance", wkr.instance).Info("shutting down")
				wkr.shutdown()
				return true
			}
		}
	}
	return false
}

// CountWorkers returns the current number of workers in each state.
func (wp *Pool) CountWorkers() map[State]int {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	r := map[State]int{}
	for _, w := range wp.workers {
		r[w.state]++
	}
	return r
}

// Running returns the container UUIDs being prepared/run on workers.
func (wp *Pool) Running() map[string]time.Time {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	r := map[string]time.Time{}
	for _, wkr := range wp.workers {
		for uuid := range wkr.running {
			r[uuid] = time.Time{}
		}
		for uuid := range wkr.starting {
			r[uuid] = time.Time{}
		}
	}
	for uuid, exited := range wp.exited {
		r[uuid] = exited
	}
	return r
}

// StartContainer starts a container on an idle worker immediately if
// possible, otherwise returns false.
func (wp *Pool) StartContainer(it arvados.InstanceType, ctr arvados.Container) bool {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	var wkr *worker
	for _, w := range wp.workers {
		if w.instType == it && w.state == StateIdle {
			if wkr == nil || w.busy.After(wkr.busy) {
				wkr = w
			}
		}
	}
	if wkr == nil {
		return false
	}
	wkr.startContainer(ctr)
	return true
}

// KillContainer kills the crunch-run process for the given container
// UUID, if it's running on any worker.
//
// KillContainer returns immediately; the act of killing the container
// takes some time, and runs in the background.
func (wp *Pool) KillContainer(uuid string) {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	if _, ok := wp.exited[uuid]; ok {
		wp.logger.WithField("ContainerUUID", uuid).Debug("clearing placeholder for exited crunch-run process")
		delete(wp.exited, uuid)
		return
	}
	for _, wkr := range wp.workers {
		if _, ok := wkr.running[uuid]; ok {
			go wp.kill(wkr, uuid)
			return
		}
	}
	wp.logger.WithField("ContainerUUID", uuid).Debug("cannot kill: already disappeared")
}

func (wp *Pool) kill(wkr *worker, uuid string) {
	logger := wp.logger.WithFields(logrus.Fields{
		"ContainerUUID": uuid,
		"Instance":      wkr.instance,
	})
	logger.Debug("killing process")
	stdout, stderr, err := wkr.executor.Execute("crunch-run --kill 15 "+uuid, nil)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"stderr": string(stderr),
			"stdout": string(stdout),
			"error":  err,
		}).Warn("kill failed")
		return
	}
	logger.Debug("killing process succeeded")
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	if _, ok := wkr.running[uuid]; ok {
		delete(wkr.running, uuid)
		if wkr.state == StateRunning && len(wkr.running)+len(wkr.starting) == 0 {
			wkr.state = StateIdle
		}
		wkr.updated = time.Now()
		go wp.notify()
	}
}

func (wp *Pool) registerMetrics(reg *prometheus.Registry) {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	wp.mInstances = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "instances_total",
		Help:      "Number of cloud VMs including pending, booting, running, held, and shutting down.",
	})
	reg.MustRegister(wp.mInstances)
	wp.mContainersRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "containers_running",
		Help:      "Number of containers reported running by cloud VMs.",
	})
	reg.MustRegister(wp.mContainersRunning)

	wp.mVCPUs = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "vcpus_total",
		Help:      "Total VCPUs on all cloud VMs.",
	})
	reg.MustRegister(wp.mVCPUs)
	wp.mVCPUsInuse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "vcpus_inuse",
		Help:      "VCPUs on cloud VMs that are running containers.",
	})
	reg.MustRegister(wp.mVCPUsInuse)
	wp.mMemory = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "memory_bytes_total",
		Help:      "Total memory on all cloud VMs.",
	})
	reg.MustRegister(wp.mMemory)
	wp.mMemoryInuse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "memory_bytes_inuse",
		Help:      "Memory on cloud VMs that are running containers.",
	})
	reg.MustRegister(wp.mMemoryInuse)
}

func (wp *Pool) runMetrics() {
	ch := wp.Subscribe()
	defer wp.Unsubscribe(ch)
	for range ch {
		wp.updateMetrics()
	}
}

func (wp *Pool) updateMetrics() {
	wp.mtx.RLock()
	defer wp.mtx.RUnlock()

	var alloc, cpu, cpuInuse, mem, memInuse int64
	for _, wkr := range wp.workers {
		cpu += int64(wkr.instType.VCPUs)
		mem += int64(wkr.instType.RAM)
		if len(wkr.running)+len(wkr.starting) == 0 {
			continue
		}
		alloc += int64(len(wkr.running) + len(wkr.starting))
		cpuInuse += int64(wkr.instType.VCPUs)
		memInuse += int64(wkr.instType.RAM)
	}
	wp.mInstances.Set(float64(len(wp.workers)))
	wp.mContainersRunning.Set(float64(alloc))
	wp.mVCPUs.Set(float64(cpu))
	wp.mMemory.Set(float64(mem))
	wp.mVCPUsInuse.Set(float64(cpuInuse))
	wp.mMemoryInuse.Set(float64(memInuse))
}

func (wp *Pool) runProbes() {
	maxPPS := wp.maxProbesPerSecond
	if maxPPS < 1 {
		maxPPS = defaultMaxProbesPerSecond
	}
	limitticker := time.NewTicker(time.Second / time.Duration(maxPPS))
	defer limitticker.Stop()

	probeticker := time.NewTicker(wp.probeInterval)
	defer probeticker.Stop()

	workers := []cloud.InstanceID{}
	for range probeticker.C {
		workers = workers[:0]
		wp.mtx.Lock()
		for id, wkr := range wp.workers {
			if wkr.state == StateShutdown || wkr.shutdownIfIdle() {
				continue
			}
			workers = append(workers, id)
		}
		wp.mtx.Unlock()

		for _, id := range workers {
			wp.mtx.Lock()
			wkr, ok := wp.workers[id]
			wp.mtx.Unlock()
			if !ok {
				// Deleted while we were probing
				// others
				continue
			}
			go wkr.ProbeAndUpdate()
			select {
			case <-wp.stop:
				return
			case <-limitticker.C:
			}
		}
	}
}

func (wp *Pool) runSync() {
	// sync once immediately, then wait syncInterval, sync again,
	// etc.
	timer := time.NewTimer(1)
	for {
		select {
		case <-timer.C:
			err := wp.getInstancesAndSync()
			if err != nil {
				wp.logger.WithError(err).Warn("sync failed")
			}
			timer.Reset(wp.syncInterval)
		case <-wp.stop:
			wp.logger.Debug("worker.Pool stopped")
			return
		}
	}
}

// Stop synchronizing with the InstanceSet.
func (wp *Pool) Stop() {
	wp.setupOnce.Do(wp.setup)
	close(wp.stop)
}

// Instances returns an InstanceView for each worker in the pool,
// summarizing its current state and recent activity.
func (wp *Pool) Instances() []InstanceView {
	var r []InstanceView
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	for _, w := range wp.workers {
		r = append(r, InstanceView{
			Instance:             w.instance.String(),
			Price:                w.instType.Price,
			ArvadosInstanceType:  w.instType.Name,
			ProviderInstanceType: w.instType.ProviderType,
			LastContainerUUID:    w.lastUUID,
			LastBusy:             w.busy,
			WorkerState:          w.state.String(),
		})
	}
	wp.mtx.Unlock()
	sort.Slice(r, func(i, j int) bool {
		return strings.Compare(r[i].Instance, r[j].Instance) < 0
	})
	return r
}

func (wp *Pool) setup() {
	wp.creating = map[arvados.InstanceType][]time.Time{}
	wp.exited = map[string]time.Time{}
	wp.workers = map[cloud.InstanceID]*worker{}
	wp.subscribers = map[<-chan struct{}]chan<- struct{}{}
}

func (wp *Pool) notify() {
	wp.mtx.RLock()
	defer wp.mtx.RUnlock()
	for _, send := range wp.subscribers {
		select {
		case send <- struct{}{}:
		default:
		}
	}
}

func (wp *Pool) getInstancesAndSync() error {
	wp.setupOnce.Do(wp.setup)
	wp.logger.Debug("getting instance list")
	threshold := time.Now()
	instances, err := wp.instanceSet.Instances(cloud.InstanceTags{})
	if err != nil {
		return err
	}
	wp.sync(threshold, instances)
	wp.logger.Debug("sync done")
	return nil
}

// Add/remove/update workers based on instances, which was obtained
// from the instanceSet. However, don't clobber any other updates that
// already happened after threshold.
func (wp *Pool) sync(threshold time.Time, instances []cloud.Instance) {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	wp.logger.WithField("Instances", len(instances)).Debug("sync instances")
	notify := false

	for _, inst := range instances {
		itTag := inst.Tags()[tagKeyInstanceType]
		it, ok := wp.instanceTypes[itTag]
		if !ok {
			wp.logger.WithField("Instance", inst).Errorf("unknown InstanceType tag %q --- ignoring", itTag)
			continue
		}
		if wkr, isNew := wp.updateWorker(inst, it, StateUnknown); isNew {
			notify = true
		} else if wkr.state == StateShutdown && time.Since(wkr.destroyed) > wp.timeoutShutdown {
			wp.logger.WithField("Instance", inst).Info("worker still listed after shutdown; retrying")
			wkr.shutdown()
		}
	}

	for id, wkr := range wp.workers {
		if wkr.updated.After(threshold) {
			continue
		}
		logger := wp.logger.WithFields(logrus.Fields{
			"Instance":    wkr.instance,
			"WorkerState": wkr.state,
		})
		logger.Info("instance disappeared in cloud")
		delete(wp.workers, id)
		go wkr.executor.Close()
		notify = true
	}

	if !wp.loaded {
		wp.loaded = true
		wp.logger.WithField("N", len(wp.workers)).Info("loaded initial instance list")
	}

	if notify {
		go wp.notify()
	}
}
