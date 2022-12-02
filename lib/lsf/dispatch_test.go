// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package lsf

import (
	"context"
	"encoding/json"
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
	disp          *dispatcher
	crTooBig      arvados.ContainerRequest
	crPending     arvados.ContainerRequest
	crCUDARequest arvados.ContainerRequest
}

func (s *suite) TearDownTest(c *check.C) {
	arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil)
}

func (s *suite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	cluster.Containers.ReserveExtraRAM = 256 << 20
	cluster.Containers.CloudVMs.PollInterval = arvados.Duration(time.Second / 4)
	cluster.Containers.MinRetryPeriod = arvados.Duration(time.Second / 4)
	cluster.InstanceTypes = arvados.InstanceTypeMap{
		"biggest_available_node": arvados.InstanceType{
			RAM:             100 << 30, // 100 GiB
			VCPUs:           4,
			IncludedScratch: 100 << 30,
			Scratch:         100 << 30,
		}}
	s.disp = newHandler(context.Background(), cluster, arvadostest.Dispatch1Token, prometheus.NewRegistry()).(*dispatcher)
	s.disp.lsfcli.stubCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo >&2 unimplemented stub; false")
	}
	err = arvados.NewClientFromEnv().RequestAndDecode(&s.crTooBig, "POST", "arvados/v1/container_requests", nil, map[string]interface{}{
		"container_request": map[string]interface{}{
			"runtime_constraints": arvados.RuntimeConstraints{
				RAM:   1000000000000,
				VCPUs: 1,
			},
			"container_image":     arvadostest.DockerImage112PDH,
			"command":             []string{"sleep", "1"},
			"mounts":              map[string]arvados.Mount{"/mnt/out": {Kind: "tmp", Capacity: 1000}},
			"output_path":         "/mnt/out",
			"state":               arvados.ContainerRequestStateCommitted,
			"priority":            1,
			"container_count_max": 1,
		},
	})
	c.Assert(err, check.IsNil)

	err = arvados.NewClientFromEnv().RequestAndDecode(&s.crPending, "POST", "arvados/v1/container_requests", nil, map[string]interface{}{
		"container_request": map[string]interface{}{
			"runtime_constraints": arvados.RuntimeConstraints{
				RAM:   100000000,
				VCPUs: 2,
			},
			"container_image":     arvadostest.DockerImage112PDH,
			"command":             []string{"sleep", "1"},
			"mounts":              map[string]arvados.Mount{"/mnt/out": {Kind: "tmp", Capacity: 1000}},
			"output_path":         "/mnt/out",
			"state":               arvados.ContainerRequestStateCommitted,
			"priority":            1,
			"container_count_max": 1,
		},
	})
	c.Assert(err, check.IsNil)

	err = arvados.NewClientFromEnv().RequestAndDecode(&s.crCUDARequest, "POST", "arvados/v1/container_requests", nil, map[string]interface{}{
		"container_request": map[string]interface{}{
			"runtime_constraints": arvados.RuntimeConstraints{
				RAM:   16000000,
				VCPUs: 1,
				CUDA: arvados.CUDARuntimeConstraints{
					DeviceCount:        1,
					DriverVersion:      "11.0",
					HardwareCapability: "8.0",
				},
			},
			"container_image":     arvadostest.DockerImage112PDH,
			"command":             []string{"sleep", "1"},
			"mounts":              map[string]arvados.Mount{"/mnt/out": {Kind: "tmp", Capacity: 1000}},
			"output_path":         "/mnt/out",
			"state":               arvados.ContainerRequestStateCommitted,
			"priority":            1,
			"container_count_max": 1,
		},
	})
	c.Assert(err, check.IsNil)

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
			if args[5] == s.crCUDARequest.ContainerUUID {
				c.Assert(len(args), check.Equals, len(defaultArgs)+len(s.disp.Cluster.Containers.LSF.BsubCUDAArguments))
			} else {
				c.Assert(len(args), check.Equals, len(defaultArgs))
			}
			// %%J must have been rewritten to %J
			c.Check(args[1], check.Equals, "/tmp/crunch-run.%J.out")
			args = args[4:]
			switch args[1] {
			case arvadostest.LockedContainerUUID:
				c.Check(args, check.DeepEquals, []string{
					"-J", arvadostest.LockedContainerUUID,
					"-n", "4",
					"-D", "11701MB",
					"-R", "rusage[mem=11701MB:tmp=0MB] span[hosts=1]",
					"-R", "select[mem>=11701MB]",
					"-R", "select[tmp>=0MB]",
					"-R", "select[ncpus>=4]"})
				mtx.Lock()
				fakejobq[nextjobid] = args[1]
				nextjobid++
				mtx.Unlock()
			case arvadostest.QueuedContainerUUID:
				c.Check(args, check.DeepEquals, []string{
					"-J", arvadostest.QueuedContainerUUID,
					"-n", "4",
					"-D", "11701MB",
					"-R", "rusage[mem=11701MB:tmp=45777MB] span[hosts=1]",
					"-R", "select[mem>=11701MB]",
					"-R", "select[tmp>=45777MB]",
					"-R", "select[ncpus>=4]"})
				mtx.Lock()
				fakejobq[nextjobid] = args[1]
				nextjobid++
				mtx.Unlock()
			case s.crPending.ContainerUUID:
				c.Check(args, check.DeepEquals, []string{
					"-J", s.crPending.ContainerUUID,
					"-n", "2",
					"-D", "352MB",
					"-R", "rusage[mem=352MB:tmp=8448MB] span[hosts=1]",
					"-R", "select[mem>=352MB]",
					"-R", "select[tmp>=8448MB]",
					"-R", "select[ncpus>=2]"})
				mtx.Lock()
				fakejobq[nextjobid] = args[1]
				nextjobid++
				mtx.Unlock()
			case s.crCUDARequest.ContainerUUID:
				c.Check(args, check.DeepEquals, []string{
					"-J", s.crCUDARequest.ContainerUUID,
					"-n", "1",
					"-D", "528MB",
					"-R", "rusage[mem=528MB:tmp=256MB] span[hosts=1]",
					"-R", "select[mem>=528MB]",
					"-R", "select[tmp>=256MB]",
					"-R", "select[ncpus>=1]",
					"-gpu", "num=1"})
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
			c.Check(args, check.DeepEquals, []string{"-u", "all", "-o", "jobid stat job_name pend_reason", "-json"})
			var records []map[string]interface{}
			for jobid, uuid := range fakejobq {
				stat, reason := "RUN", ""
				if uuid == s.crPending.ContainerUUID {
					// The real bjobs output includes a trailing ';' here:
					stat, reason = "PEND", "There are no suitable hosts for the job;"
				}
				records = append(records, map[string]interface{}{
					"JOBID":       fmt.Sprintf("%d", jobid),
					"STAT":        stat,
					"JOB_NAME":    uuid,
					"PEND_REASON": reason,
				})
			}
			out, err := json.Marshal(map[string]interface{}{
				"COMMAND": "bjobs",
				"JOBS":    len(fakejobq),
				"RECORDS": records,
			})
			if err != nil {
				panic(err)
			}
			c.Logf("bjobs out: %s", out)
			return exec.Command("printf", string(out))
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
		// "crTooBig" should never be submitted to lsf because
		// it is bigger than any configured instance type
		if ent, ok := s.disp.lsfqueue.Lookup(s.crTooBig.ContainerUUID); ok {
			c.Errorf("Lookup(crTooBig) == true, ent = %#v", ent)
			break
		}
		// "queuedcontainer" should be running
		if _, ok := s.disp.lsfqueue.Lookup(arvadostest.QueuedContainerUUID); !ok {
			c.Log("Lookup(queuedcontainer) == false")
			continue
		}
		// "crPending" should be pending
		if ent, ok := s.disp.lsfqueue.Lookup(s.crPending.ContainerUUID); !ok {
			c.Logf("Lookup(crPending) == false", ent)
			continue
		}
		// "lockedcontainer" should be cancelled because it
		// has priority 0 (no matching container requests)
		if ent, ok := s.disp.lsfqueue.Lookup(arvadostest.LockedContainerUUID); ok {
			c.Logf("Lookup(lockedcontainer) == true, ent = %#v", ent)
			continue
		}
		var ctr arvados.Container
		if err := s.disp.arvDispatcher.Arv.Get("containers", arvadostest.LockedContainerUUID, nil, &ctr); err != nil {
			c.Logf("error getting container state for %s: %s", arvadostest.LockedContainerUUID, err)
			continue
		} else if ctr.State != arvados.ContainerStateQueued {
			c.Logf("LockedContainer is not in the LSF queue but its arvados record has not been updated to state==Queued (state is %q)", ctr.State)
			continue
		}

		if err := s.disp.arvDispatcher.Arv.Get("containers", s.crTooBig.ContainerUUID, nil, &ctr); err != nil {
			c.Logf("error getting container state for %s: %s", s.crTooBig.ContainerUUID, err)
			continue
		} else if ctr.State != arvados.ContainerStateCancelled {
			c.Logf("container %s is not in the LSF queue but its arvados record has not been updated to state==Cancelled (state is %q)", s.crTooBig.ContainerUUID, ctr.State)
			continue
		} else {
			c.Check(ctr.RuntimeStatus["error"], check.Equals, "constraints not satisfiable by any configured instance type")
		}
		c.Log("reached desired state")
		break
	}
}
