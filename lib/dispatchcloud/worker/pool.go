// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	mathrand "math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	tagKeyInstanceType   = "InstanceType"
	tagKeyIdleBehavior   = "IdleBehavior"
	tagKeyInstanceSecret = "InstanceSecret"
	tagKeyInstanceSetID  = "InstanceSetID"
)

// An InstanceView shows a worker's current state and recent activity.
type InstanceView struct {
	Instance             cloud.InstanceID `json:"instance"`
	Address              string           `json:"address"`
	Price                float64          `json:"price"`
	ArvadosInstanceType  string           `json:"arvados_instance_type"`
	ProviderInstanceType string           `json:"provider_instance_type"`
	LastContainerUUID    string           `json:"last_container_uuid"`
	LastBusy             time.Time        `json:"last_busy"`
	WorkerState          string           `json:"worker_state"`
	IdleBehavior         IdleBehavior     `json:"idle_behavior"`
}

// An Executor executes shell commands on a remote host.
type Executor interface {
	// Run cmd on the current target.
	Execute(env map[string]string, cmd string, stdin io.Reader) (stdout, stderr []byte, err error)

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
	defaultSyncInterval        = time.Minute
	defaultProbeInterval       = time.Second * 10
	defaultMaxProbesPerSecond  = 10
	defaultTimeoutIdle         = time.Minute
	defaultTimeoutBooting      = time.Minute * 10
	defaultTimeoutProbe        = time.Minute * 10
	defaultTimeoutShutdown     = time.Second * 10
	defaultTimeoutTERM         = time.Minute * 2
	defaultTimeoutSignal       = time.Second * 5
	defaultTimeoutStaleRunLock = time.Second * 5

	// Time after a quota error to try again anyway, even if no
	// instances have been shutdown.
	quotaErrorTTL = time.Minute

	// Time between "X failed because rate limiting" messages
	logRateLimitErrorInterval = time.Second * 10
)

func duration(conf arvados.Duration, def time.Duration) time.Duration {
	if conf > 0 {
		return time.Duration(conf)
	}
	return def
}

// NewPool creates a Pool of workers backed by instanceSet.
//
// New instances are configured and set up according to the given
// cluster configuration.
func NewPool(logger logrus.FieldLogger, arvClient *arvados.Client, reg *prometheus.Registry, instanceSetID cloud.InstanceSetID, instanceSet cloud.InstanceSet, newExecutor func(cloud.Instance) Executor, installPublicKey ssh.PublicKey, cluster *arvados.Cluster) *Pool {
	wp := &Pool{
		logger:                         logger,
		arvClient:                      arvClient,
		instanceSetID:                  instanceSetID,
		instanceSet:                    &throttledInstanceSet{InstanceSet: instanceSet},
		newExecutor:                    newExecutor,
		cluster:                        cluster,
		bootProbeCommand:               cluster.Containers.CloudVMs.BootProbeCommand,
		runnerSource:                   cluster.Containers.CloudVMs.DeployRunnerBinary,
		imageID:                        cloud.ImageID(cluster.Containers.CloudVMs.ImageID),
		instanceTypes:                  cluster.InstanceTypes,
		maxProbesPerSecond:             cluster.Containers.CloudVMs.MaxProbesPerSecond,
		maxConcurrentInstanceCreateOps: cluster.Containers.CloudVMs.MaxConcurrentInstanceCreateOps,
		maxInstances:                   cluster.Containers.CloudVMs.MaxInstances,
		probeInterval:                  duration(cluster.Containers.CloudVMs.ProbeInterval, defaultProbeInterval),
		syncInterval:                   duration(cluster.Containers.CloudVMs.SyncInterval, defaultSyncInterval),
		timeoutIdle:                    duration(cluster.Containers.CloudVMs.TimeoutIdle, defaultTimeoutIdle),
		timeoutBooting:                 duration(cluster.Containers.CloudVMs.TimeoutBooting, defaultTimeoutBooting),
		timeoutProbe:                   duration(cluster.Containers.CloudVMs.TimeoutProbe, defaultTimeoutProbe),
		timeoutShutdown:                duration(cluster.Containers.CloudVMs.TimeoutShutdown, defaultTimeoutShutdown),
		timeoutTERM:                    duration(cluster.Containers.CloudVMs.TimeoutTERM, defaultTimeoutTERM),
		timeoutSignal:                  duration(cluster.Containers.CloudVMs.TimeoutSignal, defaultTimeoutSignal),
		timeoutStaleRunLock:            duration(cluster.Containers.CloudVMs.TimeoutStaleRunLock, defaultTimeoutStaleRunLock),
		systemRootToken:                cluster.SystemRootToken,
		installPublicKey:               installPublicKey,
		tagKeyPrefix:                   cluster.Containers.CloudVMs.TagKeyPrefix,
		runnerCmdDefault:               cluster.Containers.CrunchRunCommand,
		runnerArgs:                     append([]string{"--runtime-engine=" + cluster.Containers.RuntimeEngine}, cluster.Containers.CrunchRunArgumentsList...),
		stop:                           make(chan bool),
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
	logger                         logrus.FieldLogger
	arvClient                      *arvados.Client
	instanceSetID                  cloud.InstanceSetID
	instanceSet                    *throttledInstanceSet
	newExecutor                    func(cloud.Instance) Executor
	cluster                        *arvados.Cluster
	bootProbeCommand               string
	runnerSource                   string
	imageID                        cloud.ImageID
	instanceTypes                  map[string]arvados.InstanceType
	syncInterval                   time.Duration
	probeInterval                  time.Duration
	maxProbesPerSecond             int
	maxConcurrentInstanceCreateOps int
	maxInstances                   int
	timeoutIdle                    time.Duration
	timeoutBooting                 time.Duration
	timeoutProbe                   time.Duration
	timeoutShutdown                time.Duration
	timeoutTERM                    time.Duration
	timeoutSignal                  time.Duration
	timeoutStaleRunLock            time.Duration
	systemRootToken                string
	installPublicKey               ssh.PublicKey
	tagKeyPrefix                   string
	runnerCmdDefault               string   // crunch-run command to use if not deploying a binary
	runnerArgs                     []string // extra args passed to crunch-run

	// private state
	subscribers  map[<-chan struct{}]chan<- struct{}
	creating     map[string]createCall // unfinished (cloud.InstanceSet)Create calls (key is instance secret)
	workers      map[cloud.InstanceID]*worker
	loaded       bool                 // loaded list of instances from InstanceSet at least once
	exited       map[string]time.Time // containers whose crunch-run proc has exited, but ForgetContainer has not been called
	atQuotaUntil time.Time
	atQuotaErr   cloud.QuotaError
	stop         chan bool
	mtx          sync.RWMutex
	setupOnce    sync.Once
	runnerData   []byte
	runnerMD5    [md5.Size]byte
	runnerCmd    string

	mContainersRunning        prometheus.Gauge
	mInstances                *prometheus.GaugeVec
	mInstancesPrice           *prometheus.GaugeVec
	mVCPUs                    *prometheus.GaugeVec
	mMemory                   *prometheus.GaugeVec
	mBootOutcomes             *prometheus.CounterVec
	mDisappearances           *prometheus.CounterVec
	mTimeToSSH                prometheus.Summary
	mTimeToReadyForContainer  prometheus.Summary
	mTimeFromShutdownToGone   prometheus.Summary
	mTimeFromQueueToCrunchRun prometheus.Summary
	mRunProbeDuration         *prometheus.SummaryVec
}

type createCall struct {
	time         time.Time
	instanceType arvados.InstanceType
}

func (wp *Pool) CheckHealth() error {
	wp.setupOnce.Do(wp.setup)
	if err := wp.loadRunnerData(); err != nil {
		return fmt.Errorf("error loading runner binary: %s", err)
	}
	return nil
}

// Subscribe returns a buffered channel that becomes ready after any
// change to the pool's state that could have scheduling implications:
// a worker's state changes, a new worker appears, the cloud
// provider's API rate limiting period ends, etc.
//
// Additional events that occur while the channel is already ready
// will be dropped, so it is OK if the caller services the channel
// slowly.
//
// Example:
//
//	ch := wp.Subscribe()
//	defer wp.Unsubscribe(ch)
//	for range ch {
//		tryScheduling(wp)
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
// idle + unknown) workers for each instance type.  Workers in
// hold/drain mode are not included.
func (wp *Pool) Unallocated() map[arvados.InstanceType]int {
	wp.setupOnce.Do(wp.setup)
	wp.mtx.RLock()
	defer wp.mtx.RUnlock()
	unalloc := map[arvados.InstanceType]int{}
	creating := map[arvados.InstanceType]int{}
	oldestCreate := map[arvados.InstanceType]time.Time{}
	for _, cc := range wp.creating {
		it := cc.instanceType
		creating[it]++
		if t, ok := oldestCreate[it]; !ok || t.After(cc.time) {
			oldestCreate[it] = cc.time
		}
	}
	for _, wkr := range wp.workers {
		// Skip workers that are not expected to become
		// available soon. Note len(wkr.running)>0 is not
		// redundant here: it can be true even in
		// StateUnknown.
		if wkr.state == StateShutdown ||
			wkr.state == StateRunning ||
			wkr.idleBehavior != IdleBehaviorRun ||
			len(wkr.running) > 0 {
			continue
		}
		it := wkr.instType
		unalloc[it]++
		if wkr.state == StateUnknown && creating[it] > 0 && wkr.appeared.After(oldestCreate[it]) {
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
//
// Create returns false if a pre-existing error or a configuration
// setting prevents it from even attempting to create a new
// instance. Those errors are logged by the Pool, so the caller does
// not need to log anything in such cases.
func (wp *Pool) Create(it arvados.InstanceType) bool {
	logger := wp.logger.WithField("InstanceType", it.Name)
	wp.setupOnce.Do(wp.setup)
	if wp.loadRunnerData() != nil {
		// Boot probe is certain to fail.
		return false
	}
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	if time.Now().Before(wp.atQuotaUntil) ||
		wp.instanceSet.throttleCreate.Error() != nil ||
		(wp.maxInstances > 0 && wp.maxInstances <= len(wp.workers)+len(wp.creating)) {
		return false
	}
	// The maxConcurrentInstanceCreateOps knob throttles the number of node create
	// requests in flight. It was added to work around a limitation in Azure's
	// managed disks, which support no more than 20 concurrent node creation
	// requests from a single disk image (cf.
	// https://docs.microsoft.com/en-us/azure/virtual-machines/linux/capture-image).
	// The code assumes that node creation, from Azure's perspective, means the
	// period until the instance appears in the "get all instances" list.
	if wp.maxConcurrentInstanceCreateOps > 0 && len(wp.creating) >= wp.maxConcurrentInstanceCreateOps {
		logger.Info("reached MaxConcurrentInstanceCreateOps")
		wp.instanceSet.throttleCreate.ErrorUntil(errors.New("reached MaxConcurrentInstanceCreateOps"), time.Now().Add(5*time.Second), wp.notify)
		return false
	}
	now := time.Now()
	secret := randomHex(instanceSecretLength)
	wp.creating[secret] = createCall{time: now, instanceType: it}
	go func() {
		defer wp.notify()
		tags := cloud.InstanceTags{
			wp.tagKeyPrefix + tagKeyInstanceSetID:  string(wp.instanceSetID),
			wp.tagKeyPrefix + tagKeyInstanceType:   it.Name,
			wp.tagKeyPrefix + tagKeyIdleBehavior:   string(IdleBehaviorRun),
			wp.tagKeyPrefix + tagKeyInstanceSecret: secret,
		}
		initCmd := TagVerifier{nil, secret, nil}.InitCommand()
		inst, err := wp.instanceSet.Create(it, wp.imageID, tags, initCmd, wp.installPublicKey)
		wp.mtx.Lock()
		defer wp.mtx.Unlock()
		// delete() is deferred so the updateWorker() call
		// below knows to use StateBooting when adding a new
		// worker.
		defer delete(wp.creating, secret)
		if err != nil {
			if err, ok := err.(cloud.QuotaError); ok && err.IsQuotaError() {
				wp.atQuotaErr = err
				wp.atQuotaUntil = time.Now().Add(quotaErrorTTL)
				time.AfterFunc(quotaErrorTTL, wp.notify)
			}
			logger.WithError(err).Error("create failed")
			wp.instanceSet.throttleCreate.CheckRateLimitError(err, wp.logger, "create instance", wp.notify)
			return
		}
		wp.updateWorker(inst, it)
	}()
	if len(wp.creating)+len(wp.workers) == wp.maxInstances {
		logger.Infof("now at MaxInstances limit of %d instances", wp.maxInstances)
	}
	return true
}

// AtQuota returns true if Create is not expected to work at the
// moment (e.g., cloud provider has reported quota errors, or we are
// already at our own configured quota).
func (wp *Pool) AtQuota() bool {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	return time.Now().Before(wp.atQuotaUntil) || (wp.maxInstances > 0 && wp.maxInstances <= len(wp.workers)+len(wp.creating))
}

// SetIdleBehavior determines how the indicated instance will behave
// when it has no containers running.
func (wp *Pool) SetIdleBehavior(id cloud.InstanceID, idleBehavior IdleBehavior) error {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	wkr, ok := wp.workers[id]
	if !ok {
		return errors.New("requested instance does not exist")
	}
	wkr.setIdleBehavior(idleBehavior)
	return nil
}

// Successful connection to the SSH daemon, update the mTimeToSSH metric
func (wp *Pool) reportSSHConnected(inst cloud.Instance) {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	wkr, ok := wp.workers[inst.ID()]
	if !ok {
		// race: inst was removed from the pool
		return
	}
	if wkr.state != StateBooting || !wkr.firstSSHConnection.IsZero() {
		// the node is not in booting state (can happen if
		// a-d-c is restarted) OR this is not the first SSH
		// connection
		return
	}

	wkr.firstSSHConnection = time.Now()
	if wp.mTimeToSSH != nil {
		wp.mTimeToSSH.Observe(wkr.firstSSHConnection.Sub(wkr.appeared).Seconds())
	}
}

// Add or update worker attached to the given instance.
//
// The second return value is true if a new worker is created.
//
// A newly added instance has state=StateBooting if its tags match an
// entry in wp.creating, otherwise StateUnknown.
//
// Caller must have lock.
func (wp *Pool) updateWorker(inst cloud.Instance, it arvados.InstanceType) (*worker, bool) {
	secret := inst.Tags()[wp.tagKeyPrefix+tagKeyInstanceSecret]
	inst = TagVerifier{Instance: inst, Secret: secret, ReportVerified: wp.reportSSHConnected}
	id := inst.ID()
	if wkr := wp.workers[id]; wkr != nil {
		wkr.executor.SetTarget(inst)
		wkr.instance = inst
		wkr.updated = time.Now()
		wkr.saveTags()
		return wkr, false
	}

	state := StateUnknown
	if _, ok := wp.creating[secret]; ok {
		state = StateBooting
	}

	// If an instance has a valid IdleBehavior tag when it first
	// appears, initialize the new worker accordingly (this is how
	// we restore IdleBehavior that was set by a prior dispatch
	// process); otherwise, default to "run". After this,
	// wkr.idleBehavior is the source of truth, and will only be
	// changed via SetIdleBehavior().
	idleBehavior := IdleBehavior(inst.Tags()[wp.tagKeyPrefix+tagKeyIdleBehavior])
	if !validIdleBehavior[idleBehavior] {
		idleBehavior = IdleBehaviorRun
	}

	logger := wp.logger.WithFields(logrus.Fields{
		"InstanceType": it.Name,
		"Instance":     inst.ID(),
		"Address":      inst.Address(),
	})
	logger.WithFields(logrus.Fields{
		"State":        state,
		"IdleBehavior": idleBehavior,
	}).Infof("instance appeared in cloud")
	now := time.Now()
	wkr := &worker{
		mtx:          &wp.mtx,
		wp:           wp,
		logger:       logger,
		executor:     wp.newExecutor(inst),
		state:        state,
		idleBehavior: idleBehavior,
		instance:     inst,
		instType:     it,
		appeared:     now,
		probed:       now,
		busy:         now,
		updated:      now,
		running:      make(map[string]*remoteRunner),
		starting:     make(map[string]*remoteRunner),
		probing:      make(chan struct{}, 1),
	}
	wp.workers[id] = wkr
	return wkr, true
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
			if wkr.idleBehavior != IdleBehaviorHold && wkr.state == tryState && wkr.instType == it {
				logger.WithField("Instance", wkr.instance.ID()).Info("shutting down")
				wkr.reportBootOutcome(BootOutcomeAborted)
				wkr.shutdown()
				return true
			}
		}
	}
	return false
}

// CountWorkers returns the current number of workers in each state.
//
// CountWorkers blocks, if necessary, until the initial instance list
// has been loaded from the cloud provider.
func (wp *Pool) CountWorkers() map[State]int {
	wp.setupOnce.Do(wp.setup)
	wp.waitUntilLoaded()
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	r := map[State]int{}
	for _, w := range wp.workers {
		r[w.state]++
	}
	return r
}

// Running returns the container UUIDs being prepared/run on workers.
//
// In the returned map, the time value indicates when the Pool
// observed that the container process had exited. A container that
// has not yet exited has a zero time value. The caller should use
// ForgetContainer() to garbage-collect the entries for exited
// containers.
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
		if w.instType == it && w.state == StateIdle && w.idleBehavior == IdleBehaviorRun {
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
//
// KillContainer returns false if the container has already ended.
func (wp *Pool) KillContainer(uuid string, reason string) bool {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	logger := wp.logger.WithFields(logrus.Fields{
		"ContainerUUID": uuid,
		"Reason":        reason,
	})
	for _, wkr := range wp.workers {
		rr := wkr.running[uuid]
		if rr == nil {
			rr = wkr.starting[uuid]
		}
		if rr != nil {
			rr.Kill(reason)
			return true
		}
	}
	logger.Debug("cannot kill: already disappeared")
	return false
}

// ForgetContainer clears the placeholder for the given exited
// container, so it isn't returned by subsequent calls to Running().
//
// ForgetContainer has no effect if the container has not yet exited.
//
// The "container exited at time T" placeholder (which necessitates
// ForgetContainer) exists to make it easier for the caller
// (scheduler) to distinguish a container that exited without
// finalizing its state from a container that exited too recently for
// its final state to have appeared in the scheduler's queue cache.
func (wp *Pool) ForgetContainer(uuid string) {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	if _, ok := wp.exited[uuid]; ok {
		wp.logger.WithField("ContainerUUID", uuid).Debug("clearing placeholder for exited crunch-run process")
		delete(wp.exited, uuid)
	}
}

func (wp *Pool) registerMetrics(reg *prometheus.Registry) {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	wp.mContainersRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "containers_running",
		Help:      "Number of containers reported running by cloud VMs.",
	})
	reg.MustRegister(wp.mContainersRunning)
	wp.mInstances = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "instances_total",
		Help:      "Number of cloud VMs.",
	}, []string{"category", "instance_type"})
	reg.MustRegister(wp.mInstances)
	wp.mInstancesPrice = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "instances_price",
		Help:      "Price of cloud VMs.",
	}, []string{"category"})
	reg.MustRegister(wp.mInstancesPrice)
	wp.mVCPUs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "vcpus_total",
		Help:      "Total VCPUs on all cloud VMs.",
	}, []string{"category"})
	reg.MustRegister(wp.mVCPUs)
	wp.mMemory = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "memory_bytes_total",
		Help:      "Total memory on all cloud VMs.",
	}, []string{"category"})
	reg.MustRegister(wp.mMemory)
	wp.mBootOutcomes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "boot_outcomes",
		Help:      "Boot outcomes by type.",
	}, []string{"outcome"})
	for k := range validBootOutcomes {
		wp.mBootOutcomes.WithLabelValues(string(k)).Add(0)
	}
	reg.MustRegister(wp.mBootOutcomes)
	wp.mDisappearances = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "instances_disappeared",
		Help:      "Number of occurrences of an instance disappearing from the cloud provider's list of instances.",
	}, []string{"state"})
	for _, v := range stateString {
		wp.mDisappearances.WithLabelValues(v).Add(0)
	}
	reg.MustRegister(wp.mDisappearances)
	wp.mTimeToSSH = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  "arvados",
		Subsystem:  "dispatchcloud",
		Name:       "instances_time_to_ssh_seconds",
		Help:       "Number of seconds between instance creation and the first successful SSH connection.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	})
	reg.MustRegister(wp.mTimeToSSH)
	wp.mTimeToReadyForContainer = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  "arvados",
		Subsystem:  "dispatchcloud",
		Name:       "instances_time_to_ready_for_container_seconds",
		Help:       "Number of seconds between the first successful SSH connection and ready to run a container.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	})
	reg.MustRegister(wp.mTimeToReadyForContainer)
	wp.mTimeFromShutdownToGone = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  "arvados",
		Subsystem:  "dispatchcloud",
		Name:       "instances_time_from_shutdown_request_to_disappearance_seconds",
		Help:       "Number of seconds between the first shutdown attempt and the disappearance of the worker.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	})
	reg.MustRegister(wp.mTimeFromShutdownToGone)
	wp.mTimeFromQueueToCrunchRun = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  "arvados",
		Subsystem:  "dispatchcloud",
		Name:       "containers_time_from_queue_to_crunch_run_seconds",
		Help:       "Number of seconds between the queuing of a container and the start of crunch-run.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	})
	reg.MustRegister(wp.mTimeFromQueueToCrunchRun)
	wp.mRunProbeDuration = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  "arvados",
		Subsystem:  "dispatchcloud",
		Name:       "instances_run_probe_duration_seconds",
		Help:       "Number of seconds per runProbe call.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
	}, []string{"outcome"})
	reg.MustRegister(wp.mRunProbeDuration)
}

func (wp *Pool) runMetrics() {
	ch := wp.Subscribe()
	defer wp.Unsubscribe(ch)
	wp.updateMetrics()
	for range ch {
		wp.updateMetrics()
	}
}

func (wp *Pool) updateMetrics() {
	wp.mtx.RLock()
	defer wp.mtx.RUnlock()

	type entKey struct {
		cat      string
		instType string
	}
	instances := map[entKey]int64{}
	price := map[string]float64{}
	cpu := map[string]int64{}
	mem := map[string]int64{}
	var running int64
	for _, wkr := range wp.workers {
		var cat string
		switch {
		case len(wkr.running)+len(wkr.starting) > 0:
			cat = "inuse"
		case wkr.idleBehavior == IdleBehaviorHold:
			cat = "hold"
		case wkr.state == StateBooting:
			cat = "booting"
		case wkr.state == StateUnknown:
			cat = "unknown"
		default:
			cat = "idle"
		}
		instances[entKey{cat, wkr.instType.Name}]++
		price[cat] += wkr.instType.Price
		cpu[cat] += int64(wkr.instType.VCPUs)
		mem[cat] += int64(wkr.instType.RAM)
		running += int64(len(wkr.running) + len(wkr.starting))
	}
	for _, cat := range []string{"inuse", "hold", "booting", "unknown", "idle"} {
		wp.mInstancesPrice.WithLabelValues(cat).Set(price[cat])
		wp.mVCPUs.WithLabelValues(cat).Set(float64(cpu[cat]))
		wp.mMemory.WithLabelValues(cat).Set(float64(mem[cat]))
		// make sure to reset gauges for non-existing category/nodetype combinations
		for _, it := range wp.instanceTypes {
			if _, ok := instances[entKey{cat, it.Name}]; !ok {
				wp.mInstances.WithLabelValues(cat, it.Name).Set(float64(0))
			}
		}
	}
	for k, v := range instances {
		wp.mInstances.WithLabelValues(k.cat, k.instType).Set(float64(v))
	}
	wp.mContainersRunning.Set(float64(running))
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
		// Add some jitter. Without this, if probeInterval is
		// a multiple of syncInterval and sync is
		// instantaneous (as with the loopback driver), the
		// first few probes race with sync operations and
		// don't update the workers.
		time.Sleep(time.Duration(mathrand.Int63n(int64(wp.probeInterval) / 23)))

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
			Instance:             w.instance.ID(),
			Address:              w.instance.Address(),
			Price:                w.instType.Price,
			ArvadosInstanceType:  w.instType.Name,
			ProviderInstanceType: w.instType.ProviderType,
			LastContainerUUID:    w.lastUUID,
			LastBusy:             w.busy,
			WorkerState:          w.state.String(),
			IdleBehavior:         w.idleBehavior,
		})
	}
	wp.mtx.Unlock()
	sort.Slice(r, func(i, j int) bool {
		return strings.Compare(string(r[i].Instance), string(r[j].Instance)) < 0
	})
	return r
}

// KillInstance destroys a cloud VM instance. It returns an error if
// the given instance does not exist.
func (wp *Pool) KillInstance(id cloud.InstanceID, reason string) error {
	wkr, ok := wp.workers[id]
	if !ok {
		return errors.New("instance not found")
	}
	wkr.logger.WithField("Reason", reason).Info("shutting down")
	wkr.reportBootOutcome(BootOutcomeAborted)
	wkr.shutdown()
	return nil
}

func (wp *Pool) setup() {
	wp.creating = map[string]createCall{}
	wp.exited = map[string]time.Time{}
	wp.workers = map[cloud.InstanceID]*worker{}
	wp.subscribers = map[<-chan struct{}]chan<- struct{}{}
	wp.loadRunnerData()
}

// Load the runner program to be deployed on worker nodes into
// wp.runnerData, if necessary. Errors are logged.
//
// If auto-deploy is disabled, len(wp.runnerData) will be 0.
//
// Caller must not have lock.
func (wp *Pool) loadRunnerData() error {
	wp.mtx.Lock()
	defer wp.mtx.Unlock()
	if wp.runnerData != nil {
		return nil
	} else if wp.runnerSource == "" {
		wp.runnerCmd = wp.runnerCmdDefault
		wp.runnerData = []byte{}
		return nil
	}
	logger := wp.logger.WithField("source", wp.runnerSource)
	logger.Debug("loading runner")
	buf, err := ioutil.ReadFile(wp.runnerSource)
	if err != nil {
		logger.WithError(err).Error("failed to load runner program")
		return err
	}
	wp.runnerData = buf
	wp.runnerMD5 = md5.Sum(buf)
	wp.runnerCmd = fmt.Sprintf("/tmp/arvados-crunch-run/crunch-run~%x", wp.runnerMD5)
	return nil
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
	if err := wp.instanceSet.throttleInstances.Error(); err != nil {
		return err
	}
	wp.logger.Debug("getting instance list")
	threshold := time.Now()
	instances, err := wp.instanceSet.Instances(cloud.InstanceTags{wp.tagKeyPrefix + tagKeyInstanceSetID: string(wp.instanceSetID)})
	if err != nil {
		wp.instanceSet.throttleInstances.CheckRateLimitError(err, wp.logger, "list instances", wp.notify)
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
		itTag := inst.Tags()[wp.tagKeyPrefix+tagKeyInstanceType]
		it, ok := wp.instanceTypes[itTag]
		if !ok {
			wp.logger.WithField("Instance", inst.ID()).Errorf("unknown InstanceType tag %q --- ignoring", itTag)
			continue
		}
		if wkr, isNew := wp.updateWorker(inst, it); isNew {
			notify = true
		} else if wkr.state == StateShutdown && time.Since(wkr.destroyed) > wp.timeoutShutdown {
			wp.logger.WithField("Instance", inst.ID()).Info("worker still listed after shutdown; retrying")
			wkr.shutdown()
		}
	}

	for id, wkr := range wp.workers {
		if wkr.updated.After(threshold) {
			continue
		}
		logger := wp.logger.WithFields(logrus.Fields{
			"Instance":    wkr.instance.ID(),
			"WorkerState": wkr.state,
		})
		logger.Info("instance disappeared in cloud")
		wkr.reportBootOutcome(BootOutcomeDisappeared)
		if wp.mDisappearances != nil {
			wp.mDisappearances.WithLabelValues(stateString[wkr.state]).Inc()
		}
		// wkr.destroyed.IsZero() can happen if instance disappeared but we weren't trying to shut it down
		if wp.mTimeFromShutdownToGone != nil && !wkr.destroyed.IsZero() {
			wp.mTimeFromShutdownToGone.Observe(time.Now().Sub(wkr.destroyed).Seconds())
		}
		delete(wp.workers, id)
		go wkr.Close()
		notify = true
	}

	if !wp.loaded {
		notify = true
		wp.loaded = true
		wp.logger.WithField("N", len(wp.workers)).Info("loaded initial instance list")
	}

	if notify {
		go wp.notify()
	}
}

func (wp *Pool) waitUntilLoaded() {
	ch := wp.Subscribe()
	wp.mtx.RLock()
	defer wp.mtx.RUnlock()
	for !wp.loaded {
		wp.mtx.RUnlock()
		<-ch
		wp.mtx.RLock()
	}
}

func (wp *Pool) gatewayAuthSecret(uuid string) string {
	h := hmac.New(sha256.New, []byte(wp.systemRootToken))
	fmt.Fprint(h, uuid)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Return a random string of n hexadecimal digits (n*4 random bits). n
// must be even.
func randomHex(n int) string {
	buf := make([]byte, n/2)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf)
}
