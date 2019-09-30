// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package service provides a cmd.Handler that brings up a system service.
package service

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/lib/config"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"github.com/coreos/go-systemd/daemon"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	http.Handler
	CheckHealth() error
}

type NewHandlerFunc func(_ context.Context, _ *arvados.Cluster, token string, registry *prometheus.Registry) Handler

type command struct {
	newHandler NewHandlerFunc
	svcName    arvados.ServiceName
	ctx        context.Context // enables tests to shutdown service; no public API yet
}

// Command returns a cmd.Handler that loads site config, calls
// newHandler with the current cluster and node configs, and brings up
// an http server with the returned handler.
//
// The handler is wrapped with server middleware (adding X-Request-ID
// headers, logging requests/responses, etc).
func Command(svcName arvados.ServiceName, newHandler NewHandlerFunc) cmd.Handler {
	return &command{
		newHandler: newHandler,
		svcName:    svcName,
		ctx:        context.Background(),
	}
}

func (c *command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	log := ctxlog.New(stderr, "json", "info")

	var err error
	defer func() {
		if err != nil {
			log.WithError(err).Info("exiting")
		}
	}()

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)

	loader := config.NewLoader(stdin, log)
	loader.SetupFlags(flags)
	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	}

	if strings.HasSuffix(prog, "controller") {
		// Some config-loader checks try to make API calls via
		// controller. Those can't be expected to work if this
		// process _is_ the controller: we haven't started an
		// http server yet.
		loader.SkipAPICalls = true
	}

	cfg, err := loader.Load()
	if err != nil {
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}

	// Now that we've read the config, replace the bootstrap
	// logger with a new one according to the logging config.
	log = ctxlog.New(stderr, cluster.SystemLogs.Format, cluster.SystemLogs.LogLevel)
	logger := log.WithFields(logrus.Fields{
		"PID": os.Getpid(),
	})
	ctx := ctxlog.Context(c.ctx, logger)

	listenURL, err := getListenAddr(cluster.Services, c.svcName)
	if err != nil {
		return 1
	}
	ctx = context.WithValue(ctx, contextKeyURL{}, listenURL)

	reg := prometheus.NewRegistry()
	handler := c.newHandler(ctx, cluster, cluster.SystemRootToken, reg)
	if err = handler.CheckHealth(); err != nil {
		return 1
	}

	instrumented := httpserver.Instrument(reg, log,
		httpserver.HandlerWithContext(ctx,
			httpserver.AddRequestIDs(
				httpserver.LogRequests(
					httpserver.NewRequestLimiter(cluster.API.MaxConcurrentRequests, handler, reg)))))
	srv := &httpserver.Server{
		Server: http.Server{
			Handler: instrumented.ServeAPI(cluster.ManagementToken, instrumented),
		},
		Addr: listenURL.Host,
	}
	if listenURL.Scheme == "https" {
		tlsconfig, err := tlsConfigWithCertUpdater(cluster, logger)
		if err != nil {
			logger.WithError(err).Errorf("cannot start %s service on %s", c.svcName, listenURL.String())
			return 1
		}
		srv.TLSConfig = tlsconfig
	}
	err = srv.Start()
	if err != nil {
		return 1
	}
	logger.WithFields(logrus.Fields{
		"URL":     listenURL,
		"Listen":  srv.Addr,
		"Service": c.svcName,
	}).Info("listening")
	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		logger.WithError(err).Errorf("error notifying init daemon")
	}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	err = srv.Wait()
	if err != nil {
		return 1
	}
	return 0
}

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

func getListenAddr(svcs arvados.Services, prog arvados.ServiceName) (arvados.URL, error) {
	svc, ok := svcs.Map()[prog]
	if !ok {
		return arvados.URL{}, fmt.Errorf("unknown service name %q", prog)
	}
	for url := range svc.InternalURLs {
		if strings.HasPrefix(url.Host, "localhost:") {
			return url, nil
		}
		listener, err := net.Listen("tcp", url.Host)
		if err == nil {
			listener.Close()
			return url, nil
		}
	}
	return arvados.URL{}, fmt.Errorf("configuration does not enable the %s service on this host", prog)
}

type contextKeyURL struct{}

func URLFromContext(ctx context.Context) (arvados.URL, bool) {
	u, ok := ctx.Value(contextKeyURL{}).(arvados.URL)
	return u, ok
}
