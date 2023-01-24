// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/stats"
	"github.com/sirupsen/logrus"
)

const (
	// TODO: configurable
	maxPingFailTime = 10 * time.Minute
)

// State indicates whether a worker is available to do work, and (if
// not) whether/when it is expected to become ready.
type State int

const (
	StateUnknown  State = iota // might be running a container already
	StateBooting               // instance is booting
	StateIdle                  // instance booted, no containers are running
	StateRunning               // instance is running one or more containers
	StateShutdown              // worker has stopped monitoring the instance
)

var stateString = map[State]string{
	StateUnknown:  "unknown",
	StateBooting:  "booting",
	StateIdle:     "idle",
	StateRunning:  "running",
	StateShutdown: "shutdown",
}

// String implements fmt.Stringer.
func (s State) String() string {
	return stateString[s]
}

// MarshalText implements encoding.TextMarshaler so a JSON encoding of
// map[State]anything uses the state's string representation.
func (s State) MarshalText() ([]byte, error) {
	return []byte(stateString[s]), nil
}

// BootOutcome is the result of a worker boot. It is used as a label in a metric.
type BootOutcome string

const (
	BootOutcomeFailed      BootOutcome = "failure"
	BootOutcomeSucceeded   BootOutcome = "success"
	BootOutcomeAborted     BootOutcome = "aborted"
	BootOutcomeDisappeared BootOutcome = "disappeared"
)

var validBootOutcomes = map[BootOutcome]bool{
	BootOutcomeFailed:      true,
	BootOutcomeSucceeded:   true,
	BootOutcomeAborted:     true,
	BootOutcomeDisappeared: true,
}

// IdleBehavior indicates the behavior desired when a node becomes idle.
type IdleBehavior string

const (
	IdleBehaviorRun   IdleBehavior = "run"   // run containers, or shutdown on idle timeout
	IdleBehaviorHold  IdleBehavior = "hold"  // don't shutdown or run more containers
	IdleBehaviorDrain IdleBehavior = "drain" // shutdown immediately when idle
)

var validIdleBehavior = map[IdleBehavior]bool{
	IdleBehaviorRun:   true,
	IdleBehaviorHold:  true,
	IdleBehaviorDrain: true,
}

type worker struct {
	logger   logrus.FieldLogger
	executor Executor
	wp       *Pool

	mtx                 sync.Locker // must be wp's Locker.
	state               State
	idleBehavior        IdleBehavior
	instance            cloud.Instance
	instType            arvados.InstanceType
	vcpus               int64
	memory              int64
	appeared            time.Time
	probed              time.Time
	updated             time.Time
	busy                time.Time
	destroyed           time.Time
	firstSSHConnection  time.Time
	lastUUID            string
	running             map[string]*remoteRunner // remember to update state idle<->running when this changes
	starting            map[string]*remoteRunner // remember to update state idle<->running when this changes
	probing             chan struct{}
	bootOutcomeReported bool
	timeToReadyReported bool
	staleRunLockSince   time.Time
}

func (wkr *worker) onUnkillable(uuid string) {
	wkr.mtx.Lock()
	defer wkr.mtx.Unlock()
	logger := wkr.logger.WithField("ContainerUUID", uuid)
	if wkr.idleBehavior == IdleBehaviorHold {
		logger.Warn("unkillable container, but worker has IdleBehavior=Hold")
		return
	}
	logger.Warn("unkillable container, draining worker")
	wkr.setIdleBehavior(IdleBehaviorDrain)
}

func (wkr *worker) onKilled(uuid string) {
	wkr.mtx.Lock()
	defer wkr.mtx.Unlock()
	wkr.closeRunner(uuid)
	go wkr.wp.notify()
}

// caller must have lock.
func (wkr *worker) reportBootOutcome(outcome BootOutcome) {
	if wkr.bootOutcomeReported {
		return
	}
	if wkr.wp.mBootOutcomes != nil {
		wkr.wp.mBootOutcomes.WithLabelValues(string(outcome)).Inc()
	}
	wkr.bootOutcomeReported = true
}

// caller must have lock.
func (wkr *worker) reportTimeBetweenFirstSSHAndReadyForContainer() {
	if wkr.timeToReadyReported {
		return
	}
	if wkr.wp.mTimeToSSH != nil {
		wkr.wp.mTimeToReadyForContainer.Observe(time.Since(wkr.firstSSHConnection).Seconds())
	}
	wkr.timeToReadyReported = true
}

// caller must have lock.
func (wkr *worker) setIdleBehavior(idleBehavior IdleBehavior) {
	wkr.logger.WithField("IdleBehavior", idleBehavior).Info("set idle behavior")
	wkr.idleBehavior = idleBehavior
	wkr.saveTags()
	wkr.shutdownIfIdle()
}

// caller must have lock.
func (wkr *worker) startContainer(ctr arvados.Container) {
	logger := wkr.logger.WithFields(logrus.Fields{
		"ContainerUUID": ctr.UUID,
		"Priority":      ctr.Priority,
	})
	logger.Debug("starting container")
	rr := newRemoteRunner(ctr.UUID, wkr)
	wkr.starting[ctr.UUID] = rr
	if wkr.state != StateRunning {
		wkr.state = StateRunning
		go wkr.wp.notify()
	}
	go func() {
		rr.Start()
		if wkr.wp.mTimeFromQueueToCrunchRun != nil {
			wkr.wp.mTimeFromQueueToCrunchRun.Observe(time.Since(ctr.CreatedAt).Seconds())
		}
		wkr.mtx.Lock()
		defer wkr.mtx.Unlock()
		now := time.Now()
		wkr.updated = now
		wkr.busy = now
		delete(wkr.starting, ctr.UUID)
		wkr.running[ctr.UUID] = rr
		wkr.lastUUID = ctr.UUID
	}()
}

// ProbeAndUpdate conducts appropriate boot/running probes (if any)
// for the worker's current state. If a previous probe is still
// running, it does nothing.
//
// It should be called in a new goroutine.
func (wkr *worker) ProbeAndUpdate() {
	select {
	case wkr.probing <- struct{}{}:
		wkr.probeAndUpdate()
		<-wkr.probing
	default:
		wkr.logger.Debug("still waiting for last probe to finish")
	}
}

// probeAndUpdate calls probeBooted and/or probeRunning if needed, and
// updates state accordingly.
//
// In StateUnknown: Call both probeBooted and probeRunning.
// In StateBooting: Call probeBooted; if successful, call probeRunning.
// In StateRunning: Call probeRunning.
// In StateIdle: Call probeRunning.
// In StateShutdown: Do nothing.
//
// If both probes succeed, wkr.state changes to
// StateIdle/StateRunning.
//
// If probeRunning succeeds, wkr.running is updated. (This means
// wkr.running might be non-empty even in StateUnknown, if the boot
// probe failed.)
//
// probeAndUpdate should be called in a new goroutine.
func (wkr *worker) probeAndUpdate() {
	wkr.mtx.Lock()
	updated := wkr.updated
	initialState := wkr.state
	wkr.mtx.Unlock()

	var (
		booted   bool
		ctrUUIDs []string
		ok       bool
		stderr   []byte // from probeBooted
	)

	switch initialState {
	case StateShutdown:
		return
	case StateIdle, StateRunning:
		booted = true
	case StateUnknown, StateBooting:
	default:
		panic(fmt.Sprintf("unknown state %s", initialState))
	}

	probeStart := time.Now()
	logger := wkr.logger.WithField("ProbeStart", probeStart)

	if !booted {
		booted, stderr = wkr.probeBooted()
		if !booted {
			// Pretend this probe succeeded if another
			// concurrent attempt succeeded.
			wkr.mtx.Lock()
			booted = wkr.state == StateRunning || wkr.state == StateIdle
			wkr.mtx.Unlock()
		}
		if booted {
			logger.Info("instance booted; will try probeRunning")
		}
	}
	reportedBroken := false
	if booted || wkr.state == StateUnknown {
		ctrUUIDs, reportedBroken, ok = wkr.probeRunning()
	}
	wkr.mtx.Lock()
	defer wkr.mtx.Unlock()
	if reportedBroken && wkr.idleBehavior == IdleBehaviorRun {
		logger.Info("probe reported broken instance")
		wkr.reportBootOutcome(BootOutcomeFailed)
		wkr.setIdleBehavior(IdleBehaviorDrain)
	}
	if !ok || (!booted && len(ctrUUIDs) == 0 && len(wkr.running) == 0) {
		if wkr.state == StateShutdown && wkr.updated.After(updated) {
			// Skip the logging noise if shutdown was
			// initiated during probe.
			return
		}
		// Using the start time of the probe as the timeout
		// threshold ensures we always initiate at least one
		// probe attempt after the boot/probe timeout expires
		// (otherwise, a slow probe failure could cause us to
		// shutdown an instance even though it did in fact
		// boot/recover before the timeout expired).
		dur := probeStart.Sub(wkr.probed)
		if wkr.shutdownIfBroken(dur) {
			// stderr from failed run-probes will have
			// been logged already, but boot-probe
			// failures are normal so they are logged only
			// at Debug level. This is our chance to log
			// some evidence about why the node never
			// booted, even in non-debug mode.
			if !booted {
				wkr.reportBootOutcome(BootOutcomeFailed)
				logger.WithFields(logrus.Fields{
					"Duration": dur,
					"stderr":   string(stderr),
				}).Info("boot failed")
			}
		}
		return
	}

	updateTime := time.Now()
	wkr.probed = updateTime

	if updated != wkr.updated {
		// Worker was updated after the probe began, so
		// wkr.running might have a container UUID that was
		// not yet running when ctrUUIDs was generated. Leave
		// wkr.running alone and wait for the next probe to
		// catch up on any changes.
		logger.WithFields(logrus.Fields{
			"updated":     updated,
			"wkr.updated": wkr.updated,
		}).Debug("skipping worker state update due to probe/sync race")
		return
	}

	if len(ctrUUIDs) > 0 {
		wkr.busy = updateTime
		wkr.lastUUID = ctrUUIDs[0]
	} else if len(wkr.running) > 0 {
		// Actual last-busy time was sometime between wkr.busy
		// and now. Now is the earliest opportunity to take
		// advantage of the non-busy state, though.
		wkr.busy = updateTime
	}

	changed := wkr.updateRunning(ctrUUIDs)

	// Update state if this was the first successful boot-probe.
	if booted && (wkr.state == StateUnknown || wkr.state == StateBooting) {
		if wkr.state == StateBooting {
			wkr.reportTimeBetweenFirstSSHAndReadyForContainer()
		}
		// Note: this will change again below if
		// len(wkr.starting)+len(wkr.running) > 0.
		wkr.state = StateIdle
		changed = true
	}

	// If wkr.state and wkr.running aren't changing then there's
	// no need to log anything, notify the scheduler, move state
	// back and forth between idle/running, etc.
	if !changed {
		return
	}

	// Log whenever a run-probe reveals crunch-run processes
	// appearing/disappearing before boot-probe succeeds.
	if wkr.state == StateUnknown && changed {
		logger.WithFields(logrus.Fields{
			"RunningContainers": len(wkr.running),
			"State":             wkr.state,
		}).Info("crunch-run probe succeeded, but boot probe is still failing")
	}

	if wkr.state == StateIdle && len(wkr.starting)+len(wkr.running) > 0 {
		wkr.state = StateRunning
	} else if wkr.state == StateRunning && len(wkr.starting)+len(wkr.running) == 0 {
		wkr.state = StateIdle
	}
	wkr.updated = updateTime
	if booted && (initialState == StateUnknown || initialState == StateBooting) {
		wkr.reportBootOutcome(BootOutcomeSucceeded)
		logger.WithFields(logrus.Fields{
			"RunningContainers": len(wkr.running),
			"State":             wkr.state,
		}).Info("probes succeeded, instance is in service")
	}
	go wkr.wp.notify()
}

func (wkr *worker) probeRunning() (running []string, reportsBroken, ok bool) {
	cmd := wkr.wp.runnerCmd + " --list"
	if u := wkr.instance.RemoteUser(); u != "root" {
		cmd = "sudo " + cmd
	}
	before := time.Now()
	var stdin io.Reader
	if prices := wkr.instance.PriceHistory(wkr.instType); len(prices) > 0 {
		j, _ := json.Marshal(prices)
		stdin = bytes.NewReader(j)
	}
	stdout, stderr, err := wkr.executor.Execute(nil, cmd, stdin)
	if err != nil {
		wkr.logger.WithFields(logrus.Fields{
			"Command": cmd,
			"stdout":  string(stdout),
			"stderr":  string(stderr),
		}).WithError(err).Warn("probe failed")
		wkr.wp.mRunProbeDuration.WithLabelValues("fail").Observe(time.Now().Sub(before).Seconds())
		return
	}
	wkr.logger.WithFields(logrus.Fields{
		"Command": cmd,
		"stdout":  string(stdout),
		"stderr":  string(stderr),
	}).Debug("probe succeeded")
	wkr.wp.mRunProbeDuration.WithLabelValues("success").Observe(time.Now().Sub(before).Seconds())
	ok = true

	staleRunLock := false
	for _, s := range strings.Split(string(stdout), "\n") {
		// Each line of the "crunch-run --list" output is one
		// of the following:
		//
		// * a container UUID, indicating that processes
		//   related to that container are currently running.
		//   Optionally followed by " stale", indicating that
		//   the crunch-run process itself has exited (the
		//   remaining process is probably arv-mount).
		//
		// * the string "broken", indicating that the instance
		//   appears incapable of starting containers.
		//
		// See ListProcesses() in lib/crunchrun/background.go.
		if s == "" {
			// empty string following final newline
		} else if s == "broken" {
			reportsBroken = true
		} else if !strings.HasPrefix(s, wkr.wp.cluster.ClusterID) {
			// Ignore crunch-run processes that belong to
			// a different cluster (e.g., a single host
			// running multiple clusters with the loopback
			// driver)
			continue
		} else if toks := strings.Split(s, " "); len(toks) == 1 {
			running = append(running, s)
		} else if toks[1] == "stale" {
			wkr.logger.WithField("ContainerUUID", toks[0]).Info("probe reported stale run lock")
			staleRunLock = true
		}
	}
	wkr.mtx.Lock()
	defer wkr.mtx.Unlock()
	if !staleRunLock {
		wkr.staleRunLockSince = time.Time{}
	} else if wkr.staleRunLockSince.IsZero() {
		wkr.staleRunLockSince = time.Now()
	} else if dur := time.Now().Sub(wkr.staleRunLockSince); dur > wkr.wp.timeoutStaleRunLock {
		wkr.logger.WithField("Duration", dur).Warn("reporting broken after reporting stale run lock for too long")
		reportsBroken = true
	}
	return
}

func (wkr *worker) probeBooted() (ok bool, stderr []byte) {
	cmd := wkr.wp.bootProbeCommand
	if cmd == "" {
		cmd = "true"
	}
	stdout, stderr, err := wkr.executor.Execute(nil, cmd, nil)
	logger := wkr.logger.WithFields(logrus.Fields{
		"Command": cmd,
		"stdout":  string(stdout),
		"stderr":  string(stderr),
	})
	if err != nil {
		logger.WithError(err).Debug("boot probe failed")
		return false, stderr
	}
	logger.Info("boot probe succeeded")
	if err = wkr.wp.loadRunnerData(); err != nil {
		wkr.logger.WithError(err).Warn("cannot boot worker: error loading runner binary")
		return false, stderr
	} else if len(wkr.wp.runnerData) == 0 {
		// Assume crunch-run is already installed
	} else if _, stderr2, err := wkr.copyRunnerData(); err != nil {
		wkr.logger.WithError(err).WithField("stderr", string(stderr2)).Warn("error copying runner binary")
		return false, stderr2
	} else {
		stderr = append(stderr, stderr2...)
	}
	return true, stderr
}

func (wkr *worker) copyRunnerData() (stdout, stderr []byte, err error) {
	hash := fmt.Sprintf("%x", wkr.wp.runnerMD5)
	dstdir, _ := filepath.Split(wkr.wp.runnerCmd)
	logger := wkr.logger.WithFields(logrus.Fields{
		"hash": hash,
		"path": wkr.wp.runnerCmd,
	})

	stdout, stderr, err = wkr.executor.Execute(nil, `md5sum `+wkr.wp.runnerCmd, nil)
	if err == nil && len(stderr) == 0 && bytes.Equal(stdout, []byte(hash+"  "+wkr.wp.runnerCmd+"\n")) {
		logger.Info("runner binary already exists on worker, with correct hash")
		return
	}

	// Note touch+chmod come before writing data, to avoid the
	// possibility of md5 being correct while file mode is
	// incorrect.
	cmd := `set -e; dstdir="` + dstdir + `"; dstfile="` + wkr.wp.runnerCmd + `"; mkdir -p "$dstdir"; touch "$dstfile"; chmod 0755 "$dstdir" "$dstfile"; cat >"$dstfile"`
	if wkr.instance.RemoteUser() != "root" {
		cmd = `sudo sh -c '` + strings.Replace(cmd, "'", "'\\''", -1) + `'`
	}
	logger.WithField("cmd", cmd).Info("installing runner binary on worker")
	stdout, stderr, err = wkr.executor.Execute(nil, cmd, bytes.NewReader(wkr.wp.runnerData))
	return
}

// caller must have lock.
func (wkr *worker) shutdownIfBroken(dur time.Duration) bool {
	if wkr.idleBehavior == IdleBehaviorHold {
		// Never shut down.
		return false
	}
	label, threshold := "", wkr.wp.timeoutProbe
	if wkr.state == StateUnknown || wkr.state == StateBooting {
		label, threshold = "new ", wkr.wp.timeoutBooting
	}
	if dur < threshold {
		return false
	}
	wkr.logger.WithFields(logrus.Fields{
		"Duration": dur,
		"Since":    wkr.probed,
		"State":    wkr.state,
	}).Warnf("%sinstance unresponsive, shutting down", label)
	wkr.shutdown()
	return true
}

// Returns true if the instance is eligible for shutdown: either it's
// been idle too long, or idleBehavior=Drain and nothing is running.
//
// caller must have lock.
func (wkr *worker) eligibleForShutdown() bool {
	if wkr.idleBehavior == IdleBehaviorHold {
		return false
	}
	draining := wkr.idleBehavior == IdleBehaviorDrain
	switch wkr.state {
	case StateBooting:
		return draining
	case StateIdle:
		return draining || time.Since(wkr.busy) >= wkr.wp.timeoutIdle
	case StateRunning:
		if !draining {
			return false
		}
		for _, rr := range wkr.running {
			if !rr.givenup {
				return false
			}
		}
		for _, rr := range wkr.starting {
			if !rr.givenup {
				return false
			}
		}
		// draining, and all remaining runners are just trying
		// to force-kill their crunch-run procs
		return true
	default:
		return false
	}
}

// caller must have lock.
func (wkr *worker) shutdownIfIdle() bool {
	if !wkr.eligibleForShutdown() {
		return false
	}
	wkr.logger.WithFields(logrus.Fields{
		"State":        wkr.state,
		"IdleDuration": stats.Duration(time.Since(wkr.busy)),
		"IdleBehavior": wkr.idleBehavior,
	}).Info("shutdown worker")
	wkr.reportBootOutcome(BootOutcomeAborted)
	wkr.shutdown()
	return true
}

// caller must have lock.
func (wkr *worker) shutdown() {
	now := time.Now()
	wkr.updated = now
	wkr.destroyed = now
	wkr.state = StateShutdown
	go wkr.wp.notify()
	go func() {
		err := wkr.instance.Destroy()
		if err != nil {
			wkr.logger.WithError(err).Warn("shutdown failed")
			return
		}
	}()
}

// Save worker tags to cloud provider metadata, if they don't already
// match. Caller must have lock.
func (wkr *worker) saveTags() {
	instance := wkr.instance
	tags := instance.Tags()
	update := cloud.InstanceTags{
		wkr.wp.tagKeyPrefix + tagKeyInstanceType: wkr.instType.Name,
		wkr.wp.tagKeyPrefix + tagKeyIdleBehavior: string(wkr.idleBehavior),
	}
	save := false
	for k, v := range update {
		if tags[k] != v {
			tags[k] = v
			save = true
		}
	}
	if save {
		go func() {
			err := instance.SetTags(tags)
			if err != nil {
				wkr.wp.logger.WithField("Instance", instance.ID()).WithError(err).Warnf("error updating tags")
			}
		}()
	}
}

func (wkr *worker) Close() {
	// This might take time, so do it after unlocking mtx.
	defer wkr.executor.Close()

	wkr.mtx.Lock()
	defer wkr.mtx.Unlock()
	for uuid, rr := range wkr.running {
		wkr.logger.WithField("ContainerUUID", uuid).Info("crunch-run process abandoned")
		rr.Close()
	}
	for uuid, rr := range wkr.starting {
		wkr.logger.WithField("ContainerUUID", uuid).Info("crunch-run process abandoned")
		rr.Close()
	}
}

// Add/remove entries in wkr.running to match ctrUUIDs returned by a
// probe. Returns true if anything was added or removed.
//
// Caller must have lock.
func (wkr *worker) updateRunning(ctrUUIDs []string) (changed bool) {
	alive := map[string]bool{}
	for _, uuid := range ctrUUIDs {
		alive[uuid] = true
		if _, ok := wkr.running[uuid]; ok {
			// unchanged
		} else if rr, ok := wkr.starting[uuid]; ok {
			wkr.running[uuid] = rr
			delete(wkr.starting, uuid)
			changed = true
		} else {
			// We didn't start it -- it must have been
			// started by a previous dispatcher process.
			wkr.logger.WithField("ContainerUUID", uuid).Info("crunch-run process detected")
			wkr.running[uuid] = newRemoteRunner(uuid, wkr)
			changed = true
		}
	}
	for uuid := range wkr.running {
		if !alive[uuid] {
			wkr.closeRunner(uuid)
			changed = true
		}
	}
	return
}

// caller must have lock.
func (wkr *worker) closeRunner(uuid string) {
	rr := wkr.running[uuid]
	if rr == nil {
		return
	}
	wkr.logger.WithField("ContainerUUID", uuid).Info("crunch-run process ended")
	delete(wkr.running, uuid)
	rr.Close()

	now := time.Now()
	wkr.updated = now
	wkr.wp.exited[uuid] = now
	if wkr.state == StateRunning && len(wkr.running)+len(wkr.starting) == 0 {
		wkr.state = StateIdle
	}
}
