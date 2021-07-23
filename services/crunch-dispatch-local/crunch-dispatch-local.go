// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

// Dispatcher service for Crunch that runs containers locally.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/dispatch"
	"github.com/sirupsen/logrus"
)

var version = "dev"

func main() {
	err := doMain()
	if err != nil {
		logrus.Fatalf("%q", err)
	}
}

var (
	runningCmds      map[string]*exec.Cmd
	runningCmdsMutex sync.Mutex
	waitGroup        sync.WaitGroup
	crunchRunCommand *string
)

func doMain() error {
	logger := logrus.StandardLogger()
	if os.Getenv("DEBUG") != "" {
		logger.SetLevel(logrus.DebugLevel)
	}
	logger.Formatter = &logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	}

	flags := flag.NewFlagSet("crunch-dispatch-local", flag.ExitOnError)

	pollInterval := flags.Int(
		"poll-interval",
		10,
		"Interval in seconds to poll for queued containers")

	crunchRunCommand = flags.String(
		"crunch-run-command",
		"/usr/bin/crunch-run",
		"Crunch command to run container")

	getVersion := flags.Bool(
		"version",
		false,
		"Print version information and exit.")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	// Print version information if requested
	if *getVersion {
		fmt.Printf("crunch-dispatch-local %s\n", version)
		return nil
	}

	loader := config.NewLoader(nil, logger)
	cfg, err := loader.Load()
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return fmt.Errorf("config error: %s", err)
	}

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
		os.Setenv("ARVADOS_EXTERNAL_CLIENT", "")
	} else {
		logger.Warnf("Client credentials missing from config, so falling back on environment variables (deprecated).")
	}

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		logger.Errorf("error making Arvados client: %v", err)
		return err
	}
	arv.Retries = 25

	ctx, cancel := context.WithCancel(context.Background())

	dispatcher := dispatch.Dispatcher{
		Logger:       logger,
		Arv:          arv,
		RunContainer: (&LocalRun{startFunc, make(chan bool, 8), ctx, cluster}).run,
		PollPeriod:   time.Duration(*pollInterval) * time.Second,
	}

	err = dispatcher.Run(ctx)
	if err != nil {
		return err
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

	return nil
}

func startFunc(container arvados.Container, cmd *exec.Cmd) error {
	return cmd.Start()
}

type LocalRun struct {
	startCmd         func(container arvados.Container, cmd *exec.Cmd) error
	concurrencyLimit chan bool
	ctx              context.Context
	cluster          *arvados.Cluster
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
	status <-chan arvados.Container) {

	uuid := container.UUID

	if container.State == dispatch.Locked {

		select {
		case lr.concurrencyLimit <- true:
			break
		case <-lr.ctx.Done():
			return
		}

		defer func() { <-lr.concurrencyLimit }()

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

		cmd := exec.Command(*crunchRunCommand, "--runtime-engine="+lr.cluster.Containers.RuntimeEngine, uuid)
		cmd.Stdin = nil
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stderr

		dispatcher.Logger.Printf("starting container %v", uuid)

		// Add this crunch job to the list of runningCmds only if we
		// succeed in starting crunch-run.

		runningCmdsMutex.Lock()
		if err := lr.startCmd(container, cmd); err != nil {
			runningCmdsMutex.Unlock()
			dispatcher.Logger.Warnf("error starting %q for %s: %s", *crunchRunCommand, uuid, err)
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
			*crunchRunCommand, uuid, container.State, dispatch.Cancelled)
		dispatcher.UpdateState(uuid, dispatch.Cancelled)
	}

	// drain any subsequent status changes
	for range status {
	}

	dispatcher.Logger.Printf("finalized container %v", uuid)
}
