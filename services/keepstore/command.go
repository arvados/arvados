// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

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

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	Command = service.Command(arvados.ServiceNameKeepstore, newHandlerOrErrorHandler)
)

func runCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	args, ok, code := convertKeepstoreFlagsToServiceFlags(prog, args, ctxlog.FromContext(context.Background()), stderr)
	if !ok {
		return code
	}
	return Command.RunCommand(prog, args, stdin, stdout, stderr)
}

// Parse keepstore command line flags, and return equivalent
// service.Command flags. If the second return value ("ok") is false,
// the program should exit, and the third return value is a suitable
// exit code.
func convertKeepstoreFlagsToServiceFlags(prog string, args []string, lgr logrus.FieldLogger, stderr io.Writer) ([]string, bool, int) {
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
	flags.String("s3-access-key-file", "", "Volumes.*.DriverParameters.AccessKeyID")
	flags.String("s3-secret-key-file", "", "Volumes.*.DriverParameters.SecretAccessKey")
	flags.String("s3-race-window", "", "Volumes.*.DriverParameters.RaceWindow")
	flags.String("s3-replication", "", "Volumes.*.Replication")
	flags.String("s3-unsafe-delete", "", "Volumes.*.DriverParameters.UnsafeDelete")

	flags.String("volume", "", "Volumes")

	flags.Bool("version", false, "")
	flags.String("config", "", "")
	flags.String("legacy-keepstore-config", "", "")

	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return nil, false, code
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
		return nil, false, 2
	}

	flags = flag.NewFlagSet("", flag.ContinueOnError)
	loader := config.NewLoader(nil, lgr)
	loader.SetupFlags(flags)
	return loader.MungeLegacyConfigArgs(lgr, args, "-legacy-keepstore-config"), true, 0
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

func (h *handler) Done() <-chan struct{} {
	return nil
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

	if h.Cluster.API.MaxConcurrentRequests > 0 && h.Cluster.API.MaxConcurrentRequests < h.Cluster.API.MaxKeepBlobBuffers {
		h.Logger.Warnf("Possible configuration mistake: not useful to set API.MaxKeepBlobBuffers (%d) higher than API.MaxConcurrentRequests (%d)", h.Cluster.API.MaxKeepBlobBuffers, h.Cluster.API.MaxConcurrentRequests)
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

	h.Logger.Printf("keepstore %s starting, pid %d", cmd.Version.String(), os.Getpid())

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
