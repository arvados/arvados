// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/stats"
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

	mtx          sync.Locker // must be wp's Locker.
	state        State
	idleBehavior IdleBehavior
	instance     cloud.Instance
	instType     arvados.InstanceType
	vcpus        int64
	memory       int64
	appeared     time.Time
	probed       time.Time
	updated      time.Time
	busy         time.Time
	destroyed    time.Time
	lastUUID     string
	running      map[string]struct{} // remember to update state idle<->running when this changes
	starting     map[string]struct{} // remember to update state idle<->running when this changes
	probing      chan struct{}
}

// caller must have lock.
func (wkr *worker) startContainer(ctr arvados.Container) {
	logger := wkr.logger.WithFields(logrus.Fields{
		"ContainerUUID": ctr.UUID,
		"Priority":      ctr.Priority,
	})
	logger = logger.WithField("Instance", wkr.instance.ID())
	logger.Debug("starting container")
	wkr.starting[ctr.UUID] = struct{}{}
	if wkr.state != StateRunning {
		wkr.state = StateRunning
		go wkr.wp.notify()
	}
	go func() {
		env := map[string]string{
			"ARVADOS_API_HOST":  wkr.wp.arvClient.APIHost,
			"ARVADOS_API_TOKEN": wkr.wp.arvClient.AuthToken,
		}
		if wkr.wp.arvClient.Insecure {
			env["ARVADOS_API_HOST_INSECURE"] = "1"
		}
		envJSON, err := json.Marshal(env)
		if err != nil {
			panic(err)
		}
		stdin := bytes.NewBuffer(envJSON)
		cmd := "crunch-run --detach --stdin-env '" + ctr.UUID + "'"
		if u := wkr.instance.RemoteUser(); u != "root" {
			cmd = "sudo " + cmd
		}
		stdout, stderr, err := wkr.executor.Execute(nil, cmd, stdin)
		wkr.mtx.Lock()
		defer wkr.mtx.Unlock()
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
}

// ProbeAndUpdate conducts appropriate boot/running probes (if any)
// for the worker's curent state. If a previous probe is still
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
	if booted || wkr.state == StateUnknown {
		ctrUUIDs, ok = wkr.probeRunning()
	}
	wkr.mtx.Lock()
	defer wkr.mtx.Unlock()
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
	changed := false

	// Build a new "running" map. Set changed=true if it differs
	// from the existing map (wkr.running) to ensure the scheduler
	// gets notified below.
	running := map[string]struct{}{}
	for _, uuid := range ctrUUIDs {
		running[uuid] = struct{}{}
		if _, ok := wkr.running[uuid]; !ok {
			if _, ok := wkr.starting[uuid]; !ok {
				// We didn't start it -- it must have
				// been started by a previous
				// dispatcher process.
				logger.WithField("ContainerUUID", uuid).Info("crunch-run process detected")
			}
			changed = true
		}
	}
	for uuid := range wkr.running {
		if _, ok := running[uuid]; !ok {
			logger.WithField("ContainerUUID", uuid).Info("crunch-run process ended")
			wkr.wp.notifyExited(uuid, updateTime)
			changed = true
		}
	}

	// Update state if this was the first successful boot-probe.
	if booted && (wkr.state == StateUnknown || wkr.state == StateBooting) {
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
	if wkr.state == StateUnknown && len(running) != len(wkr.running) {
		logger.WithFields(logrus.Fields{
			"RunningContainers": len(running),
			"State":             wkr.state,
		}).Info("crunch-run probe succeeded, but boot probe is still failing")
	}

	wkr.running = running
	if wkr.state == StateIdle && len(wkr.starting)+len(wkr.running) > 0 {
		wkr.state = StateRunning
	} else if wkr.state == StateRunning && len(wkr.starting)+len(wkr.running) == 0 {
		wkr.state = StateIdle
	}
	wkr.updated = updateTime
	if booted && (initialState == StateUnknown || initialState == StateBooting) {
		logger.WithFields(logrus.Fields{
			"RunningContainers": len(running),
			"State":             wkr.state,
		}).Info("probes succeeded, instance is in service")
	}
	go wkr.wp.notify()
}

func (wkr *worker) probeRunning() (running []string, ok bool) {
	cmd := "crunch-run --list"
	if u := wkr.instance.RemoteUser(); u != "root" {
		cmd = "sudo " + cmd
	}
	stdout, stderr, err := wkr.executor.Execute(nil, cmd, nil)
	if err != nil {
		wkr.logger.WithFields(logrus.Fields{
			"Command": cmd,
			"stdout":  string(stdout),
			"stderr":  string(stderr),
		}).WithError(err).Warn("probe failed")
		return nil, false
	}
	stdout = bytes.TrimRight(stdout, "\n")
	if len(stdout) == 0 {
		return nil, true
	}
	return strings.Split(string(stdout), "\n"), true
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
	return true, stderr
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

// caller must have lock.
func (wkr *worker) shutdownIfIdle() bool {
	if wkr.idleBehavior == IdleBehaviorHold {
		// Never shut down.
		return false
	}
	age := time.Since(wkr.busy)

	old := age >= wkr.wp.timeoutIdle
	draining := wkr.idleBehavior == IdleBehaviorDrain
	shouldShutdown := ((old || draining) && wkr.state == StateIdle) ||
		(draining && wkr.state == StateBooting)
	if !shouldShutdown {
		return false
	}

	wkr.logger.WithFields(logrus.Fields{
		"State":        wkr.state,
		"IdleDuration": stats.Duration(age),
		"IdleBehavior": wkr.idleBehavior,
	}).Info("shutdown idle worker")
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
		tagKeyInstanceType: wkr.instType.Name,
		tagKeyIdleBehavior: string(wkr.idleBehavior),
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
