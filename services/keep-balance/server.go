// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

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
	Logger      logrus.FieldLogger
	Dumper      logrus.FieldLogger

	// SafeRendezvousState from the most recent balance operation,
	// or "" if unknown. If this changes from one run to the next,
	// we need to watch out for races. See
	// (*Balancer)ClearTrashLists.
	SafeRendezvousState string
}

type Server struct {
	http.Handler
	Cluster    *arvados.Cluster
	ArvClient  *arvados.Client
	RunOptions RunOptions
	Metrics    *metrics

	Logger logrus.FieldLogger
	Dumper logrus.FieldLogger
}

// CheckHealth implements service.Handler.
func (srv *Server) CheckHealth() error {
	return nil
}

// Start sets up and runs the balancer.
func (srv *Server) Start(ctx context.Context) {
	if srv.RunOptions.Logger == nil {
		srv.RunOptions.Logger = ctxlog.FromContext(ctx)
	}

	srv.Logger = srv.RunOptions.Logger
	srv.Dumper = srv.RunOptions.Dumper

	var err error
	if srv.RunOptions.Once {
		_, err = srv.run()
	} else {
		err = srv.runForever(nil)
	}
	if err != nil {
		srv.Logger.Error(err)
	}
}

func (srv *Server) run() (*Balancer, error) {
	bal := &Balancer{
		Logger:         srv.Logger,
		Dumper:         srv.Dumper,
		Metrics:        srv.Metrics,
		LostBlocksFile: srv.Cluster.Collections.BlobMissingReport,
	}
	var err error
	srv.RunOptions, err = bal.Run(srv.ArvClient, srv.Cluster, srv.RunOptions)
	return bal, err
}

// RunForever runs forever, or (for testing purposes) until the given
// stop channel is ready to receive.
func (srv *Server) runForever(stop <-chan interface{}) error {
	logger := srv.Logger

	ticker := time.NewTicker(time.Duration(srv.Cluster.Collections.BalancePeriod))

	// The unbuffered channel here means we only hear SIGUSR1 if
	// it arrives while we're waiting in select{}.
	sigUSR1 := make(chan os.Signal)
	signal.Notify(sigUSR1, syscall.SIGUSR1)

	logger.Printf("starting up: will scan every %v and on SIGUSR1", srv.Cluster.Collections.BalancePeriod)

	for {
		if !srv.RunOptions.CommitPulls && !srv.RunOptions.CommitTrash {
			logger.Print("WARNING: Will scan periodically, but no changes will be committed.")
			logger.Print("=======  Consider using -commit-pulls and -commit-trash flags.")
		}

		_, err := srv.run()
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
			ticker = time.NewTicker(time.Duration(srv.Cluster.Collections.BalancePeriod))
		}
		logger.Print("starting next run")
	}
}
