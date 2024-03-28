// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package lsf

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/controller/dblock"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/lib/dispatchcloud"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/dispatch"
	"git.arvados.org/arvados.git/sdk/go/health"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var DispatchCommand cmd.Handler = service.Command(arvados.ServiceNameDispatchLSF, newHandler)

func newHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) service.Handler {
	ac, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, fmt.Errorf("error initializing client from cluster config: %s", err))
	}
	d := &dispatcher{
		Cluster:   cluster,
		Context:   ctx,
		ArvClient: ac,
		AuthToken: token,
		Registry:  reg,
	}
	go d.Start()
	return d
}

type dispatcher struct {
	Cluster   *arvados.Cluster
	Context   context.Context
	ArvClient *arvados.Client
	AuthToken string
	Registry  *prometheus.Registry

	logger        logrus.FieldLogger
	dbConnector   ctrlctx.DBConnector
	lsfcli        lsfcli
	lsfqueue      lsfqueue
	arvDispatcher *dispatch.Dispatcher
	httpHandler   http.Handler

	initOnce sync.Once
	stop     chan struct{}
	stopped  chan struct{}
}

// Start starts the dispatcher. Start can be called multiple times
// with no ill effect.
func (disp *dispatcher) Start() {
	disp.initOnce.Do(func() {
		disp.init()
		dblock.Dispatch.Lock(context.Background(), disp.dbConnector.GetDB)
		go func() {
			defer dblock.Dispatch.Unlock()
			disp.checkLsfQueueForOrphans()
			err := disp.arvDispatcher.Run(disp.Context)
			if err != nil {
				disp.logger.Error(err)
				disp.Close()
			}
		}()
	})
}

// ServeHTTP implements service.Handler.
func (disp *dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	disp.Start()
	disp.httpHandler.ServeHTTP(w, r)
}

// CheckHealth implements service.Handler.
func (disp *dispatcher) CheckHealth() error {
	disp.Start()
	select {
	case <-disp.stopped:
		return errors.New("stopped")
	default:
		return nil
	}
}

// Done implements service.Handler.
func (disp *dispatcher) Done() <-chan struct{} {
	return disp.stopped
}

// Stop dispatching containers and release resources. Used by tests.
func (disp *dispatcher) Close() {
	disp.Start()
	select {
	case disp.stop <- struct{}{}:
	default:
	}
	<-disp.stopped
}

func (disp *dispatcher) init() {
	disp.logger = ctxlog.FromContext(disp.Context)
	disp.lsfcli.logger = disp.logger
	disp.lsfqueue = lsfqueue{
		logger: disp.logger,
		period: disp.Cluster.Containers.CloudVMs.PollInterval.Duration(),
		lsfcli: &disp.lsfcli,
	}
	disp.ArvClient.AuthToken = disp.AuthToken
	disp.dbConnector = ctrlctx.DBConnector{PostgreSQL: disp.Cluster.PostgreSQL}
	disp.stop = make(chan struct{}, 1)
	disp.stopped = make(chan struct{})

	arv, err := arvadosclient.New(disp.ArvClient)
	if err != nil {
		disp.logger.Fatalf("Error making Arvados client: %v", err)
	}
	arv.Retries = 25
	arv.ApiToken = disp.AuthToken
	disp.arvDispatcher = &dispatch.Dispatcher{
		Arv:            arv,
		Logger:         disp.logger,
		BatchSize:      disp.Cluster.API.MaxItemsPerResponse,
		RunContainer:   disp.runContainer,
		PollPeriod:     time.Duration(disp.Cluster.Containers.CloudVMs.PollInterval),
		MinRetryPeriod: time.Duration(disp.Cluster.Containers.MinRetryPeriod),
	}

	if disp.Cluster.ManagementToken == "" {
		disp.httpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Management API authentication is not configured", http.StatusForbidden)
		})
	} else {
		mux := httprouter.New()
		metricsH := promhttp.HandlerFor(disp.Registry, promhttp.HandlerOpts{
			ErrorLog: disp.logger,
		})
		mux.Handler("GET", "/metrics", metricsH)
		mux.Handler("GET", "/metrics.json", metricsH)
		mux.Handler("GET", "/_health/:check", &health.Handler{
			Token:  disp.Cluster.ManagementToken,
			Prefix: "/_health/",
			Routes: health.Routes{"ping": disp.CheckHealth},
		})
		disp.httpHandler = auth.RequireLiteralToken(disp.Cluster.ManagementToken, mux)
	}
}

func (disp *dispatcher) runContainer(_ *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) error {
	ctx, cancel := context.WithCancel(disp.Context)
	defer cancel()

	if ctr.State != dispatch.Locked {
		// already started by prior invocation
	} else if _, ok := disp.lsfqueue.Lookup(ctr.UUID); !ok {
		if _, err := dispatchcloud.ChooseInstanceType(disp.Cluster, &ctr); errors.As(err, &dispatchcloud.ConstraintsNotSatisfiableError{}) {
			err := disp.arvDispatcher.Arv.Update("containers", ctr.UUID, arvadosclient.Dict{
				"container": map[string]interface{}{
					"runtime_status": map[string]string{
						"error": err.Error(),
					},
				},
			}, nil)
			if err != nil {
				return fmt.Errorf("error setting runtime_status on %s: %s", ctr.UUID, err)
			}
			return disp.arvDispatcher.UpdateState(ctr.UUID, dispatch.Cancelled)
		}
		disp.logger.Printf("Submitting container %s to LSF", ctr.UUID)
		cmd := []string{disp.Cluster.Containers.CrunchRunCommand}
		cmd = append(cmd, "--runtime-engine="+disp.Cluster.Containers.RuntimeEngine)
		cmd = append(cmd, disp.Cluster.Containers.CrunchRunArgumentsList...)
		err := disp.submit(ctr, cmd)
		if err != nil {
			return err
		}
	}

	disp.logger.Printf("Start monitoring container %v in state %q", ctr.UUID, ctr.State)
	defer disp.logger.Printf("Done monitoring container %s", ctr.UUID)

	go func(uuid string) {
		for ctx.Err() == nil {
			_, ok := disp.lsfqueue.Lookup(uuid)
			if !ok {
				// If the container disappears from
				// the lsf queue, there is no point in
				// waiting for further dispatch
				// updates: just clean up and return.
				disp.logger.Printf("container %s job disappeared from LSF queue", uuid)
				cancel()
				return
			}
		}
	}(ctr.UUID)

	for done := false; !done; {
		select {
		case <-ctx.Done():
			// Disappeared from lsf queue
			if err := disp.arvDispatcher.Arv.Get("containers", ctr.UUID, nil, &ctr); err != nil {
				disp.logger.Printf("error getting final container state for %s: %s", ctr.UUID, err)
			}
			switch ctr.State {
			case dispatch.Running:
				disp.arvDispatcher.UpdateState(ctr.UUID, dispatch.Cancelled)
			case dispatch.Locked:
				disp.arvDispatcher.Unlock(ctr.UUID)
			}
			return nil
		case updated, ok := <-status:
			if !ok {
				// status channel is closed, which is
				// how arvDispatcher tells us to stop
				// touching the container record, kill
				// off any remaining LSF processes,
				// etc.
				done = true
				break
			}
			if updated.State != ctr.State {
				disp.logger.Infof("container %s changed state from %s to %s", ctr.UUID, ctr.State, updated.State)
			}
			ctr = updated
			if ctr.Priority < 1 {
				disp.logger.Printf("container %s has state %s, priority %d: cancel lsf job", ctr.UUID, ctr.State, ctr.Priority)
				disp.bkill(ctr)
			} else {
				disp.lsfqueue.SetPriority(ctr.UUID, int64(ctr.Priority))
			}
		}
	}
	disp.logger.Printf("container %s is done", ctr.UUID)

	// Try "bkill" every few seconds until the LSF job disappears
	// from the queue.
	ticker := time.NewTicker(disp.Cluster.Containers.CloudVMs.PollInterval.Duration() / 2)
	defer ticker.Stop()
	for qent, ok := disp.lsfqueue.Lookup(ctr.UUID); ok; _, ok = disp.lsfqueue.Lookup(ctr.UUID) {
		err := disp.lsfcli.Bkill(qent.ID)
		if err != nil {
			disp.logger.Warnf("%s: bkill(%s): %s", ctr.UUID, qent.ID, err)
		}
		<-ticker.C
	}
	return nil
}

func (disp *dispatcher) submit(container arvados.Container, crunchRunCommand []string) error {
	// Start with an empty slice here to ensure append() doesn't
	// modify crunchRunCommand's underlying array
	var crArgs []string
	crArgs = append(crArgs, crunchRunCommand...)
	crArgs = append(crArgs, container.UUID)

	h := hmac.New(sha256.New, []byte(disp.Cluster.SystemRootToken))
	fmt.Fprint(h, container.UUID)
	authsecret := fmt.Sprintf("%x", h.Sum(nil))

	crScript := execScript(crArgs, map[string]string{"GatewayAuthSecret": authsecret})

	bsubArgs, err := disp.bsubArgs(container)
	if err != nil {
		return err
	}
	return disp.lsfcli.Bsub(crScript, bsubArgs, disp.ArvClient)
}

func (disp *dispatcher) bkill(ctr arvados.Container) {
	if qent, ok := disp.lsfqueue.Lookup(ctr.UUID); !ok {
		disp.logger.Debugf("bkill(%s): redundant, job not in queue", ctr.UUID)
	} else if err := disp.lsfcli.Bkill(qent.ID); err != nil {
		disp.logger.Warnf("%s: bkill(%s): %s", ctr.UUID, qent.ID, err)
	}
}

func (disp *dispatcher) bsubArgs(container arvados.Container) ([]string, error) {
	args := []string{"bsub"}

	tmp := int64(math.Ceil(float64(dispatchcloud.EstimateScratchSpace(&container)) / 1048576))
	vcpus := container.RuntimeConstraints.VCPUs
	mem := int64(math.Ceil(float64(container.RuntimeConstraints.RAM+
		container.RuntimeConstraints.KeepCacheRAM+
		int64(disp.Cluster.Containers.ReserveExtraRAM)) / 1048576))

	maxruntime := time.Duration(container.SchedulingParameters.MaxRunTime) * time.Second
	if maxruntime == 0 {
		maxruntime = disp.Cluster.Containers.LSF.MaxRunTimeDefault.Duration()
	}
	if maxruntime > 0 {
		maxruntime += disp.Cluster.Containers.LSF.MaxRunTimeOverhead.Duration()
	}
	maxrunminutes := int64(math.Ceil(float64(maxruntime.Seconds()) / 60))

	repl := map[string]string{
		"%%": "%",
		"%C": fmt.Sprintf("%d", vcpus),
		"%M": fmt.Sprintf("%d", mem),
		"%T": fmt.Sprintf("%d", tmp),
		"%U": container.UUID,
		"%G": fmt.Sprintf("%d", container.RuntimeConstraints.CUDA.DeviceCount),
		"%W": fmt.Sprintf("%d", maxrunminutes),
	}

	re := regexp.MustCompile(`%.`)
	var substitutionErrors string
	argumentTemplate := disp.Cluster.Containers.LSF.BsubArgumentsList
	if container.RuntimeConstraints.CUDA.DeviceCount > 0 {
		argumentTemplate = append(argumentTemplate, disp.Cluster.Containers.LSF.BsubCUDAArguments...)
	}
	for idx, a := range argumentTemplate {
		if idx > 0 && (argumentTemplate[idx-1] == "-W" || argumentTemplate[idx-1] == "-We") && a == "%W" && maxrunminutes == 0 {
			// LSF docs don't specify an argument to "-W"
			// or "-We" that indicates "unknown", so
			// instead we drop the "-W %W" part of the
			// command line entirely when max runtime is
			// unknown.
			args = args[:len(args)-1]
			continue
		}
		args = append(args, re.ReplaceAllStringFunc(a, func(s string) string {
			subst := repl[s]
			if len(subst) == 0 {
				substitutionErrors += fmt.Sprintf("Unknown substitution parameter %s in BsubArgumentsList, ", s)
			}
			return subst
		}))
	}
	if len(substitutionErrors) != 0 {
		return nil, fmt.Errorf("%s", substitutionErrors[:len(substitutionErrors)-2])
	}

	if u := disp.Cluster.Containers.LSF.BsubSudoUser; u != "" {
		args = append([]string{"sudo", "-E", "-u", u}, args...)
	}
	return args, nil
}

// Check the next bjobs report, and invoke TrackContainer for all the
// containers in the report. This gives us a chance to cancel existing
// Arvados LSF jobs (started by a previous dispatch process) that
// never released their LSF job allocations even though their
// container states are Cancelled or Complete. See
// https://dev.arvados.org/issues/10979
func (disp *dispatcher) checkLsfQueueForOrphans() {
	containerUuidPattern := regexp.MustCompile(`^[a-z0-9]{5}-dz642-[a-z0-9]{15}$`)
	for _, uuid := range disp.lsfqueue.All() {
		if !containerUuidPattern.MatchString(uuid) || !strings.HasPrefix(uuid, disp.Cluster.ClusterID) {
			continue
		}
		err := disp.arvDispatcher.TrackContainer(uuid)
		if err != nil {
			disp.logger.Warnf("checkLsfQueueForOrphans: TrackContainer(%s): %s", uuid, err)
		}
	}
}

func execScript(args []string, env map[string]string) []byte {
	s := "#!/bin/sh\n"
	for k, v := range env {
		s += k + `='`
		s += strings.Replace(v, `'`, `'\''`, -1)
		s += `' `
	}
	s += `exec`
	for _, w := range args {
		s += ` '`
		s += strings.Replace(w, `'`, `'\''`, -1)
		s += `'`
	}
	return []byte(s + "\n")
}
