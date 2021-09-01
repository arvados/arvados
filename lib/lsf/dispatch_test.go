// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package lsf

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&suite{})

type suite struct {
	disp *dispatcher
}

func (s *suite) TearDownTest(c *check.C) {
	arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil)
}

func (s *suite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	cluster.Containers.CloudVMs.PollInterval = arvados.Duration(time.Second)
	s.disp = newHandler(context.Background(), cluster, arvadostest.Dispatch1Token, prometheus.NewRegistry()).(*dispatcher)
	s.disp.lsfcli.stubCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo >&2 unimplemented stub; false")
	}
}

type lsfstub struct {
	sudoUser  string
	errorRate float64
}

func (stub lsfstub) stubCommand(s *suite, c *check.C) func(prog string, args ...string) *exec.Cmd {
	mtx := sync.Mutex{}
	nextjobid := 100
	fakejobq := map[int]string{}
	return func(prog string, args ...string) *exec.Cmd {
		c.Logf("stubCommand: %q %q", prog, args)
		if rand.Float64() < stub.errorRate {
			return exec.Command("bash", "-c", "echo >&2 'stub random failure' && false")
		}
		if stub.sudoUser != "" && len(args) > 3 &&
			prog == "sudo" &&
			args[0] == "-E" &&
			args[1] == "-u" &&
			args[2] == stub.sudoUser {
			prog, args = args[3], args[4:]
		}
		switch prog {
		case "bsub":
			defaultArgs := s.disp.Cluster.Containers.LSF.BsubArgumentsList
			c.Assert(args, check.HasLen, 4+len(defaultArgs))
			c.Check(args[:len(defaultArgs)], check.DeepEquals, defaultArgs)
			args = args[len(defaultArgs):]

			c.Check(args[0], check.Equals, "-J")
			switch args[1] {
			case arvadostest.LockedContainerUUID:
				c.Check(args, check.DeepEquals, []string{"-J", arvadostest.LockedContainerUUID, "-R", "rusage[mem=11701MB:tmp=0MB] affinity[core(4)]"})
				mtx.Lock()
				fakejobq[nextjobid] = args[1]
				nextjobid++
				mtx.Unlock()
			case arvadostest.QueuedContainerUUID:
				c.Check(args, check.DeepEquals, []string{"-J", arvadostest.QueuedContainerUUID, "-R", "rusage[mem=11701MB:tmp=45777MB] affinity[core(4)]"})
				mtx.Lock()
				fakejobq[nextjobid] = args[1]
				nextjobid++
				mtx.Unlock()
			default:
				c.Errorf("unexpected uuid passed to bsub: args %q", args)
				return exec.Command("false")
			}
			return exec.Command("echo", "submitted job")
		case "bjobs":
			c.Check(args, check.DeepEquals, []string{"-u", "all", "-noheader", "-o", "jobid stat job_name:30"})
			out := ""
			for jobid, uuid := range fakejobq {
				out += fmt.Sprintf(`%d %s %s\n`, jobid, "RUN", uuid)
			}
			c.Logf("bjobs out: %q", out)
			return exec.Command("printf", out)
		case "bkill":
			killid, _ := strconv.Atoi(args[0])
			if uuid, ok := fakejobq[killid]; !ok {
				return exec.Command("bash", "-c", fmt.Sprintf("printf >&2 'Job <%d>: No matching job found\n'", killid))
			} else if uuid == "" {
				return exec.Command("bash", "-c", fmt.Sprintf("printf >&2 'Job <%d>: Job has already finished\n'", killid))
			} else {
				go func() {
					time.Sleep(time.Millisecond)
					mtx.Lock()
					delete(fakejobq, killid)
					mtx.Unlock()
				}()
				return exec.Command("bash", "-c", fmt.Sprintf("printf 'Job <%d> is being terminated\n'", killid))
			}
		default:
			return exec.Command("bash", "-c", fmt.Sprintf("echo >&2 'stub: command not found: %+q'", prog))
		}
	}
}

func (s *suite) TestSubmit(c *check.C) {
	s.disp.lsfcli.stubCommand = lsfstub{
		errorRate: 0.1,
		sudoUser:  s.disp.Cluster.Containers.LSF.BsubSudoUser,
	}.stubCommand(s, c)
	s.disp.Start()
	deadline := time.Now().Add(20 * time.Second)
	for range time.NewTicker(time.Second).C {
		if time.Now().After(deadline) {
			c.Error("timed out")
			break
		}
		// "queuedcontainer" should be running
		if _, ok := s.disp.lsfqueue.JobID(arvadostest.QueuedContainerUUID); !ok {
			continue
		}
		// "lockedcontainer" should be cancelled because it
		// has priority 0 (no matching container requests)
		if _, ok := s.disp.lsfqueue.JobID(arvadostest.LockedContainerUUID); ok {
			continue
		}
		var ctr arvados.Container
		if err := s.disp.arvDispatcher.Arv.Get("containers", arvadostest.LockedContainerUUID, nil, &ctr); err != nil {
			c.Logf("error getting container state for %s: %s", arvadostest.LockedContainerUUID, err)
			continue
		}
		if ctr.State != arvados.ContainerStateQueued {
			c.Logf("LockedContainer is not in the LSF queue but its arvados record has not been updated to state==Queued (state is %q)", ctr.State)
			continue
		}
		c.Log("reached desired state")
		break
	}
}
