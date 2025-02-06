// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

// Dispatcher service for Crunch that runs containers locally.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/dispatch"
	"github.com/pbnjay/memory"
	"github.com/sirupsen/logrus"
)

var version = "dev"

var (
	runningCmds      map[string]*exec.Cmd
	runningCmdsMutex sync.Mutex
	waitGroup        sync.WaitGroup
	crunchRunCommand string
)

func main() {
	baseLogger := logrus.StandardLogger()
	if os.Getenv("DEBUG") != "" {
		baseLogger.SetLevel(logrus.DebugLevel)
	}
	baseLogger.Formatter = &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	}

	flags := flag.NewFlagSet("crunch-dispatch-local", flag.ExitOnError)

	pollInterval := flags.Int(
		"poll-interval",
		10,
		"Interval in seconds to poll for queued containers")

	flags.StringVar(&crunchRunCommand,
		"crunch-run-command",
		"",
		"Crunch command to run container")

	getVersion := flags.Bool(
		"version",
		false,
		"Print version information and exit.")

	if ok, code := cmd.ParseFlags(flags, os.Args[0], os.Args[1:], "", os.Stderr); !ok {
		os.Exit(code)
	}

	// Print version information if requested
	if *getVersion {
		fmt.Printf("crunch-dispatch-local %s\n", version)
		return
	}

	loader := config.NewLoader(nil, baseLogger)
	cfg, err := loader.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %s\n", err)
		os.Exit(1)
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %s\n", err)
		os.Exit(1)
	}

	if crunchRunCommand == "" {
		crunchRunCommand = cluster.Containers.CrunchRunCommand
	}

	logger := baseLogger.WithField("ClusterID", cluster.ClusterID)
	logger.Printf("crunch-dispatch-local %s started", version)

	runningCmds = make(map[string]*exec.Cmd)

	var client arvados.Client
	client.APIHost = cluster.Services.Controller.ExternalURL.Host
	client.AuthToken = cluster.SystemRootToken
	client.Insecure = cluster.TLS.Insecure

	if client.APIHost != "" || client.AuthToken != "" {
		// Copy real configs into env vars so [a]
		// MakeArvadosClient() uses them, and [b] they get
		// propagated to crunch-run via SLURM.
		os.Setenv("ARVADOS_API_HOST", client.APIHost)
		os.Setenv("ARVADOS_API_TOKEN", client.AuthToken)
		os.Setenv("ARVADOS_API_HOST_INSECURE", "")
		if client.Insecure {
			os.Setenv("ARVADOS_API_HOST_INSECURE", "1")
		}
	} else {
		logger.Warnf("Client credentials missing from config, so falling back on environment variables (deprecated).")
	}

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		logger.Errorf("error making Arvados client: %v", err)
		os.Exit(1)
	}
	arv.Retries = 25

	ctx, cancel := context.WithCancel(context.Background())

	localRun := LocalRun{startFunc, make(chan ResourceRequest), make(chan ResourceAlloc), ctx, cluster}

	go localRun.throttle(logger)

	dispatcher := dispatch.Dispatcher{
		Logger:       logger,
		Arv:          arv,
		RunContainer: localRun.run,
		PollPeriod:   time.Duration(*pollInterval) * time.Second,
	}

	err = dispatcher.Run(ctx)
	if err != nil {
		logger.Error(err)
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-c
	logger.Printf("Received %s, shutting down", sig)
	signal.Stop(c)

	cancel()

	runningCmdsMutex.Lock()
	// Finished dispatching; interrupt any crunch jobs that are still running
	for _, cmd := range runningCmds {
		cmd.Process.Signal(os.Interrupt)
	}
	runningCmdsMutex.Unlock()

	// Wait for all running crunch jobs to complete / terminate
	waitGroup.Wait()
}

func startFunc(container arvados.Container, cmd *exec.Cmd) error {
	return cmd.Start()
}

type ResourceAlloc struct {
	uuid     string
	vcpus    int
	ram      int64
	gpuStack string
	gpus     []string
}

type ResourceRequest struct {
	uuid     string
	vcpus    int
	ram      int64
	gpuStack string
	gpus     int
	ready    chan ResourceAlloc
}

type LocalRun struct {
	startCmd         func(container arvados.Container, cmd *exec.Cmd) error
	requestResources chan ResourceRequest
	releaseResources chan ResourceAlloc
	ctx              context.Context
	cluster          *arvados.Cluster
}

func (lr *LocalRun) throttle(logger logrus.FieldLogger) {
	maxVcpus := runtime.NumCPU()
	var maxRam int64 = int64(memory.TotalMemory())

	logger.Infof("AMD_VISIBLE_DEVICES=%v", os.Getenv("AMD_VISIBLE_DEVICES"))
	logger.Infof("CUDA_VISIBLE_DEVICES=%v", os.Getenv("CUDA_VISIBLE_DEVICES"))

	availableCUDAGpus := strings.Split(os.Getenv("CUDA_VISIBLE_DEVICES"), ",")
	availableROCmGpus := strings.Split(os.Getenv("AMD_VISIBLE_DEVICES"), ",")

	gpuStack := ""
	maxGpus := 0
	availableGpus := []string{}

	if maxGpus = len(availableCUDAGpus); maxGpus > 0 {
		gpuStack = "cuda"
		availableGpus = availableCUDAGpus
	} else if maxGpus = len(availableROCmGpus); maxGpus > 0 {
		gpuStack = "rocm"
		availableGpus = availableROCmGpus
	}

	availableVcpus := maxVcpus
	availableRam := maxRam

	pending := []ResourceRequest{}

NextEvent:
	for {
		select {
		case rr := <-lr.requestResources:
			pending = append(pending, rr)

		case rr := <-lr.releaseResources:
			availableVcpus += rr.vcpus
			availableRam += rr.ram
			for _, gpu := range rr.gpus {
				availableGpus = append(availableGpus, gpu)
			}

			logger.Infof("%v released allocation (cpus: %v ram: %v gpus: %v); now available (cpus: %v ram: %v gpus: %v)",
				rr.uuid, rr.vcpus, rr.ram, rr.gpus,
				availableVcpus, availableRam, availableGpus)

		case <-lr.ctx.Done():
			return
		}

		for len(pending) > 0 {
			rr := pending[0]
			if rr.vcpus < 1 || rr.vcpus > maxVcpus {
				logger.Infof("%v requested vcpus %v but maxVcpus is %v", rr.uuid, rr.vcpus, maxVcpus)
				// resource request can never be fulfilled,
				// return a zero struct
				rr.ready <- ResourceAlloc{}
				continue
			}
			if rr.ram < 1 || rr.ram > maxRam {
				logger.Infof("%v requested ram %v but maxRam is %v", rr.uuid, rr.ram, maxRam)
				// resource request can never be fulfilled,
				// return a zero struct
				rr.ready <- ResourceAlloc{}
				continue
			}
			if rr.gpus > maxGpus || (rr.gpus > 0 && rr.gpuStack != gpuStack) {
				logger.Infof("%v requested %v gpus with stack %v but maxGpus is %v and gpuStack is %v", rr.uuid, rr.gpus, rr.gpuStack, maxGpus, gpuStack)
				// resource request can never be fulfilled,
				// return a zero struct
				rr.ready <- ResourceAlloc{}
				continue
			}

			if rr.vcpus > availableVcpus || rr.ram > availableRam || rr.gpus > len(availableGpus) {
				logger.Infof("Insufficient resources to start %v, waiting for next event", rr.uuid)
				// can't be scheduled yet, go up to
				// the top and wait for the next event
				continue NextEvent
			}

			alloc := ResourceAlloc{uuid: rr.uuid, vcpus: rr.vcpus, ram: rr.ram}

			availableVcpus -= rr.vcpus
			availableRam -= rr.ram
			alloc.gpuStack = rr.gpuStack

			for i := 0; i < rr.gpus; i++ {
				alloc.gpus = append(alloc.gpus, availableGpus[len(availableGpus)-1])
				availableGpus = availableGpus[0 : len(availableGpus)-1]
			}
			rr.ready <- alloc

			logger.Infof("%v added allocation (cpus: %v ram: %v gpus: %v); now available (cpus: %v ram: %v gpus: %v)",
				rr.uuid, rr.vcpus, rr.ram, rr.gpus,
				availableVcpus, availableRam, availableGpus)

			// shift array down
			for i := 0; i < len(pending)-1; i++ {
				pending[i] = pending[i+1]
			}
			pending = pending[0 : len(pending)-1]
		}

	}
}

// Run a container.
//
// If the container is Locked, start a new crunch-run process and wait until
// crunch-run completes.  If the priority is set to zero, set an interrupt
// signal to the crunch-run process.
//
// If the container is in any other state, or is not Complete/Cancelled after
// crunch-run terminates, mark the container as Cancelled.
func (lr *LocalRun) run(dispatcher *dispatch.Dispatcher,
	container arvados.Container,
	status <-chan arvados.Container) error {

	uuid := container.UUID

	if container.State == dispatch.Locked {

		gpuStack := container.RuntimeConstraints.GPU.Stack
		gpus := container.RuntimeConstraints.GPU.DeviceCount

		resourceRequest := ResourceRequest{
			uuid:  container.UUID,
			vcpus: container.RuntimeConstraints.VCPUs,
			ram: (container.RuntimeConstraints.RAM +
				container.RuntimeConstraints.KeepCacheRAM +
				int64(lr.cluster.Containers.ReserveExtraRAM)),
			gpuStack: gpuStack,
			gpus:     gpus,
			ready:    make(chan ResourceAlloc)}

		select {
		case lr.requestResources <- resourceRequest:
			break
		case <-lr.ctx.Done():
			return lr.ctx.Err()
		}

		var resourceAlloc ResourceAlloc
		select {
		case resourceAlloc = <-resourceRequest.ready:
		case <-lr.ctx.Done():
			return lr.ctx.Err()
		}

		if resourceAlloc.vcpus == 0 {
			dispatcher.Logger.Warnf("Container resource request %v cannot be fulfilled.", uuid)
			dispatcher.UpdateState(uuid, dispatch.Cancelled)
			return nil
		}

		defer func() {
			lr.releaseResources <- resourceAlloc
		}()

		select {
		case c := <-status:
			// Check for state updates after possibly
			// waiting to be ready-to-run
			if c.Priority == 0 {
				goto Finish
			}
		default:
			break
		}

		waitGroup.Add(1)
		defer waitGroup.Done()

		args := []string{"--runtime-engine=" + lr.cluster.Containers.RuntimeEngine}
		args = append(args, lr.cluster.Containers.CrunchRunArgumentsList...)
		args = append(args, uuid)

		cmd := exec.Command(crunchRunCommand, args...)
		cmd.Stdin = nil
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stderr

		cmd.Env = append(cmd.Env, fmt.Sprintf("PATH=%v", os.Getenv("PATH")))
		cmd.Env = append(cmd.Env, fmt.Sprintf("TMPDIR=%v", os.Getenv("TMPDIR")))
		cmd.Env = append(cmd.Env, fmt.Sprintf("ARVADOS_API_HOST=%v", os.Getenv("ARVADOS_API_HOST")))
		cmd.Env = append(cmd.Env, fmt.Sprintf("ARVADOS_API_TOKEN=%v", os.Getenv("ARVADOS_API_TOKEN")))

		h := hmac.New(sha256.New, []byte(lr.cluster.SystemRootToken))
		fmt.Fprint(h, container.UUID)
		cmd.Env = append(cmd.Env, fmt.Sprintf("GatewayAuthSecret=%x", h.Sum(nil)))

		if resourceAlloc.gpuStack == "rocm" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("AMD_VISIBLE_DEVICES=%v", strings.Join(resourceAlloc.gpus, ",")))
		}
		if resourceAlloc.gpuStack == "cuda" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("CUDA_VISIBLE_DEVICES=%v", strings.Join(resourceAlloc.gpus, ",")))
		}

		dispatcher.Logger.Printf("starting container %v", uuid)

		// Add this crunch job to the list of runningCmds only if we
		// succeed in starting crunch-run.

		runningCmdsMutex.Lock()
		if err := lr.startCmd(container, cmd); err != nil {
			runningCmdsMutex.Unlock()
			dispatcher.Logger.Warnf("error starting %q for %s: %s", crunchRunCommand, uuid, err)
			dispatcher.UpdateState(uuid, dispatch.Cancelled)
		} else {
			runningCmds[uuid] = cmd
			runningCmdsMutex.Unlock()

			// Need to wait for crunch-run to exit
			done := make(chan struct{})

			go func() {
				if _, err := cmd.Process.Wait(); err != nil {
					dispatcher.Logger.Warnf("error while waiting for crunch job to finish for %v: %q", uuid, err)
				}
				dispatcher.Logger.Debugf("sending done")
				done <- struct{}{}
			}()

		Loop:
			for {
				select {
				case <-done:
					break Loop
				case c := <-status:
					// Interrupt the child process if priority changes to 0
					if (c.State == dispatch.Locked || c.State == dispatch.Running) && c.Priority == 0 {
						dispatcher.Logger.Printf("sending SIGINT to pid %d to cancel container %v", cmd.Process.Pid, uuid)
						cmd.Process.Signal(os.Interrupt)
					}
				}
			}
			close(done)

			dispatcher.Logger.Printf("finished container run for %v", uuid)

			// Remove the crunch job from runningCmds
			runningCmdsMutex.Lock()
			delete(runningCmds, uuid)
			runningCmdsMutex.Unlock()
		}
	}

Finish:

	// If the container is not finalized, then change it to "Cancelled".
	err := dispatcher.Arv.Get("containers", uuid, nil, &container)
	if err != nil {
		dispatcher.Logger.Warnf("error getting final container state: %v", err)
	}
	if container.State == dispatch.Locked || container.State == dispatch.Running {
		dispatcher.Logger.Warnf("after %q process termination, container state for %v is %q; updating it to %q",
			crunchRunCommand, uuid, container.State, dispatch.Cancelled)
		dispatcher.UpdateState(uuid, dispatch.Cancelled)
	}

	// drain any subsequent status changes
	for range status {
	}

	dispatcher.Logger.Printf("finalized container %v", uuid)
	return nil
}
