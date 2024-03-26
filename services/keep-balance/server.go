// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/controller/dblock"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/jmoiron/sqlx"
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
	Once                  bool
	CommitConfirmedFields bool
	ChunkPrefix           string
	Logger                logrus.FieldLogger
	Dumper                logrus.FieldLogger

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

	DB *sqlx.DB
}

// CheckHealth implements service.Handler.
func (srv *Server) CheckHealth() error {
	return srv.DB.Ping()
}

// Done implements service.Handler.
func (srv *Server) Done() <-chan struct{} {
	return nil
}

func (srv *Server) run(ctx context.Context) {
	var err error
	if srv.RunOptions.Once {
		_, err = srv.runOnce(ctx)
	} else {
		err = srv.runForever(ctx)
	}
	if err != nil {
		srv.Logger.Error(err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func (srv *Server) runOnce(ctx context.Context) (*Balancer, error) {
	bal := &Balancer{
		DB:             srv.DB,
		Logger:         srv.Logger,
		Dumper:         srv.Dumper,
		Metrics:        srv.Metrics,
		LostBlocksFile: srv.Cluster.Collections.BlobMissingReport,
		ChunkPrefix:    srv.RunOptions.ChunkPrefix,
	}
	var err error
	srv.RunOptions, err = bal.Run(ctx, srv.ArvClient, srv.Cluster, srv.RunOptions)
	return bal, err
}

// RunForever runs forever, or until ctx is cancelled.
func (srv *Server) runForever(ctx context.Context) error {
	logger := srv.Logger

	ticker := time.NewTicker(time.Duration(srv.Cluster.Collections.BalancePeriod))

	sigUSR1 := make(chan os.Signal, 1)
	signal.Notify(sigUSR1, syscall.SIGUSR1)
	defer signal.Stop(sigUSR1)

	logger.Info("acquiring service lock")
	dblock.KeepBalanceService.Lock(ctx, func(context.Context) (*sqlx.DB, error) { return srv.DB, nil })
	defer dblock.KeepBalanceService.Unlock()

	logger.Printf("starting up: will scan every %v and on SIGUSR1", srv.Cluster.Collections.BalancePeriod)

	for {
		if srv.Cluster.Collections.BalancePullLimit < 1 && srv.Cluster.Collections.BalanceTrashLimit < 1 {
			logger.Print("WARNING: Will scan periodically, but no changes will be committed.")
			logger.Print("=======  To commit changes, set BalancePullLimit and BalanceTrashLimit values greater than zero.")
		}

		if !dblock.KeepBalanceService.Check() {
			// context canceled
			return nil
		}
		_, err := srv.runOnce(ctx)
		if err != nil {
			logger.Print("run failed: ", err)
		} else {
			logger.Print("run succeeded")
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			logger.Print("timer went off")
		case <-sigUSR1:
			logger.Print("received SIGUSR1, resetting timer")
			// Reset the timer so we don't start the N+1st
			// run too soon after the Nth run is triggered
			// by SIGUSR1.
			ticker.Reset(time.Duration(srv.Cluster.Collections.BalancePeriod))
		}
		logger.Print("starting next run")
	}
}
