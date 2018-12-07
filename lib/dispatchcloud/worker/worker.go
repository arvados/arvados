// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Sirupsen/logrus"
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
	StateHold                  // running, but not available to run new containers
)

const (
	// TODO: configurable
	maxPingFailTime = 10 * time.Minute
)

var stateString = map[State]string{
	StateUnknown:  "unknown",
	StateBooting:  "booting",
	StateIdle:     "idle",
	StateRunning:  "running",
	StateShutdown: "shutdown",
	StateHold:     "hold",
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

type worker struct {
	logger   logrus.FieldLogger
	executor Executor
	wp       *Pool

	mtx       sync.Locker // must be wp's Locker.
	state     State
	instance  cloud.Instance
	instType  arvados.InstanceType
	vcpus     int64
	memory    int64
	probed    time.Time
	updated   time.Time
	busy      time.Time
	destroyed time.Time
	lastUUID  string
	running   map[string]struct{} // remember to update state idle<->running when this changes
	starting  map[string]struct{} // remember to update state idle<->running when this changes
	probing   chan struct{}
}

// caller must have lock.
func (wkr *worker) startContainer(ctr arvados.Container) {
	logger := wkr.logger.WithFields(logrus.Fields{
		"ContainerUUID": ctr.UUID,
		"Priority":      ctr.Priority,
	})
	logger = logger.WithField("Instance", wkr.instance)
	logger.Debug("starting container")
	wkr.starting[ctr.UUID] = struct{}{}
	wkr.state = StateRunning
	go func() {
		stdout, stderr, err := wkr.executor.Execute("crunch-run --detach '"+ctr.UUID+"'", nil)
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

// should be called in a new goroutine
func (wkr *worker) probeAndUpdate() {
	wkr.mtx.Lock()
	updated := wkr.updated
	needProbeRunning := wkr.state == StateRunning || wkr.state == StateIdle
	needProbeBooted := wkr.state == StateUnknown || wkr.state == StateBooting
	wkr.mtx.Unlock()
	if !needProbeBooted && !needProbeRunning {
		return
	}

	var (
		ctrUUIDs []string
		ok       bool
		stderr   []byte
	)
	if needProbeBooted {
		ok, stderr = wkr.probeBooted()
		wkr.mtx.Lock()
		if ok || wkr.state == StateRunning || wkr.state == StateIdle {
			wkr.logger.Info("instance booted; will try probeRunning")
			needProbeRunning = true
		}
		wkr.mtx.Unlock()
	}
	if needProbeRunning {
		ctrUUIDs, ok, stderr = wkr.probeRunning()
	}
	logger := wkr.logger.WithField("stderr", string(stderr))
	wkr.mtx.Lock()
	defer wkr.mtx.Unlock()
	if !ok {
		if wkr.state == StateShutdown && wkr.updated.After(updated) {
			// Skip the logging noise if shutdown was
			// initiated during probe.
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
		wkr.shutdownIfBroken(dur)
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
			wkr.wp.notifyExited(uuid, updateTime)
			changed = true
		}
	}
	if wkr.state == StateUnknown || wkr.state == StateBooting {
		wkr.state = StateIdle
		changed = true
	}
	if changed {
		wkr.running = running
		if wkr.state == StateIdle && len(wkr.starting)+len(wkr.running) > 0 {
			wkr.state = StateRunning
		} else if wkr.state == StateRunning && len(wkr.starting)+len(wkr.running) == 0 {
			wkr.state = StateIdle
		}
		wkr.updated = updateTime
		go wkr.wp.notify()
	}
}

func (wkr *worker) probeRunning() (running []string, ok bool, stderr []byte) {
	cmd := "crunch-run --list"
	stdout, stderr, err := wkr.executor.Execute(cmd, nil)
	if err != nil {
		wkr.logger.WithFields(logrus.Fields{
			"Command": cmd,
			"stdout":  string(stdout),
			"stderr":  string(stderr),
		}).WithError(err).Warn("probe failed")
		return nil, false, stderr
	}
	stdout = bytes.TrimRight(stdout, "\n")
	if len(stdout) == 0 {
		return nil, true, stderr
	}
	return strings.Split(string(stdout), "\n"), true, stderr
}

func (wkr *worker) probeBooted() (ok bool, stderr []byte) {
	cmd := wkr.wp.bootProbeCommand
	if cmd == "" {
		cmd = "true"
	}
	stdout, stderr, err := wkr.executor.Execute(cmd, nil)
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
func (wkr *worker) shutdownIfBroken(dur time.Duration) {
	if wkr.state == StateHold {
		return
	}
	label, threshold := "", wkr.wp.timeoutProbe
	if wkr.state == StateBooting {
		label, threshold = "new ", wkr.wp.timeoutBooting
	}
	if dur < threshold {
		return
	}
	wkr.logger.WithFields(logrus.Fields{
		"Duration": dur,
		"Since":    wkr.probed,
		"State":    wkr.state,
	}).Warnf("%sinstance unresponsive, shutting down", label)
	wkr.shutdown()
}

// caller must have lock.
func (wkr *worker) shutdownIfIdle() bool {
	if wkr.state != StateIdle {
		return false
	}
	age := time.Since(wkr.busy)
	if age < wkr.wp.timeoutIdle {
		return false
	}
	wkr.logger.WithField("Age", age).Info("shutdown idle worker")
	wkr.shutdown()
	return true
}

// caller must have lock
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
