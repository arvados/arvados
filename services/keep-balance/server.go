// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"github.com/Sirupsen/logrus"
)

var version = "dev"

const (
	defaultConfigPath = "/etc/arvados/keep-balance/keep-balance.yml"
	rfc3339NanoFixed  = "2006-01-02T15:04:05.000000000Z07:00"
)

// Config specifies site configuration, like API credentials and the
// choice of which servers are to be balanced.
//
// Config is loaded from a JSON config file (see usage()).
type Config struct {
	// Arvados API endpoint and credentials.
	Client arvados.Client

	// List of service types (e.g., "disk") to balance.
	KeepServiceTypes []string

	KeepServiceList arvados.KeepServiceList

	// address, address:port, or :port for management interface
	Listen string

	// token for management APIs
	ManagementToken string

	// How often to check
	RunPeriod arvados.Duration

	// Number of collections to request in each API call
	CollectionBatchSize int

	// Max collections to buffer in memory (bigger values consume
	// more memory, but can reduce store-and-forward latency when
	// fetching pages)
	CollectionBuffers int

	// Timeout for outgoing http request/response cycle.
	RequestTimeout arvados.Duration

	// Destination filename for the list of lost block hashes, one
	// per line. Updated atomically during each successful run.
	LostBlocksFile string
}

// RunOptions controls runtime behavior. The flags/options that belong
// here are the ones that are useful for interactive use. For example,
// "CommitTrash" is a runtime option rather than a config item because
// it invokes a troubleshooting feature rather than expressing how
// balancing is meant to be done at a given site.
//
// RunOptions fields are controlled by command line flags.
type RunOptions struct {
	Once        bool
	CommitPulls bool
	CommitTrash bool
	Logger      *logrus.Logger
	Dumper      *logrus.Logger

	// SafeRendezvousState from the most recent balance operation,
	// or "" if unknown. If this changes from one run to the next,
	// we need to watch out for races. See
	// (*Balancer)ClearTrashLists.
	SafeRendezvousState string
}

type Server struct {
	config     Config
	runOptions RunOptions
	metrics    *metrics
	listening  string // for tests

	Logger *logrus.Logger
	Dumper *logrus.Logger
}

// NewServer returns a new Server that runs Balancers using the given
// config and runOptions.
func NewServer(config Config, runOptions RunOptions) (*Server, error) {
	if len(config.KeepServiceList.Items) > 0 && config.KeepServiceTypes != nil {
		return nil, fmt.Errorf("cannot specify both KeepServiceList and KeepServiceTypes in config")
	}
	if !runOptions.Once && config.RunPeriod == arvados.Duration(0) {
		return nil, fmt.Errorf("you must either use the -once flag, or specify RunPeriod in config")
	}

	if runOptions.Logger == nil {
		log := logrus.New()
		log.Formatter = &logrus.JSONFormatter{
			TimestampFormat: rfc3339NanoFixed,
		}
		log.Out = os.Stderr
		runOptions.Logger = log
	}

	srv := &Server{
		config:     config,
		runOptions: runOptions,
		metrics:    newMetrics(),
		Logger:     runOptions.Logger,
		Dumper:     runOptions.Dumper,
	}
	return srv, srv.start()
}

func (srv *Server) start() error {
	if srv.config.Listen == "" {
		return nil
	}
	server := &httpserver.Server{
		Server: http.Server{
			Handler: httpserver.LogRequests(srv.Logger,
				auth.RequireLiteralToken(srv.config.ManagementToken,
					srv.metrics.Handler(srv.Logger))),
		},
		Addr: srv.config.Listen,
	}
	err := server.Start()
	if err != nil {
		return err
	}
	srv.Logger.Printf("listening at %s", server.Addr)
	srv.listening = server.Addr
	return nil
}

func (srv *Server) Run() (*Balancer, error) {
	bal := &Balancer{
		Logger:         srv.Logger,
		Dumper:         srv.Dumper,
		Metrics:        srv.metrics,
		LostBlocksFile: srv.config.LostBlocksFile,
	}
	var err error
	srv.runOptions, err = bal.Run(srv.config, srv.runOptions)
	return bal, err
}

// RunForever runs forever, or (for testing purposes) until the given
// stop channel is ready to receive.
func (srv *Server) RunForever(stop <-chan interface{}) error {
	logger := srv.runOptions.Logger

	ticker := time.NewTicker(time.Duration(srv.config.RunPeriod))

	// The unbuffered channel here means we only hear SIGUSR1 if
	// it arrives while we're waiting in select{}.
	sigUSR1 := make(chan os.Signal)
	signal.Notify(sigUSR1, syscall.SIGUSR1)

	logger.Printf("starting up: will scan every %v and on SIGUSR1", srv.config.RunPeriod)

	for {
		if !srv.runOptions.CommitPulls && !srv.runOptions.CommitTrash {
			logger.Print("WARNING: Will scan periodically, but no changes will be committed.")
			logger.Print("=======  Consider using -commit-pulls and -commit-trash flags.")
		}

		_, err := srv.Run()
		if err != nil {
			logger.Print("run failed: ", err)
		} else {
			logger.Print("run succeeded")
		}

		select {
		case <-stop:
			signal.Stop(sigUSR1)
			return nil
		case <-ticker.C:
			logger.Print("timer went off")
		case <-sigUSR1:
			logger.Print("received SIGUSR1, resetting timer")
			// Reset the timer so we don't start the N+1st
			// run too soon after the Nth run is triggered
			// by SIGUSR1.
			ticker.Stop()
			ticker = time.NewTicker(time.Duration(srv.config.RunPeriod))
		}
		logger.Print("starting next run")
	}
}
