// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"errors"
	"io"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/test"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&WorkerSuite{})

type WorkerSuite struct{}

func (suite *WorkerSuite) TestProbeAndUpdate(c *check.C) {
	logger := ctxlog.TestLogger(c)
	bootTimeout := time.Minute
	probeTimeout := time.Second

	is, err := (&test.StubDriver{}).InstanceSet(nil, "", logger)
	c.Assert(err, check.IsNil)
	inst, err := is.Create(arvados.InstanceType{}, "", nil, "echo InitCommand", nil)
	c.Assert(err, check.IsNil)

	type trialT struct {
		testCaseComment string // displayed in test output to help identify failure case
		age             time.Duration
		state           State
		running         int
		starting        int
		respBoot        stubResp // zero value is success
		respRun         stubResp // zero value is success + nothing running
		expectState     State
		expectRunning   int
	}

	errFail := errors.New("failed")
	respFail := stubResp{"", "command failed\n", errFail}
	respContainerRunning := stubResp{"zzzzz-dz642-abcdefghijklmno\n", "", nil}
	for _, trial := range []trialT{
		{
			testCaseComment: "Unknown, probes fail",
			state:           StateUnknown,
			respBoot:        respFail,
			respRun:         respFail,
			expectState:     StateUnknown,
		},
		{
			testCaseComment: "Unknown, boot probe fails, but one container is running",
			state:           StateUnknown,
			respBoot:        respFail,
			respRun:         respContainerRunning,
			expectState:     StateUnknown,
			expectRunning:   1,
		},
		{
			testCaseComment: "Unknown, boot probe fails, previously running container has exited",
			state:           StateUnknown,
			running:         1,
			respBoot:        respFail,
			expectState:     StateUnknown,
			expectRunning:   0,
		},
		{
			testCaseComment: "Unknown, boot timeout exceeded, boot probe fails",
			state:           StateUnknown,
			age:             bootTimeout + time.Second,
			respBoot:        respFail,
			respRun:         respFail,
			expectState:     StateShutdown,
		},
		{
			testCaseComment: "Unknown, boot timeout exceeded, boot probe succeeds but crunch-run fails",
			state:           StateUnknown,
			age:             bootTimeout * 2,
			respRun:         respFail,
			expectState:     StateShutdown,
		},
		{
			testCaseComment: "Unknown, boot timeout exceeded, boot probe fails but crunch-run succeeds",
			state:           StateUnknown,
			age:             bootTimeout * 2,
			respBoot:        respFail,
			expectState:     StateShutdown,
		},
		{
			testCaseComment: "Unknown, boot timeout exceeded, boot probe fails but container is running",
			state:           StateUnknown,
			age:             bootTimeout * 2,
			respBoot:        respFail,
			respRun:         respContainerRunning,
			expectState:     StateUnknown,
			expectRunning:   1,
		},
		{
			testCaseComment: "Booting, boot probe fails, run probe fails",
			state:           StateBooting,
			respBoot:        respFail,
			respRun:         respFail,
			expectState:     StateBooting,
		},
		{
			testCaseComment: "Booting, boot probe fails, run probe succeeds (but isn't expected to be called)",
			state:           StateBooting,
			respBoot:        respFail,
			expectState:     StateBooting,
		},
		{
			testCaseComment: "Booting, boot probe succeeds, run probe fails",
			state:           StateBooting,
			respRun:         respFail,
			expectState:     StateBooting,
		},
		{
			testCaseComment: "Booting, boot probe succeeds, run probe succeeds",
			state:           StateBooting,
			expectState:     StateIdle,
		},
		{
			testCaseComment: "Booting, boot probe succeeds, run probe succeeds, container is running",
			state:           StateBooting,
			respRun:         respContainerRunning,
			expectState:     StateRunning,
			expectRunning:   1,
		},
		{
			testCaseComment: "Booting, boot timeout exceeded",
			state:           StateBooting,
			age:             bootTimeout * 2,
			respRun:         respFail,
			expectState:     StateShutdown,
		},
		{
			testCaseComment: "Idle, probe timeout exceeded, one container running",
			state:           StateIdle,
			age:             probeTimeout * 2,
			respRun:         respContainerRunning,
			expectState:     StateRunning,
			expectRunning:   1,
		},
		{
			testCaseComment: "Idle, probe timeout exceeded, one container running, probe fails",
			state:           StateIdle,
			age:             probeTimeout * 2,
			running:         1,
			respRun:         respFail,
			expectState:     StateShutdown,
			expectRunning:   1,
		},
		{
			testCaseComment: "Idle, probe timeout exceeded, nothing running, probe fails",
			state:           StateIdle,
			age:             probeTimeout * 2,
			respRun:         respFail,
			expectState:     StateShutdown,
		},
		{
			testCaseComment: "Running, one container still running",
			state:           StateRunning,
			running:         1,
			respRun:         respContainerRunning,
			expectState:     StateRunning,
			expectRunning:   1,
		},
		{
			testCaseComment: "Running, container has exited",
			state:           StateRunning,
			running:         1,
			expectState:     StateIdle,
			expectRunning:   0,
		},
		{
			testCaseComment: "Running, probe timeout exceeded, nothing running, new container being started",
			state:           StateRunning,
			age:             probeTimeout * 2,
			starting:        1,
			expectState:     StateRunning,
		},
	} {
		c.Logf("------- %#v", trial)
		ctime := time.Now().Add(-trial.age)
		exr := stubExecutor{
			"bootprobe":         trial.respBoot,
			"crunch-run --list": trial.respRun,
		}
		wp := &Pool{
			newExecutor:      func(cloud.Instance) Executor { return exr },
			bootProbeCommand: "bootprobe",
			timeoutBooting:   bootTimeout,
			timeoutProbe:     probeTimeout,
			exited:           map[string]time.Time{},
		}
		wkr := &worker{
			logger:   logger,
			executor: exr,
			wp:       wp,
			mtx:      &wp.mtx,
			state:    trial.state,
			instance: inst,
			appeared: ctime,
			busy:     ctime,
			probed:   ctime,
			updated:  ctime,
		}
		if trial.running > 0 {
			wkr.running = map[string]struct{}{"zzzzz-dz642-abcdefghijklmno": struct{}{}}
		}
		if trial.starting > 0 {
			wkr.starting = map[string]struct{}{"zzzzz-dz642-abcdefghijklmno": struct{}{}}
		}
		wkr.probeAndUpdate()
		c.Check(wkr.state, check.Equals, trial.expectState)
		c.Check(len(wkr.running), check.Equals, trial.expectRunning)
	}
}

type stubResp struct {
	stdout string
	stderr string
	err    error
}
type stubExecutor map[string]stubResp

func (se stubExecutor) SetTarget(cloud.ExecutorTarget) {}
func (se stubExecutor) Close()                         {}
func (se stubExecutor) Execute(env map[string]string, cmd string, stdin io.Reader) (stdout, stderr []byte, err error) {
	resp, ok := se[cmd]
	if !ok {
		return nil, []byte("command not found\n"), errors.New("command not found")
	}
	return []byte(resp.stdout), []byte(resp.stderr), resp.err
}
