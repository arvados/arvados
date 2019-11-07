// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"git.curoverse.com/arvados.git/lib/config"
	"git.curoverse.com/arvados.git/lib/service"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	version = "dev"
	Command = service.Command(arvados.ServiceNameKeepstore, newHandlerOrErrorHandler)
)

func main() {
	os.Exit(runCommand(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func runCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	args, ok := convertKeepstoreFlagsToServiceFlags(args, ctxlog.FromContext(context.Background()))
	if !ok {
		return 2
	}
	return Command.RunCommand(prog, args, stdin, stdout, stderr)
}

// Parse keepstore command line flags, and return equivalent
// service.Command flags. The second return value ("ok") is true if
// all provided flags were successfully converted.
func convertKeepstoreFlagsToServiceFlags(args []string, lgr logrus.FieldLogger) ([]string, bool) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.String("listen", "", "Services.Keepstore.InternalURLs")
	flags.Int("max-buffers", 0, "API.MaxKeepBlobBuffers")
	flags.Int("max-requests", 0, "API.MaxConcurrentRequests")
	flags.Bool("never-delete", false, "Collections.BlobTrash")
	flags.Bool("enforce-permissions", false, "Collections.BlobSigning")
	flags.String("permission-key-file", "", "Collections.BlobSigningKey")
	flags.String("blob-signing-key-file", "", "Collections.BlobSigningKey")
	flags.String("data-manager-token-file", "", "SystemRootToken")
	flags.Int("permission-ttl", 0, "Collections.BlobSigningTTL")
	flags.Int("blob-signature-ttl", 0, "Collections.BlobSigningTTL")
	flags.String("trash-lifetime", "", "Collections.BlobTrashLifetime")
	flags.Bool("serialize", false, "Volumes.*.DriverParameters.Serialize")
	flags.Bool("readonly", false, "Volumes.*.ReadOnly")
	flags.String("pid", "", "-")
	flags.String("trash-check-interval", "", "Collections.BlobTrashCheckInterval")

	flags.String("azure-storage-container-volume", "", "Volumes.*.Driver")
	flags.String("azure-storage-account-name", "", "Volumes.*.DriverParameters.StorageAccountName")
	flags.String("azure-storage-account-key-file", "", "Volumes.*.DriverParameters.StorageAccountKey")
	flags.String("azure-storage-replication", "", "Volumes.*.Replication")
	flags.String("azure-max-get-bytes", "", "Volumes.*.DriverParameters.MaxDataReadSize")

	flags.String("s3-bucket-volume", "", "Volumes.*.DriverParameters.Bucket")
	flags.String("s3-region", "", "Volumes.*.DriverParameters.Region")
	flags.String("s3-endpoint", "", "Volumes.*.DriverParameters.Endpoint")
	flags.String("s3-access-key-file", "", "Volumes.*.DriverParameters.AccessKey")
	flags.String("s3-secret-key-file", "", "Volumes.*.DriverParameters.SecretKey")
	flags.String("s3-race-window", "", "Volumes.*.DriverParameters.RaceWindow")
	flags.String("s3-replication", "", "Volumes.*.Replication")
	flags.String("s3-unsafe-delete", "", "Volumes.*.DriverParameters.UnsafeDelete")

	flags.String("volume", "", "Volumes")

	flags.Bool("version", false, "")
	flags.String("config", "", "")
	flags.String("legacy-keepstore-config", "", "")

	err := flags.Parse(args)
	if err == flag.ErrHelp {
		return []string{"-help"}, true
	} else if err != nil {
		return nil, false
	}

	args = nil
	ok := true
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "config" || f.Name == "legacy-keepstore-config" || f.Name == "version" {
			args = append(args, "-"+f.Name, f.Value.String())
		} else if f.Usage == "-" {
			ok = false
			lgr.Errorf("command line flag -%s is no longer supported", f.Name)
		} else {
			ok = false
			lgr.Errorf("command line flag -%s is no longer supported -- use Clusters.*.%s in cluster config file instead", f.Name, f.Usage)
		}
	})
	if !ok {
		return nil, false
	}

	flags = flag.NewFlagSet("", flag.ExitOnError)
	loader := config.NewLoader(nil, lgr)
	loader.SetupFlags(flags)
	return loader.MungeLegacyConfigArgs(lgr, args, "-legacy-keepstore-config"), true
}

type handler struct {
	http.Handler
	Cluster *arvados.Cluster
	Logger  logrus.FieldLogger

	pullq      *WorkQueue
	trashq     *WorkQueue
	volmgr     *RRVolumeManager
	keepClient *keepclient.KeepClient

	err       error
	setupOnce sync.Once
}

func (h *handler) CheckHealth() error {
	return h.err
}

func newHandlerOrErrorHandler(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry) service.Handler {
	var h handler
	serviceURL, ok := service.URLFromContext(ctx)
	if !ok {
		return service.ErrorHandler(ctx, cluster, errors.New("BUG: no URL from service.URLFromContext"))
	}
	err := h.setup(ctx, cluster, token, reg, serviceURL)
	if err != nil {
		return service.ErrorHandler(ctx, cluster, err)
	}
	return &h
}

func (h *handler) setup(ctx context.Context, cluster *arvados.Cluster, token string, reg *prometheus.Registry, serviceURL arvados.URL) error {
	h.Cluster = cluster
	h.Logger = ctxlog.FromContext(ctx)
	if h.Cluster.API.MaxKeepBlobBuffers <= 0 {
		return fmt.Errorf("API.MaxKeepBlobBuffers must be greater than zero")
	}
	bufs = newBufferPool(h.Logger, h.Cluster.API.MaxKeepBlobBuffers, BlockSize)

	if h.Cluster.API.MaxConcurrentRequests < 1 {
		h.Cluster.API.MaxConcurrentRequests = h.Cluster.API.MaxKeepBlobBuffers * 2
		h.Logger.Warnf("API.MaxConcurrentRequests <1 or not specified; defaulting to MaxKeepBlobBuffers * 2 == %d", h.Cluster.API.MaxConcurrentRequests)
	}

	if h.Cluster.Collections.BlobSigningKey != "" {
	} else if h.Cluster.Collections.BlobSigning {
		return errors.New("cannot enable Collections.BlobSigning with no Collections.BlobSigningKey")
	} else {
		h.Logger.Warn("Running without a blob signing key. Block locators returned by this server will not be signed, and will be rejected by a server that enforces permissions. To fix this, configure Collections.BlobSigning and Collections.BlobSigningKey.")
	}

	if len(h.Cluster.Volumes) == 0 {
		return errors.New("no volumes configured")
	}

	h.Logger.Printf("keepstore %s starting, pid %d", version, os.Getpid())

	// Start a round-robin VolumeManager with the configured volumes.
	vm, err := makeRRVolumeManager(h.Logger, h.Cluster, serviceURL, newVolumeMetricsVecs(reg))
	if err != nil {
		return err
	}
	if len(vm.readables) == 0 {
		return fmt.Errorf("no volumes configured for %s", serviceURL)
	}
	h.volmgr = vm

	// Initialize the pullq and workers
	h.pullq = NewWorkQueue()
	for i := 0; i < 1 || i < h.Cluster.Collections.BlobReplicateConcurrency; i++ {
		go h.runPullWorker(h.pullq)
	}

	// Initialize the trashq and workers
	h.trashq = NewWorkQueue()
	for i := 0; i < 1 || i < h.Cluster.Collections.BlobTrashConcurrency; i++ {
		go RunTrashWorker(h.volmgr, h.Logger, h.Cluster, h.trashq)
	}

	// Set up routes and metrics
	h.Handler = MakeRESTRouter(ctx, cluster, reg, vm, h.pullq, h.trashq)

	// Initialize keepclient for pull workers
	c, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return err
	}
	ac, err := arvadosclient.New(c)
	if err != nil {
		return err
	}
	h.keepClient = &keepclient.KeepClient{
		Arvados:       ac,
		Want_replicas: 1,
	}
	h.keepClient.Arvados.ApiToken = fmt.Sprintf("%x", rand.Int63())

	if d := h.Cluster.Collections.BlobTrashCheckInterval.Duration(); d > 0 {
		go emptyTrash(h.volmgr.writables, d)
	}

	return nil
}
