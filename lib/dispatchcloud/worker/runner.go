// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// remoteRunner handles the starting and stopping of a crunch-run
// process on a remote machine.
type remoteRunner struct {
	uuid          string
	executor      Executor
	envJSON       json.RawMessage
	runnerCmd     string
	runnerArgs    []string
	remoteUser    string
	timeoutTERM   time.Duration
	timeoutSignal time.Duration
	onUnkillable  func(uuid string) // callback invoked when giving up on SIGTERM
	onKilled      func(uuid string) // callback invoked when process exits after SIGTERM
	logger        logrus.FieldLogger

	stopping bool          // true if Stop() has been called
	givenup  bool          // true if timeoutTERM has been reached
	closed   chan struct{} // channel is closed if Close() has been called
}

// newRemoteRunner returns a new remoteRunner. Caller should ensure
// Close() is called to release resources.
func newRemoteRunner(uuid string, wkr *worker) *remoteRunner {
	// Send the instance type record as a JSON doc so crunch-run
	// can log it.
	var instJSON bytes.Buffer
	enc := json.NewEncoder(&instJSON)
	enc.SetIndent("", "    ")
	if err := enc.Encode(wkr.instType); err != nil {
		panic(err)
	}
	env := map[string]string{
		"ARVADOS_API_HOST":  wkr.wp.arvClient.APIHost,
		"ARVADOS_API_TOKEN": wkr.wp.arvClient.AuthToken,
		"InstanceType":      instJSON.String(),
	}
	if wkr.wp.arvClient.Insecure {
		env["ARVADOS_API_HOST_INSECURE"] = "1"
	}
	envJSON, err := json.Marshal(env)
	if err != nil {
		panic(err)
	}
	rr := &remoteRunner{
		uuid:          uuid,
		executor:      wkr.executor,
		envJSON:       envJSON,
		runnerCmd:     wkr.wp.runnerCmd,
		runnerArgs:    wkr.wp.runnerArgs,
		remoteUser:    wkr.instance.RemoteUser(),
		timeoutTERM:   wkr.wp.timeoutTERM,
		timeoutSignal: wkr.wp.timeoutSignal,
		onUnkillable:  wkr.onUnkillable,
		onKilled:      wkr.onKilled,
		logger:        wkr.logger.WithField("ContainerUUID", uuid),
		closed:        make(chan struct{}),
	}
	return rr
}

// Start a crunch-run process on the remote host.
//
// Start does not return any error encountered. The caller should
// assume the remote process _might_ have started, at least until it
// probes the worker and finds otherwise.
func (rr *remoteRunner) Start() {
	cmd := rr.runnerCmd + " --detach --stdin-env"
	for _, arg := range rr.runnerArgs {
		cmd += " '" + strings.Replace(arg, "'", "'\\''", -1) + "'"
	}
	cmd += " '" + rr.uuid + "'"
	if rr.remoteUser != "root" {
		cmd = "sudo " + cmd
	}
	stdin := bytes.NewBuffer(rr.envJSON)
	stdout, stderr, err := rr.executor.Execute(nil, cmd, stdin)
	if err != nil {
		rr.logger.WithField("stdout", string(stdout)).
			WithField("stderr", string(stderr)).
			WithError(err).
			Error("error starting crunch-run process")
		return
	}
	rr.logger.Info("crunch-run process started")
}

// Close abandons the remote process (if any) and releases
// resources. Close must not be called more than once.
func (rr *remoteRunner) Close() {
	close(rr.closed)
}

// Kill starts a background task to kill the remote process, first
// trying SIGTERM until reaching timeoutTERM, then calling
// onUnkillable().
//
// SIGKILL is not used. It would merely kill the crunch-run supervisor
// and thereby make the docker container, arv-mount, etc. invisible to
// us without actually stopping them.
//
// Once Kill has been called, calling it again has no effect.
func (rr *remoteRunner) Kill(reason string) {
	if rr.stopping {
		return
	}
	rr.stopping = true
	rr.logger.WithField("Reason", reason).Info("killing crunch-run process")
	go func() {
		termDeadline := time.Now().Add(rr.timeoutTERM)
		t := time.NewTicker(rr.timeoutSignal)
		defer t.Stop()
		for range t.C {
			switch {
			case rr.isClosed():
				return
			case time.Now().After(termDeadline):
				rr.logger.Debug("giving up")
				rr.givenup = true
				rr.onUnkillable(rr.uuid)
				return
			default:
				rr.kill(syscall.SIGTERM)
			}
		}
	}()
}

func (rr *remoteRunner) kill(sig syscall.Signal) {
	logger := rr.logger.WithField("Signal", int(sig))
	logger.Info("sending signal")
	cmd := fmt.Sprintf(rr.runnerCmd+" --kill %d %s", sig, rr.uuid)
	if rr.remoteUser != "root" {
		cmd = "sudo " + cmd
	}
	stdout, stderr, err := rr.executor.Execute(nil, cmd, nil)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"stderr": string(stderr),
			"stdout": string(stdout),
			"error":  err,
		}).Info("kill attempt unsuccessful")
		return
	}
	rr.onKilled(rr.uuid)
}

func (rr *remoteRunner) isClosed() bool {
	select {
	case <-rr.closed:
		return true
	default:
		return false
	}
}
