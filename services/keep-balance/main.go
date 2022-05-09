// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	_ "net/http/pprof"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

type command struct{}

var Command = command{}

func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var options RunOptions
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	flags.BoolVar(&options.Once, "once", false,
		"balance once and then exit")
	flags.BoolVar(&options.CommitPulls, "commit-pulls", false,
		"send pull requests (make more replicas of blocks that are underreplicated or are not in optimal rendezvous probe order)")
	flags.BoolVar(&options.CommitTrash, "commit-trash", false,
		"send trash requests (delete unreferenced old blocks, and excess replicas of overreplicated blocks)")
	flags.BoolVar(&options.CommitConfirmedFields, "commit-confirmed-fields", true,
		"update collection fields (replicas_confirmed, storage_classes_confirmed, etc.)")
	// These options are implemented by service.Command, so we
	// don't need the vars here -- we just need the flags
	// to pass flags.Parse().
	flags.Bool("dump", false, "dump details for each block to stdout")
	flags.String("pprof", "", "serve Go profile data at `[addr]:port`")
	flags.Bool("version", false, "Write version information to stdout and exit 0")

	logger := ctxlog.New(stderr, "json", "info")
	loader := config.NewLoader(&bytes.Buffer{}, logger)
	loader.SetupFlags(flags)
	munged := loader.MungeLegacyConfigArgs(logger, args, "-legacy-keepbalance-config")
	if ok, code := cmd.ParseFlags(flags, prog, munged, "", stderr); !ok {
		return code
	}

	// Drop our custom args that would be rejected by the generic
	// service.Command
	args = nil
	dropFlag := map[string]bool{
		"once":                    true,
		"commit-pulls":            true,
		"commit-trash":            true,
		"commit-confirmed-fields": true,
		"dump":                    true,
	}
	flags.Visit(func(f *flag.Flag) {
		if !dropFlag[f.Name] {
			args = append(args, "-"+f.Name+"="+f.Value.String())
		}
	})

	return service.Command(arvados.ServiceNameKeepbalance,
		func(ctx context.Context, cluster *arvados.Cluster, token string, registry *prometheus.Registry) service.Handler {
			if !options.Once && cluster.Collections.BalancePeriod == arvados.Duration(0) {
				return service.ErrorHandler(ctx, cluster, fmt.Errorf("cannot start service: Collections.BalancePeriod is zero (if you want to run once and then exit, use the -once flag)"))
			}

			ac, err := arvados.NewClientFromConfig(cluster)
			ac.AuthToken = token
			if err != nil {
				return service.ErrorHandler(ctx, cluster, fmt.Errorf("error initializing client from cluster config: %s", err))
			}

			db, err := sqlx.Open("postgres", cluster.PostgreSQL.Connection.String())
			if err != nil {
				return service.ErrorHandler(ctx, cluster, fmt.Errorf("postgresql connection failed: %s", err))
			}
			if p := cluster.PostgreSQL.ConnectionPool; p > 0 {
				db.SetMaxOpenConns(p)
			}
			err = db.Ping()
			if err != nil {
				return service.ErrorHandler(ctx, cluster, fmt.Errorf("postgresql connection succeeded but ping failed: %s", err))
			}

			if options.Logger == nil {
				options.Logger = ctxlog.FromContext(ctx)
			}

			srv := &Server{
				Cluster:    cluster,
				ArvClient:  ac,
				RunOptions: options,
				Metrics:    newMetrics(registry),
				Logger:     options.Logger,
				Dumper:     options.Dumper,
				DB:         db,
			}
			srv.Handler = &health.Handler{
				Token:  cluster.ManagementToken,
				Prefix: "/_health/",
				Routes: health.Routes{"ping": srv.CheckHealth},
			}

			go srv.run()
			return srv
		}).RunCommand(prog, args, stdin, stdout, stderr)
}
