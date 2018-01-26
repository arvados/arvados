// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

// Dispatcher service for Crunch that submits containers to the slurm queue.

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	"git.curoverse.com/arvados.git/services/dispatchcloud"
	"github.com/coreos/go-systemd/daemon"
)

var version = "dev"

type command struct {
	dispatcher *dispatch.Dispatcher
	cluster    *arvados.Cluster
	sqCheck    *SqueueChecker
	slurm      Slurm

	Client arvados.Client

	SbatchArguments []string
	PollPeriod      arvados.Duration

	// crunch-run command to invoke. The container UUID will be
	// appended. If nil, []string{"crunch-run"} will be used.
	//
	// Example: []string{"crunch-run", "--cgroup-parent-subsystem=memory"}
	CrunchRunCommand []string

	// Minimum time between two attempts to run the same container
	MinRetryPeriod arvados.Duration
}

func main() {
	err := (&command{}).Run(os.Args[0], os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}

const defaultConfigPath = "/etc/arvados/crunch-dispatch-slurm/crunch-dispatch-slurm.yml"

func (cmd *command) Run(prog string, args []string) error {
	flags := flag.NewFlagSet(prog, flag.ExitOnError)
	flags.Usage = func() { usage(flags) }

	configPath := flags.String(
		"config",
		defaultConfigPath,
		"`path` to JSON or YAML configuration file")
	dumpConfig := flag.Bool(
		"dump-config",
		false,
		"write current configuration to stdout and exit")
	getVersion := flags.Bool(
		"version",
		false,
		"Print version information and exit.")
	// Parse args; omit the first arg which is the command name
	flags.Parse(args)

	// Print version information if requested
	if *getVersion {
		fmt.Printf("crunch-dispatch-slurm %s\n", version)
		return nil
	}

	log.Printf("crunch-dispatch-slurm %s started", version)

	err := cmd.readConfig(*configPath)
	if err != nil {
		return err
	}

	if cmd.CrunchRunCommand == nil {
		cmd.CrunchRunCommand = []string{"crunch-run"}
	}

	if cmd.PollPeriod == 0 {
		cmd.PollPeriod = arvados.Duration(10 * time.Second)
	}

	if cmd.Client.APIHost != "" || cmd.Client.AuthToken != "" {
		// Copy real configs into env vars so [a]
		// MakeArvadosClient() uses them, and [b] they get
		// propagated to crunch-run via SLURM.
		os.Setenv("ARVADOS_API_HOST", cmd.Client.APIHost)
		os.Setenv("ARVADOS_API_TOKEN", cmd.Client.AuthToken)
		os.Setenv("ARVADOS_API_HOST_INSECURE", "")
		if cmd.Client.Insecure {
			os.Setenv("ARVADOS_API_HOST_INSECURE", "1")
		}
		os.Setenv("ARVADOS_KEEP_SERVICES", strings.Join(cmd.Client.KeepServiceURIs, " "))
		os.Setenv("ARVADOS_EXTERNAL_CLIENT", "")
	} else {
		log.Printf("warning: Client credentials missing from config, so falling back on environment variables (deprecated).")
	}

	if *dumpConfig {
		log.Fatal(config.DumpAndExit(cmd))
	}

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Printf("Error making Arvados client: %v", err)
		return err
	}
	arv.Retries = 25

	siteConfig, err := arvados.GetConfig(arvados.DefaultConfigFile)
	if os.IsNotExist(err) {
		log.Printf("warning: no cluster config file %q (%s), proceeding with no node types defined", arvados.DefaultConfigFile, err)
	} else if err != nil {
		log.Fatalf("error loading config: %s", err)
	} else if cmd.cluster, err = siteConfig.GetCluster(""); err != nil {
		log.Fatalf("config error: %s", err)
	}

	if cmd.slurm == nil {
		cmd.slurm = &slurmCLI{}
	}

	cmd.sqCheck = &SqueueChecker{
		Period: time.Duration(cmd.PollPeriod),
		Slurm:  cmd.slurm,
	}
	defer cmd.sqCheck.Stop()

	cmd.dispatcher = &dispatch.Dispatcher{
		Arv:            arv,
		RunContainer:   cmd.run,
		PollPeriod:     time.Duration(cmd.PollPeriod),
		MinRetryPeriod: time.Duration(cmd.MinRetryPeriod),
	}

	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}

	go cmd.checkSqueueForOrphans()

	return cmd.dispatcher.Run(context.Background())
}

var containerUuidPattern = regexp.MustCompile(`^[a-z0-9]{5}-dz642-[a-z0-9]{15}$`)

// Check the next squeue report, and invoke TrackContainer for all the
// containers in the report. This gives us a chance to cancel slurm
// jobs started by a previous dispatch process that never released
// their slurm allocations even though their container states are
// Cancelled or Complete. See https://dev.arvados.org/issues/10979
func (cmd *command) checkSqueueForOrphans() {
	for _, uuid := range cmd.sqCheck.All() {
		if !containerUuidPattern.MatchString(uuid) {
			continue
		}
		err := cmd.dispatcher.TrackContainer(uuid)
		if err != nil {
			log.Printf("checkSqueueForOrphans: TrackContainer(%s): %s", uuid, err)
		}
	}
}

func (cmd *command) niceness(priority int) int {
	if priority > 1000 {
		priority = 1000
	}
	if priority < 0 {
		priority = 0
	}
	// Niceness range 1-10000
	return (1000 - priority) * 10
}

func (cmd *command) sbatchArgs(container arvados.Container) ([]string, error) {
	mem := int64(math.Ceil(float64(container.RuntimeConstraints.RAM+container.RuntimeConstraints.KeepCacheRAM) / float64(1048576)))

	var disk int64
	for _, m := range container.Mounts {
		if m.Kind == "tmp" {
			disk += m.Capacity
		}
	}
	disk = int64(math.Ceil(float64(disk) / float64(1048576)))

	var sbatchArgs []string
	sbatchArgs = append(sbatchArgs, cmd.SbatchArguments...)
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--job-name=%s", container.UUID))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--mem=%d", mem))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--cpus-per-task=%d", container.RuntimeConstraints.VCPUs))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--tmp=%d", disk))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--nice=%d", cmd.niceness(container.Priority)))
	if len(container.SchedulingParameters.Partitions) > 0 {
		sbatchArgs = append(sbatchArgs, fmt.Sprintf("--partition=%s", strings.Join(container.SchedulingParameters.Partitions, ",")))
	}

	return sbatchArgs, nil
}

func (cmd *command) submit(container arvados.Container, crunchRunCommand []string) error {
	// append() here avoids modifying crunchRunCommand's
	// underlying array, which is shared with other goroutines.
	crArgs := append([]string(nil), crunchRunCommand...)
	crArgs = append(crArgs, container.UUID)
	crScript := strings.NewReader(execScript(crArgs))

	cmd.sqCheck.L.Lock()
	defer cmd.sqCheck.L.Unlock()

	sbArgs, err := cmd.sbatchArgs(container)
	if err != nil {
		return err
	}
	log.Printf("running sbatch %+q", sbArgs)
	return cmd.slurm.Batch(crScript, sbArgs)
}

// Submit a container to the slurm queue (or resume monitoring if it's
// already in the queue).  Cancel the slurm job if the container's
// priority changes to zero or its state indicates it's no longer
// running.
func (cmd *command) run(_ *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if ctr.State == dispatch.Locked && !cmd.sqCheck.HasUUID(ctr.UUID) {
		log.Printf("Submitting container %s to slurm", ctr.UUID)
		if err := cmd.submit(ctr, cmd.CrunchRunCommand); err != nil {
			text := fmt.Sprintf("Error submitting container %s to slurm: %s", ctr.UUID, err)
			log.Print(text)

			lr := arvadosclient.Dict{"log": arvadosclient.Dict{
				"object_uuid": ctr.UUID,
				"event_type":  "dispatch",
				"properties":  map[string]string{"text": text}}}
			cmd.dispatcher.Arv.Create("logs", lr, nil)

			cmd.dispatcher.Unlock(ctr.UUID)
			return
		}
	}

	log.Printf("Start monitoring container %v in state %q", ctr.UUID, ctr.State)
	defer log.Printf("Done monitoring container %s", ctr.UUID)

	// If the container disappears from the slurm queue, there is
	// no point in waiting for further dispatch updates: just
	// clean up and return.
	go func(uuid string) {
		for ctx.Err() == nil && cmd.sqCheck.HasUUID(uuid) {
		}
		cancel()
	}(ctr.UUID)

	for {
		select {
		case <-ctx.Done():
			// Disappeared from squeue
			if err := cmd.dispatcher.Arv.Get("containers", ctr.UUID, nil, &ctr); err != nil {
				log.Printf("Error getting final container state for %s: %s", ctr.UUID, err)
			}
			switch ctr.State {
			case dispatch.Running:
				cmd.dispatcher.UpdateState(ctr.UUID, dispatch.Cancelled)
			case dispatch.Locked:
				cmd.dispatcher.Unlock(ctr.UUID)
			}
			return
		case updated, ok := <-status:
			if !ok {
				log.Printf("Dispatcher says container %s is done: cancel slurm job", ctr.UUID)
				cmd.scancel(ctr)
			} else if updated.Priority == 0 {
				log.Printf("Container %s has state %q, priority %d: cancel slurm job", ctr.UUID, updated.State, updated.Priority)
				cmd.scancel(ctr)
			} else {
				cmd.renice(updated)
			}
		}
	}
}

func (cmd *command) scancel(ctr arvados.Container) {
	cmd.sqCheck.L.Lock()
	err := cmd.slurm.Cancel(ctr.UUID)
	cmd.sqCheck.L.Unlock()

	if err != nil {
		log.Printf("scancel: %s", err)
		time.Sleep(time.Second)
	} else if cmd.sqCheck.HasUUID(ctr.UUID) {
		log.Printf("container %s is still in squeue after scancel", ctr.UUID)
		time.Sleep(time.Second)
	}
}

func (cmd *command) renice(ctr arvados.Container) {
	nice := cmd.niceness(ctr.Priority)
	oldnice := cmd.sqCheck.GetNiceness(ctr.UUID)
	if nice == oldnice || oldnice == -1 {
		return
	}
	log.Printf("updating slurm nice value to %d (was %d)", nice, oldnice)
	cmd.sqCheck.L.Lock()
	err := cmd.slurm.Renice(ctr.UUID, nice)
	cmd.sqCheck.L.Unlock()

	if err != nil {
		log.Printf("renice: %s", err)
		time.Sleep(time.Second)
		return
	}
	if cmd.sqCheck.HasUUID(ctr.UUID) {
		log.Printf("container %s has arvados priority %d, slurm nice %d",
			ctr.UUID, ctr.Priority, cmd.sqCheck.GetNiceness(ctr.UUID))
	}
}

func (cmd *command) readConfig(path string) error {
	err := config.LoadFile(cmd, path)
	if err != nil && os.IsNotExist(err) && path == defaultConfigPath {
		log.Printf("Config not specified. Continue with default configuration.")
		err = nil
	}
	return err
}
