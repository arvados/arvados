// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

var Command cmd.Handler = bootCommand{}

type supervisedTask interface {
	// Execute the task. Run should return nil when the task is
	// done enough to satisfy a dependency relationship (e.g., the
	// service is running and ready). If the task starts a
	// goroutine that fails after Run returns (e.g., the service
	// shuts down), it should call fail().
	Run(ctx context.Context, fail func(error), super *Supervisor) error
	String() string
}

var errNeedConfigReload = errors.New("config changed, restart needed")
var errParseFlags = errors.New("error parsing command line arguments")

type bootCommand struct{}

func (bcmd bootCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "json", "info")
	ctx := ctxlog.Context(context.Background(), logger)
	for {
		err := bcmd.run(ctx, prog, args, stdin, stdout, stderr)
		if err == errNeedConfigReload {
			continue
		} else if err == errParseFlags {
			return 2
		} else if err != nil {
			logger.WithError(err).Info("exiting")
			return 1
		} else {
			return 0
		}
	}
}

func (bcmd bootCommand) run(ctx context.Context, prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	super := &Supervisor{
		Stdin:  stdin,
		Stderr: stderr,
		logger: ctxlog.FromContext(ctx),
	}

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	flags.StringVar(&super.ConfigPath, "config", "/etc/arvados/config.yml", "arvados config file `path`")
	flags.StringVar(&super.SourcePath, "source", ".", "arvados source tree `directory`")
	flags.StringVar(&super.ClusterType, "type", "production", "cluster `type`: development, test, or production")
	flags.StringVar(&super.ListenHost, "listen-host", "localhost", "host name or interface address for internal services whose InternalURLs are not configured")
	flags.StringVar(&super.ControllerAddr, "controller-address", ":0", "desired controller address, `host:port` or `:port`")
	flags.StringVar(&super.Workbench2Source, "workbench2-source", "../arvados-workbench2", "path to arvados-workbench2 source tree")
	flags.BoolVar(&super.NoWorkbench1, "no-workbench1", false, "do not run workbench1")
	flags.BoolVar(&super.NoWorkbench2, "no-workbench2", true, "do not run workbench2")
	flags.BoolVar(&super.OwnTemporaryDatabase, "own-temporary-database", false, "bring up a postgres server and create a temporary database")
	timeout := flags.Duration("timeout", 0, "maximum time to wait for cluster to be ready")
	shutdown := flags.Bool("shutdown", false, "shut down when the cluster becomes ready")
	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		if code == 0 {
			return nil
		} else {
			return errParseFlags
		}
	} else if *versionFlag {
		cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
		return nil
	} else if super.ClusterType != "development" && super.ClusterType != "test" && super.ClusterType != "production" {
		return fmt.Errorf("cluster type must be 'development', 'test', or 'production'")
	}

	super.Start(ctx)
	defer super.Stop()

	var timer *time.Timer
	if *timeout > 0 {
		timer = time.AfterFunc(*timeout, super.Stop)
	}

	ok := super.WaitReady()
	if timer != nil && !timer.Stop() {
		return errors.New("boot timed out")
	} else if !ok {
		super.logger.Error("boot failed")
	} else {
		// Write each cluster's controller URL, id, and URL
		// host:port to stdout.  Nothing else goes to stdout,
		// so this allows a calling script to determine when
		// the cluster is ready to use, and the controller's
		// host:port (which may have been dynamically assigned
		// depending on config/options).
		//
		// Sort output by cluster ID for convenience.
		var ids []string
		for id := range super.Clusters() {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			cc := super.Cluster(id)
			// Providing both scheme://host:port and
			// host:port is redundant, but convenient.
			fmt.Fprintln(stdout, cc.Services.Controller.ExternalURL, id, cc.Services.Controller.ExternalURL.Host)
		}
		// Write ".\n" to mark the end of the list of
		// controllers, in case the caller doesn't already
		// know how many clusters are coming up.
		fmt.Fprintln(stdout, ".")
		if *shutdown {
			super.Stop()
			// Wait for children to exit. Don't report the
			// ensuing "context cancelled" error, though:
			// return nil to indicate successful startup.
			_ = super.Wait()
			fmt.Fprintln(stderr, "PASS - all services booted successfully")
			return nil
		}
	}
	// Wait for signal/crash + orderly shutdown
	return super.Wait()
}
