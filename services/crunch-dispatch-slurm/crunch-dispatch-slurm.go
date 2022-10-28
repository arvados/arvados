// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Dispatcher service for Crunch that submits containers to the slurm queue.
package dispatchslurm

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/controller/dblock"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/lib/dispatchcloud"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/dispatch"
	"github.com/coreos/go-systemd/daemon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var Command cmd.Handler = service.Command(arvados.ServiceNameDispatchSLURM, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, _ string, _ *prometheus.Registry) service.Handler {
	logger := ctxlog.FromContext(ctx)
	disp := &Dispatcher{logger: logger, cluster: cluster}
	if err := disp.configure(); err != nil {
		return service.ErrorHandler(ctx, cluster, err)
	}
	disp.setup()
	go func() {
		disp.err = disp.run()
		close(disp.done)
	}()
	return disp
}

type logger interface {
	dispatch.Logger
	Fatalf(string, ...interface{})
}

const initialNiceValue int64 = 10000

type Dispatcher struct {
	*dispatch.Dispatcher
	logger      logrus.FieldLogger
	cluster     *arvados.Cluster
	sqCheck     *SqueueChecker
	slurm       Slurm
	dbConnector ctrlctx.DBConnector

	done chan struct{}
	err  error

	Client arvados.Client
}

func (disp *Dispatcher) CheckHealth() error {
	return disp.err
}

func (disp *Dispatcher) Done() <-chan struct{} {
	return disp.done
}

func (disp *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// configure() loads config files. Some tests skip this (see
// StubbedSuite).
func (disp *Dispatcher) configure() error {
	if disp.logger == nil {
		disp.logger = logrus.StandardLogger()
	}
	disp.logger = disp.logger.WithField("ClusterID", disp.cluster.ClusterID)
	disp.logger.Printf("crunch-dispatch-slurm %s started", cmd.Version.String())

	disp.Client.APIHost = disp.cluster.Services.Controller.ExternalURL.Host
	disp.Client.AuthToken = disp.cluster.SystemRootToken
	disp.Client.Insecure = disp.cluster.TLS.Insecure
	disp.dbConnector = ctrlctx.DBConnector{PostgreSQL: disp.cluster.PostgreSQL}

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
		for k, v := range disp.cluster.Containers.SLURM.SbatchEnvironmentVariables {
			os.Setenv(k, v)
		}
	} else {
		disp.logger.Warnf("Client credentials missing from config, so falling back on environment variables (deprecated).")
	}
	return nil
}

// setup() initializes private fields after configure().
func (disp *Dispatcher) setup() {
	disp.done = make(chan struct{})
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		disp.logger.Fatalf("Error making Arvados client: %v", err)
	}
	arv.Retries = 25

	disp.slurm = NewSlurmCLI()
	disp.sqCheck = &SqueueChecker{
		Logger:         disp.logger,
		Period:         time.Duration(disp.cluster.Containers.CloudVMs.PollInterval),
		PrioritySpread: disp.cluster.Containers.SLURM.PrioritySpread,
		Slurm:          disp.slurm,
	}
	disp.Dispatcher = &dispatch.Dispatcher{
		Arv:            arv,
		Logger:         disp.logger,
		BatchSize:      disp.cluster.API.MaxItemsPerResponse,
		RunContainer:   disp.runContainer,
		PollPeriod:     time.Duration(disp.cluster.Containers.CloudVMs.PollInterval),
		MinRetryPeriod: time.Duration(disp.cluster.Containers.MinRetryPeriod),
	}
}

func (disp *Dispatcher) run() error {
	dblock.Dispatch.Lock(context.Background(), disp.dbConnector.GetDB)
	defer dblock.Dispatch.Unlock()
	defer disp.sqCheck.Stop()

	if disp.cluster != nil && len(disp.cluster.InstanceTypes) > 0 {
		go SlurmNodeTypeFeatureKludge(disp.cluster)
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
		if !containerUuidPattern.MatchString(uuid) || !strings.HasPrefix(uuid, disp.cluster.ClusterID) {
			continue
		}
		err := disp.TrackContainer(uuid)
		if err != nil {
			log.Printf("checkSqueueForOrphans: TrackContainer(%s): %s", uuid, err)
		}
	}
}

func (disp *Dispatcher) slurmConstraintArgs(container arvados.Container) []string {
	mem := int64(math.Ceil(float64(container.RuntimeConstraints.RAM+
		container.RuntimeConstraints.KeepCacheRAM+
		int64(disp.cluster.Containers.ReserveExtraRAM)) / float64(1048576)))

	disk := dispatchcloud.EstimateScratchSpace(&container)
	disk = int64(math.Ceil(float64(disk) / float64(1048576)))
	return []string{
		fmt.Sprintf("--mem=%d", mem),
		fmt.Sprintf("--cpus-per-task=%d", container.RuntimeConstraints.VCPUs),
		fmt.Sprintf("--tmp=%d", disk),
	}
}

func (disp *Dispatcher) sbatchArgs(container arvados.Container) ([]string, error) {
	var args []string
	args = append(args, disp.cluster.Containers.SLURM.SbatchArgumentsList...)
	args = append(args, "--job-name="+container.UUID, fmt.Sprintf("--nice=%d", initialNiceValue), "--no-requeue")

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
	crArgs = append(crArgs, "--runtime-engine="+disp.cluster.Containers.RuntimeEngine)
	crArgs = append(crArgs, container.UUID)

	h := hmac.New(sha256.New, []byte(disp.cluster.SystemRootToken))
	fmt.Fprint(h, container.UUID)
	authsecret := fmt.Sprintf("%x", h.Sum(nil))

	crScript := strings.NewReader(execScript(crArgs, map[string]string{"GatewayAuthSecret": authsecret}))

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
func (disp *Dispatcher) runContainer(_ *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if ctr.State == dispatch.Locked && !disp.sqCheck.HasUUID(ctr.UUID) {
		log.Printf("Submitting container %s to slurm", ctr.UUID)
		cmd := []string{disp.cluster.Containers.CrunchRunCommand}
		cmd = append(cmd, disp.cluster.Containers.CrunchRunArgumentsList...)
		err := disp.submit(ctr, cmd)
		if err != nil {
			return err
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
				log.Printf("error getting final container state for %s: %s", ctr.UUID, err)
			}
			switch ctr.State {
			case dispatch.Running:
				disp.UpdateState(ctr.UUID, dispatch.Cancelled)
			case dispatch.Locked:
				disp.Unlock(ctr.UUID)
			}
			return nil
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
	err := disp.slurm.Cancel(ctr.UUID)
	if err != nil {
		log.Printf("scancel: %s", err)
		time.Sleep(time.Second)
	} else if disp.sqCheck.HasUUID(ctr.UUID) {
		log.Printf("container %s is still in squeue after scancel", ctr.UUID)
		time.Sleep(time.Second)
	}
}
