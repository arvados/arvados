// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/container"
	"git.arvados.org/arvados.git/lib/dispatchcloud/scheduler"
	"git.arvados.org/arvados.git/lib/dispatchcloud/sshexecutor"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	defaultPollInterval     = time.Second
	defaultStaleLockTimeout = time.Minute
)

type pool interface {
	scheduler.WorkerPool
	CheckHealth() error
	Instances() []worker.InstanceView
	SetIdleBehavior(cloud.InstanceID, worker.IdleBehavior) error
	KillInstance(id cloud.InstanceID, reason string) error
	Stop()
}

type dispatcher struct {
	Cluster       *arvados.Cluster
	Context       context.Context
	ArvClient     *arvados.Client
	AuthToken     string
	Registry      *prometheus.Registry
	InstanceSetID cloud.InstanceSetID

	logger      logrus.FieldLogger
	instanceSet cloud.InstanceSet
	pool        pool
	queue       scheduler.ContainerQueue
	httpHandler http.Handler
	sshKey      ssh.Signer

	setupOnce sync.Once
	stop      chan struct{}
	stopped   chan struct{}
}

// Start starts the dispatcher. Start can be called multiple times
// with no ill effect.
func (disp *dispatcher) Start() {
	disp.setupOnce.Do(disp.setup)
}

// ServeHTTP implements service.Handler.
func (disp *dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	disp.Start()
	disp.httpHandler.ServeHTTP(w, r)
}

// CheckHealth implements service.Handler.
func (disp *dispatcher) CheckHealth() error {
	disp.Start()
	return disp.pool.CheckHealth()
}

// Done implements service.Handler.
func (disp *dispatcher) Done() <-chan struct{} {
	return disp.stopped
}

// Stop dispatching containers and release resources. Typically used
// in tests.
func (disp *dispatcher) Close() {
	disp.Start()
	select {
	case disp.stop <- struct{}{}:
	default:
	}
	<-disp.stopped
}

// Make a worker.Executor for the given instance.
func (disp *dispatcher) newExecutor(inst cloud.Instance) worker.Executor {
	exr := sshexecutor.New(inst)
	exr.SetTargetPort(disp.Cluster.Containers.CloudVMs.SSHPort)
	exr.SetSigners(disp.sshKey)
	return exr
}

func (disp *dispatcher) typeChooser(ctr *arvados.Container) (arvados.InstanceType, error) {
	return ChooseInstanceType(disp.Cluster, ctr)
}

func (disp *dispatcher) setup() {
	disp.initialize()
	go disp.run()
}

func (disp *dispatcher) initialize() {
	disp.logger = ctxlog.FromContext(disp.Context)

	disp.ArvClient.AuthToken = disp.AuthToken

	if disp.InstanceSetID == "" {
		if strings.HasPrefix(disp.AuthToken, "v2/") {
			disp.InstanceSetID = cloud.InstanceSetID(strings.Split(disp.AuthToken, "/")[1])
		} else {
			// Use some other string unique to this token
			// that doesn't reveal the token itself.
			disp.InstanceSetID = cloud.InstanceSetID(fmt.Sprintf("%x", md5.Sum([]byte(disp.AuthToken))))
		}
	}
	disp.stop = make(chan struct{}, 1)
	disp.stopped = make(chan struct{})

	if key, err := ssh.ParsePrivateKey([]byte(disp.Cluster.Containers.DispatchPrivateKey)); err != nil {
		disp.logger.Fatalf("error parsing configured Containers.DispatchPrivateKey: %s", err)
	} else {
		disp.sshKey = key
	}

	instanceSet, err := newInstanceSet(disp.Cluster, disp.InstanceSetID, disp.logger, disp.Registry)
	if err != nil {
		disp.logger.Fatalf("error initializing driver: %s", err)
	}
	disp.instanceSet = instanceSet
	disp.pool = worker.NewPool(disp.logger, disp.ArvClient, disp.Registry, disp.InstanceSetID, disp.instanceSet, disp.newExecutor, disp.sshKey.PublicKey(), disp.Cluster)
	disp.queue = container.NewQueue(disp.logger, disp.Registry, disp.typeChooser, disp.ArvClient)

	if disp.Cluster.ManagementToken == "" {
		disp.httpHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Management API authentication is not configured", http.StatusForbidden)
		})
	} else {
		mux := httprouter.New()
		mux.HandlerFunc("GET", "/arvados/v1/dispatch/containers", disp.apiContainers)
		mux.HandlerFunc("POST", "/arvados/v1/dispatch/containers/kill", disp.apiContainerKill)
		mux.HandlerFunc("GET", "/arvados/v1/dispatch/instances", disp.apiInstances)
		mux.HandlerFunc("POST", "/arvados/v1/dispatch/instances/hold", disp.apiInstanceHold)
		mux.HandlerFunc("POST", "/arvados/v1/dispatch/instances/drain", disp.apiInstanceDrain)
		mux.HandlerFunc("POST", "/arvados/v1/dispatch/instances/run", disp.apiInstanceRun)
		mux.HandlerFunc("POST", "/arvados/v1/dispatch/instances/kill", disp.apiInstanceKill)
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

func (disp *dispatcher) run() {
	defer close(disp.stopped)
	defer disp.instanceSet.Stop()
	defer disp.pool.Stop()

	staleLockTimeout := time.Duration(disp.Cluster.Containers.StaleLockTimeout)
	if staleLockTimeout == 0 {
		staleLockTimeout = defaultStaleLockTimeout
	}
	pollInterval := time.Duration(disp.Cluster.Containers.CloudVMs.PollInterval)
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	sched := scheduler.New(disp.Context, disp.queue, disp.pool, disp.Registry, staleLockTimeout, pollInterval)
	sched.Start()
	defer sched.Stop()

	<-disp.stop
}

// Management API: all active and queued containers.
func (disp *dispatcher) apiContainers(w http.ResponseWriter, r *http.Request) {
	var resp struct {
		Items []container.QueueEnt `json:"items"`
	}
	qEntries, _ := disp.queue.Entries()
	for _, ent := range qEntries {
		resp.Items = append(resp.Items, ent)
	}
	json.NewEncoder(w).Encode(resp)
}

// Management API: all active instances (cloud VMs).
func (disp *dispatcher) apiInstances(w http.ResponseWriter, r *http.Request) {
	var resp struct {
		Items []worker.InstanceView `json:"items"`
	}
	resp.Items = disp.pool.Instances()
	json.NewEncoder(w).Encode(resp)
}

// Management API: set idle behavior to "hold" for specified instance.
func (disp *dispatcher) apiInstanceHold(w http.ResponseWriter, r *http.Request) {
	disp.apiInstanceIdleBehavior(w, r, worker.IdleBehaviorHold)
}

// Management API: set idle behavior to "drain" for specified instance.
func (disp *dispatcher) apiInstanceDrain(w http.ResponseWriter, r *http.Request) {
	disp.apiInstanceIdleBehavior(w, r, worker.IdleBehaviorDrain)
}

// Management API: set idle behavior to "run" for specified instance.
func (disp *dispatcher) apiInstanceRun(w http.ResponseWriter, r *http.Request) {
	disp.apiInstanceIdleBehavior(w, r, worker.IdleBehaviorRun)
}

// Management API: shutdown/destroy specified instance now.
func (disp *dispatcher) apiInstanceKill(w http.ResponseWriter, r *http.Request) {
	id := cloud.InstanceID(r.FormValue("instance_id"))
	if id == "" {
		httpserver.Error(w, "instance_id parameter not provided", http.StatusBadRequest)
		return
	}
	err := disp.pool.KillInstance(id, "via management API: "+r.FormValue("reason"))
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}

// Management API: send SIGTERM to specified container's crunch-run
// process now.
func (disp *dispatcher) apiContainerKill(w http.ResponseWriter, r *http.Request) {
	uuid := r.FormValue("container_uuid")
	if uuid == "" {
		httpserver.Error(w, "container_uuid parameter not provided", http.StatusBadRequest)
		return
	}
	if !disp.pool.KillContainer(uuid, "via management API: "+r.FormValue("reason")) {
		httpserver.Error(w, "container not found", http.StatusNotFound)
		return
	}
}

func (disp *dispatcher) apiInstanceIdleBehavior(w http.ResponseWriter, r *http.Request, want worker.IdleBehavior) {
	id := cloud.InstanceID(r.FormValue("instance_id"))
	if id == "" {
		httpserver.Error(w, "instance_id parameter not provided", http.StatusBadRequest)
		return
	}
	err := disp.pool.SetIdleBehavior(id, want)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}
