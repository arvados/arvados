// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/container"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/scheduler"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/ssh_executor"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/worker"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/ssh"
)

const (
	defaultPollInterval = time.Second
)

type containerQueue interface {
	scheduler.ContainerQueue
	Update() error
}

type pool interface {
	scheduler.WorkerPool
	View() []worker.View
}

type dispatcher struct {
	Cluster       *arvados.Cluster
	InstanceSetID cloud.InstanceSetID

	logger       logrus.FieldLogger
	reg          *prometheus.Registry
	instanceSet  cloud.InstanceSet
	pool         pool
	queue        containerQueue
	httpHandler  http.Handler
	pollInterval time.Duration
	sshKey       ssh.Signer

	setupOnce sync.Once
	stop      chan struct{}
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
	return nil
}

// Stop dispatching containers and release resources. Typically used
// in tests.
func (disp *dispatcher) Close() {
	disp.Start()
	select {
	case disp.stop <- struct{}{}:
	default:
	}
}

// Make a worker.Executor for the given instance.
func (disp *dispatcher) newExecutor(inst cloud.Instance) worker.Executor {
	exr := ssh_executor.New(inst)
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
	arvClient := arvados.NewClientFromEnv()
	if disp.InstanceSetID == "" {
		if strings.HasPrefix(arvClient.AuthToken, "v2/") {
			disp.InstanceSetID = cloud.InstanceSetID(strings.Split(arvClient.AuthToken, "/")[1])
		} else {
			// Use some other string unique to this token
			// that doesn't reveal the token itself.
			disp.InstanceSetID = cloud.InstanceSetID(fmt.Sprintf("%x", md5.Sum([]byte(arvClient.AuthToken))))
		}
	}
	disp.stop = make(chan struct{}, 1)
	disp.logger = logrus.StandardLogger()

	if key, err := ssh.ParsePrivateKey(disp.Cluster.Dispatch.PrivateKey); err != nil {
		disp.logger.Fatalf("error parsing configured Dispatch.PrivateKey: %s", err)
	} else {
		disp.sshKey = key
	}

	instanceSet, err := newInstanceSet(disp.Cluster, disp.InstanceSetID)
	if err != nil {
		disp.logger.Fatalf("error initializing driver: %s", err)
	}
	disp.instanceSet = &instanceSetProxy{instanceSet}
	disp.reg = prometheus.NewRegistry()
	disp.pool = worker.NewPool(disp.logger, disp.reg, disp.instanceSet, disp.newExecutor, disp.Cluster)
	disp.queue = container.NewQueue(disp.logger, disp.reg, disp.typeChooser, arvClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/arvados/v1/dispatch/containers", disp.apiContainers)
	mux.HandleFunc("/arvados/v1/dispatch/instances", disp.apiInstances)
	metricsH := promhttp.HandlerFor(disp.reg, promhttp.HandlerOpts{
		ErrorLog: disp.logger,
	})
	mux.Handle("/metrics", metricsH)
	mux.Handle("/metrics.json", metricsH)
	disp.httpHandler = auth.RequireLiteralToken(disp.Cluster.ManagementToken, mux)

	if d := disp.Cluster.Dispatch.PollInterval; d > 0 {
		disp.pollInterval = time.Duration(d)
	} else {
		disp.pollInterval = defaultPollInterval
	}
}

func (disp *dispatcher) run() {
	defer disp.instanceSet.Stop()

	t0 := time.Now()
	disp.logger.Infof("FixStaleLocks starting.")
	scheduler.FixStaleLocks(disp.logger, disp.queue, disp.pool, time.Duration(disp.Cluster.Dispatch.StaleLockTimeout))
	disp.logger.Infof("FixStaleLocks finished (%s), starting scheduling.", time.Since(t0))

	wp := disp.pool.Subscribe()
	defer disp.pool.Unsubscribe(wp)
	poll := time.NewTicker(disp.pollInterval)
	for {
		scheduler.Map(disp.logger, disp.queue, disp.pool)
		scheduler.Sync(disp.logger, disp.queue, disp.pool)
		select {
		case <-disp.stop:
			return
		case <-wp:
		case <-poll.C:
			err := disp.queue.Update()
			if err != nil {
				disp.logger.Errorf("error updating queue: %s", err)
			}
		}
	}
}

// Management API: all active and queued containers.
func (disp *dispatcher) apiContainers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpserver.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var resp struct {
		Items []container.QueueEnt
	}
	qEntries, _ := disp.queue.Entries()
	for _, ent := range qEntries {
		resp.Items = append(resp.Items, ent)
	}
	json.NewEncoder(w).Encode(resp)
}

// Management API: all active instances (cloud VMs).
func (disp *dispatcher) apiInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		httpserver.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var resp struct {
		Items []worker.View
	}
	resp.Items = disp.pool.View()
	json.NewEncoder(w).Encode(resp)
}
