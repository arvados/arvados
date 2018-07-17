// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

// Dispatcher service for Crunch that submits containers to the slurm queue.

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	"github.com/coreos/go-systemd/daemon"
)

const initialNiceValue int64 = 10000

var (
	version           = "dev"
	defaultConfigPath = "/etc/arvados/crunch-dispatch-slurm/crunch-dispatch-slurm.yml"
)

type Dispatcher struct {
	*dispatch.Dispatcher
	cluster *arvados.Cluster
	sqCheck *SqueueChecker
	slurm   Slurm

	Client arvados.Client

	SbatchArguments []string
	PollPeriod      arvados.Duration
	PrioritySpread  int64

	// crunch-run command to invoke. The container UUID will be
	// appended. If nil, []string{"crunch-run"} will be used.
	//
	// Example: []string{"crunch-run", "--cgroup-parent-subsystem=memory"}
	CrunchRunCommand []string

	// Extra RAM to reserve (in Bytes) for SLURM job, in addition
	// to the amount specified in the container's RuntimeConstraints
	ReserveExtraRAM int64

	// Minimum time between two attempts to run the same container
	MinRetryPeriod arvados.Duration
}

func main() {
	disp := &Dispatcher{}
	err := disp.Run(os.Args[0], os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}

func (disp *Dispatcher) Run(prog string, args []string) error {
	if err := disp.configure(prog, args); err != nil {
		return err
	}
	disp.setup()
	return disp.run()
}

// configure() loads config files. Tests skip this.
func (disp *Dispatcher) configure(prog string, args []string) error {
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

	err := disp.readConfig(*configPath)
	if err != nil {
		return err
	}

	if disp.CrunchRunCommand == nil {
		disp.CrunchRunCommand = []string{"crunch-run"}
	}

	if disp.PollPeriod == 0 {
		disp.PollPeriod = arvados.Duration(10 * time.Second)
	}

	if disp.Client.APIHost != "" || disp.Client.AuthToken != "" {
		// Copy real configs into env vars so [a]
		// MakeArvadosClient() uses them, and [b] they get
		// propagated to crunch-run via SLURM.
		os.Setenv("ARVADOS_API_HOST", disp.Client.APIHost)
		os.Setenv("ARVADOS_API_TOKEN", disp.Client.AuthToken)
		os.Setenv("ARVADOS_API_HOST_INSECURE", "")
		if disp.Client.Insecure {
			os.Setenv("ARVADOS_API_HOST_INSECURE", "1")
		}
		os.Setenv("ARVADOS_KEEP_SERVICES", strings.Join(disp.Client.KeepServiceURIs, " "))
		os.Setenv("ARVADOS_EXTERNAL_CLIENT", "")
	} else {
		log.Printf("warning: Client credentials missing from config, so falling back on environment variables (deprecated).")
	}

	if *dumpConfig {
		return config.DumpAndExit(disp)
	}

	siteConfig, err := arvados.GetConfig(arvados.DefaultConfigFile)
	if os.IsNotExist(err) {
		log.Printf("warning: no cluster config (%s), proceeding with no node types defined", err)
	} else if err != nil {
		return fmt.Errorf("error loading config: %s", err)
	} else if disp.cluster, err = siteConfig.GetCluster(""); err != nil {
		return fmt.Errorf("config error: %s", err)
	}

	return nil
}

// setup() initializes private fields after configure().
func (disp *Dispatcher) setup() {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error making Arvados client: %v", err)
	}
	arv.Retries = 25

	disp.slurm = &slurmCLI{}
	disp.sqCheck = &SqueueChecker{
		Period:         time.Duration(disp.PollPeriod),
		PrioritySpread: disp.PrioritySpread,
		Slurm:          disp.slurm,
	}
	disp.Dispatcher = &dispatch.Dispatcher{
		Arv:            arv,
		RunContainer:   disp.runContainer,
		PollPeriod:     time.Duration(disp.PollPeriod),
		MinRetryPeriod: time.Duration(disp.MinRetryPeriod),
	}
}

func (disp *Dispatcher) run() error {
	defer disp.sqCheck.Stop()

	if disp.cluster != nil && len(disp.cluster.InstanceTypes) > 0 {
		go dispatchcloud.SlurmNodeTypeFeatureKludge(disp.cluster)
	}

	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}
	go disp.checkSqueueForOrphans()
	return disp.Dispatcher.Run(context.Background())
}

var containerUuidPattern = regexp.MustCompile(`^[a-z0-9]{5}-dz642-[a-z0-9]{15}$`)

// Check the next squeue report, and invoke TrackContainer for all the
// containers in the report. This gives us a chance to cancel slurm
// jobs started by a previous dispatch process that never released
// their slurm allocations even though their container states are
// Cancelled or Complete. See https://dev.arvados.org/issues/10979
func (disp *Dispatcher) checkSqueueForOrphans() {
	for _, uuid := range disp.sqCheck.All() {
		if !containerUuidPattern.MatchString(uuid) {
			continue
		}
		err := disp.TrackContainer(uuid)
		if err != nil {
			log.Printf("checkSqueueForOrphans: TrackContainer(%s): %s", uuid, err)
		}
	}
}

func (disp *Dispatcher) slurmConstraintArgs(container arvados.Container) []string {
	mem := int64(math.Ceil(float64(container.RuntimeConstraints.RAM+container.RuntimeConstraints.KeepCacheRAM+disp.ReserveExtraRAM) / float64(1048576)))

	var disk int64
	for _, m := range container.Mounts {
		if m.Kind == "tmp" {
			disk += m.Capacity
		}
	}
	disk = int64(math.Ceil(float64(disk) / float64(1048576)))
	return []string{
		fmt.Sprintf("--mem=%d", mem),
		fmt.Sprintf("--cpus-per-task=%d", container.RuntimeConstraints.VCPUs),
		fmt.Sprintf("--tmp=%d", disk),
	}
}

func (disp *Dispatcher) sbatchArgs(container arvados.Container) ([]string, error) {
	var args []string
	args = append(args, disp.SbatchArguments...)
	args = append(args, "--job-name="+container.UUID, fmt.Sprintf("--nice=%d", initialNiceValue))

	if disp.cluster == nil {
		// no instance types configured
		args = append(args, disp.slurmConstraintArgs(container)...)
	} else if it, err := dispatchcloud.ChooseInstanceType(disp.cluster, &container); err == dispatchcloud.ErrInstanceTypesNotConfigured {
		// ditto
		args = append(args, disp.slurmConstraintArgs(container)...)
	} else if err != nil {
		return nil, err
	} else {
		// use instancetype constraint instead of slurm mem/cpu/tmp specs
		args = append(args, "--constraint=instancetype="+it.Name)
	}

	if len(container.SchedulingParameters.Partitions) > 0 {
		args = append(args, "--partition="+strings.Join(container.SchedulingParameters.Partitions, ","))
	}

	return args, nil
}

func (disp *Dispatcher) submit(container arvados.Container, crunchRunCommand []string) error {
	// append() here avoids modifying crunchRunCommand's
	// underlying array, which is shared with other goroutines.
	crArgs := append([]string(nil), crunchRunCommand...)
	crArgs = append(crArgs, container.UUID)
	crScript := strings.NewReader(execScript(crArgs))

	disp.sqCheck.L.Lock()
	defer disp.sqCheck.L.Unlock()

	sbArgs, err := disp.sbatchArgs(container)
	if err != nil {
		return err
	}
	log.Printf("running sbatch %+q", sbArgs)
	return disp.slurm.Batch(crScript, sbArgs)
}

// Submit a container to the slurm queue (or resume monitoring if it's
// already in the queue).  Cancel the slurm job if the container's
// priority changes to zero or its state indicates it's no longer
// running.
func (disp *Dispatcher) runContainer(_ *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if ctr.State == dispatch.Locked && !disp.sqCheck.HasUUID(ctr.UUID) {
		log.Printf("Submitting container %s to slurm", ctr.UUID)
		if err := disp.submit(ctr, disp.CrunchRunCommand); err != nil {
			var text string
			if err, ok := err.(dispatchcloud.ConstraintsNotSatisfiableError); ok {
				var logBuf bytes.Buffer
				fmt.Fprintf(&logBuf, "cannot run container %s: %s\n", ctr.UUID, err)
				if len(err.AvailableTypes) == 0 {
					fmt.Fprint(&logBuf, "No instance types are configured.\n")
				} else {
					fmt.Fprint(&logBuf, "Available instance types:\n")
					for _, t := range err.AvailableTypes {
						fmt.Fprintf(&logBuf,
							"Type %q: %d VCPUs, %d RAM, %d Scratch, %f Price\n",
							t.Name, t.VCPUs, t.RAM, t.Scratch, t.Price,
						)
					}
				}
				text = logBuf.String()
				disp.UpdateState(ctr.UUID, dispatch.Cancelled)
			} else {
				text = fmt.Sprintf("Error submitting container %s to slurm: %s", ctr.UUID, err)
			}
			log.Print(text)

			lr := arvadosclient.Dict{"log": arvadosclient.Dict{
				"object_uuid": ctr.UUID,
				"event_type":  "dispatch",
				"properties":  map[string]string{"text": text}}}
			disp.Arv.Create("logs", lr, nil)

			disp.Unlock(ctr.UUID)
			return
		}
	}

	log.Printf("Start monitoring container %v in state %q", ctr.UUID, ctr.State)
	defer log.Printf("Done monitoring container %s", ctr.UUID)

	// If the container disappears from the slurm queue, there is
	// no point in waiting for further dispatch updates: just
	// clean up and return.
	go func(uuid string) {
		for ctx.Err() == nil && disp.sqCheck.HasUUID(uuid) {
		}
		cancel()
	}(ctr.UUID)

	for {
		select {
		case <-ctx.Done():
			// Disappeared from squeue
			if err := disp.Arv.Get("containers", ctr.UUID, nil, &ctr); err != nil {
				log.Printf("Error getting final container state for %s: %s", ctr.UUID, err)
			}
			switch ctr.State {
			case dispatch.Running:
				disp.UpdateState(ctr.UUID, dispatch.Cancelled)
			case dispatch.Locked:
				disp.Unlock(ctr.UUID)
			}
			return
		case updated, ok := <-status:
			if !ok {
				log.Printf("container %s is done: cancel slurm job", ctr.UUID)
				disp.scancel(ctr)
			} else if updated.Priority == 0 {
				log.Printf("container %s has state %q, priority %d: cancel slurm job", ctr.UUID, updated.State, updated.Priority)
				disp.scancel(ctr)
			} else {
				p := int64(updated.Priority)
				if p <= 1000 {
					// API is providing
					// user-assigned priority. If
					// ctrs have equal priority,
					// run the older one first.
					p = int64(p)<<50 - (updated.CreatedAt.UnixNano() >> 14)
				}
				disp.sqCheck.SetPriority(ctr.UUID, p)
			}
		}
	}
}
func (disp *Dispatcher) scancel(ctr arvados.Container) {
	disp.sqCheck.L.Lock()
	err := disp.slurm.Cancel(ctr.UUID)
	disp.sqCheck.L.Unlock()

	if err != nil {
		log.Printf("scancel: %s", err)
		time.Sleep(time.Second)
	} else if disp.sqCheck.HasUUID(ctr.UUID) {
		log.Printf("container %s is still in squeue after scancel", ctr.UUID)
		time.Sleep(time.Second)
	}
}

func (disp *Dispatcher) readConfig(path string) error {
	err := config.LoadFile(disp, path)
	if err != nil && os.IsNotExist(err) && path == defaultConfigPath {
		log.Printf("Config not specified. Continue with default configuration.")
		err = nil
	}
	return err
}
