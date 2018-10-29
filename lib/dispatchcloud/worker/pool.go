// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"bytes"
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
	Unallocated          time.Time
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

	// private state
	subscribers  map[<-chan struct{}]chan<- struct{}
	creating     map[arvados.InstanceType]int // goroutines waiting for (InstanceSet)Create to return
	workers      map[cloud.InstanceID]*worker
	loaded       bool                 // loaded list of instances from InstanceSet at least once
	exited       map[string]time.Time // containers whose crunch-run proc has exited, but KillContainer has not been called
	atQuotaUntil time.Time
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

type worker struct {
	state       State
	instance    cloud.Instance
	executor    Executor
	instType    arvados.InstanceType
	vcpus       int64
	memory      int64
	booted      bool
	probed      time.Time
	updated     time.Time
	busy        time.Time
	unallocated time.Time
	lastUUID    string
	running     map[string]struct{}
	starting    map[string]struct{}
	probing     chan struct{}
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
	u := map[arvados.InstanceType]int{}
	for it, c := range wp.creating {
		u[it] = c
	}
	for _, wkr := range wp.workers {
		if len(wkr.running)+len(wkr.starting) == 0 && (wkr.state == StateRunning || wkr.state == StateBooting || wkr.state == StateUnknown) {
			u[wkr.instType]++
		}
	}
	return u
}

// Create a new instance with the given type, and add it to the worker
// pool. The worker is added immediately; instance creation runs in
// the background.
func (wp *Pool) Create(it arvados.InstanceType) error {
	logger := wp.logger.WithField("InstanceType", it.Name)
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	tags := cloud.InstanceTags{tagKeyInstanceType: it.Name}
	wp.creating[it]++
	go func() {
		defer wp.notify()
		inst, err := wp.instanceSet.Create(it, wp.imageID, tags, nil)
		wp.mtx.Lock()
		defer wp.mtx.Unlock()
		wp.creating[it]--
		if err, ok := err.(cloud.QuotaError); ok && err.IsQuotaError() {
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
// initialState if a new worker is created. Caller must have lock.
//
// Returns true when a new worker is created.
func (wp *Pool) updateWorker(inst cloud.Instance, it arvados.InstanceType, initialState State) bool {
	id := inst.ID()
	if wp.workers[id] != nil {
		wp.workers[id].executor.SetTarget(inst)
		wp.workers[id].instance = inst
		wp.workers[id].updated = time.Now()
		if initialState == StateBooting && wp.workers[id].state == StateUnknown {
			wp.workers[id].state = StateBooting
		}
		return false
	}
	if initialState == StateUnknown && inst.Tags()[tagKeyHold] != "" {
		initialState = StateHold
	}
	wp.logger.WithFields(logrus.Fields{
		"InstanceType": it.Name,
		"Instance":     inst,
		"State":        initialState,
	}).Infof("instance appeared in cloud")
	now := time.Now()
	wp.workers[id] = &worker{
		executor:    wp.newExecutor(inst),
		state:       initialState,
		instance:    inst,
		instType:    it,
		probed:      now,
		busy:        now,
		updated:     now,
		unallocated: now,
		running:     make(map[string]struct{}),
		starting:    make(map[string]struct{}),
		probing:     make(chan struct{}, 1),
	}
	return true
}

// Shutdown shuts down a worker with the given type, or returns false
// if all workers with the given type are busy.
func (wp *Pool) Shutdown(it arvados.InstanceType) bool {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	logger := wp.logger.WithField("InstanceType", it.Name)
	logger.Info("shutdown requested")
	for _, tryState := range []State{StateBooting, StateRunning} {
		// TODO: shutdown the worker with the longest idle
		// time (Running) or the earliest create time
		// (Booting)
		for _, wkr := range wp.workers {
			if wkr.state != tryState || len(wkr.running)+len(wkr.starting) > 0 {
				continue
			}
			if wkr.instType != it {
				continue
			}
			logger = logger.WithField("Instance", wkr.instance)
			logger.Info("shutting down")
			wp.shutdown(wkr, logger)
			return true
		}
	}
	return false
}

// caller must have lock
func (wp *Pool) shutdown(wkr *worker, logger logrus.FieldLogger) {
	wkr.updated = time.Now()
	wkr.state = StateShutdown
	go func() {
		err := wkr.instance.Destroy()
		if err != nil {
			logger.WithError(err).WithField("Instance", wkr.instance).Warn("shutdown failed")
			return
		}
		wp.mtx.Lock()
		wp.atQuotaUntil = time.Now()
		wp.mtx.Unlock()
		wp.notify()
	}()
}

// Workers returns the current number of workers in each state.
func (wp *Pool) Workers() map[State]int {
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
	logger := wp.logger.WithFields(logrus.Fields{
		"InstanceType":  it.Name,
		"ContainerUUID": ctr.UUID,
		"Priority":      ctr.Priority,
	})
	wp.setupOnce.Do(wp.setup)
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	var wkr *worker
	for _, w := range wp.workers {
		if w.instType == it && w.state == StateRunning && len(w.running)+len(w.starting) == 0 {
			if wkr == nil || w.busy.After(wkr.busy) {
				wkr = w
			}
		}
	}
	if wkr == nil {
		return false
	}
	logger = logger.WithField("Instance", wkr.instance)
	logger.Debug("starting container")
	wkr.starting[ctr.UUID] = struct{}{}
	go func() {
		stdout, stderr, err := wkr.executor.Execute("crunch-run --detach '"+ctr.UUID+"'", nil)
		wp.mtx.Lock()
		defer wp.mtx.Unlock()
		now := time.Now()
		wkr.updated = now
		wkr.busy = now
		delete(wkr.starting, ctr.UUID)
		wkr.running[ctr.UUID] = struct{}{}
		wkr.lastUUID = ctr.UUID
		if err != nil {
			logger.WithField("stdout", string(stdout)).
				WithField("stderr", string(stderr)).
				WithError(err).
				Error("error starting crunch-run process")
			// Leave uuid in wkr.running, though: it's
			// possible the error was just a communication
			// failure and the process was in fact
			// started.  Wait for next probe to find out.
			return
		}
		logger.Info("crunch-run process started")
		wkr.lastUUID = ctr.UUID
	}()
	return true
}

// KillContainer kills the crunch-run process for the given container
// UUID, if it's running on any worker.
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
	stdout, stderr, err := wkr.executor.Execute("crunch-run --kill "+uuid, nil)
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
			if wkr.state == StateShutdown || wp.shutdownIfIdle(wkr) {
				continue
			}
			workers = append(workers, id)
		}
		wp.mtx.Unlock()

		for _, id := range workers {
			wp.mtx.Lock()
			wkr, ok := wp.workers[id]
			wp.mtx.Unlock()
			if !ok || wkr.state == StateShutdown {
				// Deleted/shutdown while we
				// were probing others
				continue
			}
			select {
			case wkr.probing <- struct{}{}:
				go func() {
					wp.probeAndUpdate(wkr)
					<-wkr.probing
				}()
			default:
				wp.logger.WithField("Instance", wkr.instance).Debug("still waiting for last probe to finish")
			}
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

// caller must have lock.
func (wp *Pool) shutdownIfBroken(wkr *worker, dur time.Duration) {
	if wkr.state == StateHold {
		return
	}
	label, threshold := "", wp.timeoutProbe
	if wkr.state == StateBooting {
		label, threshold = "new ", wp.timeoutBooting
	}
	if dur < threshold {
		return
	}
	wp.logger.WithFields(logrus.Fields{
		"Instance": wkr.instance,
		"Duration": dur,
		"Since":    wkr.probed,
		"State":    wkr.state,
	}).Warnf("%sinstance unresponsive, shutting down", label)
	wp.shutdown(wkr, wp.logger)
}

// caller must have lock.
func (wp *Pool) shutdownIfIdle(wkr *worker) bool {
	if len(wkr.running)+len(wkr.starting) > 0 || wkr.state != StateRunning {
		return false
	}
	age := time.Since(wkr.unallocated)
	if age < wp.timeoutIdle {
		return false
	}
	logger := wp.logger.WithFields(logrus.Fields{
		"Age":      age,
		"Instance": wkr.instance,
	})
	logger.Info("shutdown idle worker")
	wp.shutdown(wkr, logger)
	return true
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
			Unallocated:          w.unallocated,
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
	wp.creating = map[arvados.InstanceType]int{}
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
		if wp.updateWorker(inst, it, StateUnknown) {
			notify = true
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

// should be called in a new goroutine
func (wp *Pool) probeAndUpdate(wkr *worker) {
	logger := wp.logger.WithField("Instance", wkr.instance)
	wp.mtx.Lock()
	updated := wkr.updated
	booted := wkr.booted
	wp.mtx.Unlock()

	var (
		ctrUUIDs []string
		ok       bool
		stderr   []byte
	)
	if !booted {
		booted, stderr = wp.probeBooted(wkr)
		wp.mtx.Lock()
		if booted && !wkr.booted {
			wkr.booted = booted
			logger.Info("instance booted")
		} else {
			booted = wkr.booted
		}
		wp.mtx.Unlock()
	}
	if booted {
		ctrUUIDs, ok, stderr = wp.probeRunning(wkr)
	}
	logger = logger.WithField("stderr", string(stderr))
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	if !ok {
		if wkr.state == StateShutdown {
			return
		}
		dur := time.Since(wkr.probed)
		logger := logger.WithFields(logrus.Fields{
			"Duration": dur,
			"State":    wkr.state,
		})
		if wkr.state == StateBooting {
			logger.Debug("new instance not responding")
		} else {
			logger.Info("instance not responding")
		}
		wp.shutdownIfBroken(wkr, dur)
		return
	}

	updateTime := time.Now()
	wkr.probed = updateTime
	if wkr.state == StateShutdown || wkr.state == StateHold {
	} else if booted {
		if wkr.state != StateRunning {
			wkr.state = StateRunning
			go wp.notify()
		}
	} else {
		wkr.state = StateBooting
	}

	if updated != wkr.updated {
		// Worker was updated after the probe began, so
		// wkr.running might have a container UUID that was
		// not yet running when ctrUUIDs was generated. Leave
		// wkr.running alone and wait for the next probe to
		// catch up on any changes.
		return
	}

	if len(ctrUUIDs) > 0 {
		wkr.busy = updateTime
		wkr.lastUUID = ctrUUIDs[0]
	} else if len(wkr.running) > 0 {
		wkr.unallocated = updateTime
	}
	running := map[string]struct{}{}
	changed := false
	for _, uuid := range ctrUUIDs {
		running[uuid] = struct{}{}
		if _, ok := wkr.running[uuid]; !ok {
			changed = true
		}
	}
	for uuid := range wkr.running {
		if _, ok := running[uuid]; !ok {
			logger.WithField("ContainerUUID", uuid).Info("crunch-run process ended")
			wp.exited[uuid] = updateTime
			changed = true
		}
	}
	if changed {
		wkr.running = running
		wkr.updated = updateTime
		go wp.notify()
	}
}

func (wp *Pool) probeRunning(wkr *worker) (running []string, ok bool, stderr []byte) {
	cmd := "crunch-run --list"
	stdout, stderr, err := wkr.executor.Execute(cmd, nil)
	if err != nil {
		wp.logger.WithFields(logrus.Fields{
			"Instance": wkr.instance,
			"Command":  cmd,
			"stdout":   string(stdout),
			"stderr":   string(stderr),
		}).WithError(err).Warn("probe failed")
		return nil, false, stderr
	}
	stdout = bytes.TrimRight(stdout, "\n")
	if len(stdout) == 0 {
		return nil, true, stderr
	}
	return strings.Split(string(stdout), "\n"), true, stderr
}

func (wp *Pool) probeBooted(wkr *worker) (ok bool, stderr []byte) {
	cmd := wp.bootProbeCommand
	if cmd == "" {
		cmd = "true"
	}
	stdout, stderr, err := wkr.executor.Execute(cmd, nil)
	logger := wp.logger.WithFields(logrus.Fields{
		"Instance": wkr.instance,
		"Command":  cmd,
		"stdout":   string(stdout),
		"stderr":   string(stderr),
	})
	if err != nil {
		logger.WithError(err).Debug("boot probe failed")
		return false, stderr
	}
	logger.Info("boot probe succeeded")
	return true, stderr
}
