// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package service provides a cmd.Handler that brings up a system service.
package service

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"strings"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/coreos/go-systemd/daemon"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Handler interface {
	http.Handler
	CheckHealth() error
	// Done returns a channel that closes when the handler shuts
	// itself down, or nil if this never happens.
	Done() <-chan struct{}
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
			log.WithError(err).Error("exiting")
		}
	}()

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)

	loader := config.NewLoader(stdin, log)
	loader.SetupFlags(flags)

	// prog is [keepstore, keep-web, git-httpd, ...]  but the
	// legacy config flags are [-legacy-keepstore-config,
	// -legacy-keepweb-config, -legacy-git-httpd-config, ...]
	legacyFlag := "-legacy-" + strings.Replace(prog, "keep-", "keep", 1) + "-config"
	args = loader.MungeLegacyConfigArgs(log, args, legacyFlag)

	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	pprofAddr := flags.String("pprof", "", "Serve Go profile data at `[addr]:port`")
	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return code
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	}

	if *pprofAddr != "" {
		go func() {
			log.Println(http.ListenAndServe(*pprofAddr, nil))
		}()
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
		"PID":       os.Getpid(),
		"ClusterID": cluster.ClusterID,
	})
	ctx := ctxlog.Context(c.ctx, logger)

	listenURL, err := getListenAddr(cluster.Services, c.svcName, log)
	if err != nil {
		return 1
	}
	ctx = context.WithValue(ctx, contextKeyURL{}, listenURL)

	reg := prometheus.NewRegistry()
	loader.RegisterMetrics(reg)

	// arvados_version_running{version="1.2.3~4"} 1.0
	mVersion := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Name:      "version_running",
		Help:      "Indicated version is running.",
	}, []string{"version"})
	mVersion.WithLabelValues(cmd.Version.String()).Set(1)
	reg.MustRegister(mVersion)

	handler := c.newHandler(ctx, cluster, cluster.SystemRootToken, reg)
	if err = handler.CheckHealth(); err != nil {
		return 1
	}

	instrumented := httpserver.Instrument(reg, log,
		httpserver.HandlerWithDeadline(cluster.API.RequestTimeout.Duration(),
			httpserver.AddRequestIDs(
				httpserver.Inspect(reg, cluster.ManagementToken,
					httpserver.LogRequests(
						interceptHealthReqs(cluster.ManagementToken, handler.CheckHealth,
							httpserver.NewRequestLimiter(cluster.API.MaxConcurrentRequests, handler, reg)))))))
	srv := &httpserver.Server{
		Server: http.Server{
			Handler:     ifCollectionInHost(instrumented, instrumented.ServeAPI(cluster.ManagementToken, instrumented)),
			BaseContext: func(net.Listener) context.Context { return ctx },
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
		"Version": cmd.Version.String(),
	}).Info("listening")
	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		logger.WithError(err).Errorf("error notifying init daemon")
	}
	go func() {
		// Shut down server if caller cancels context
		<-ctx.Done()
		srv.Close()
	}()
	go func() {
		// Shut down server if handler dies
		<-handler.Done()
		srv.Close()
	}()
	err = srv.Wait()
	if err != nil {
		return 1
	}
	return 0
}

// If an incoming request's target vhost has an embedded collection
// UUID or PDH, handle it with hTrue, otherwise handle it with
// hFalse.
//
// Facilitates routing "http://collections.example/metrics" to metrics
// and "http://{uuid}.collections.example/metrics" to a file in a
// collection.
func ifCollectionInHost(hTrue, hFalse http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if arvados.CollectionIDFromDNSName(r.Host) != "" {
			hTrue.ServeHTTP(w, r)
		} else {
			hFalse.ServeHTTP(w, r)
		}
	})
}

func interceptHealthReqs(mgtToken string, checkHealth func() error, next http.Handler) http.Handler {
	mux := httprouter.New()
	mux.Handler("GET", "/_health/ping", &health.Handler{
		Token:  mgtToken,
		Prefix: "/_health/",
		Routes: health.Routes{"ping": checkHealth},
	})
	mux.NotFound = next
	return ifCollectionInHost(next, mux)
}

func getListenAddr(svcs arvados.Services, prog arvados.ServiceName, log logrus.FieldLogger) (arvados.URL, error) {
	svc, ok := svcs.Map()[prog]
	if !ok {
		return arvados.URL{}, fmt.Errorf("unknown service name %q", prog)
	}

	if want := os.Getenv("ARVADOS_SERVICE_INTERNAL_URL"); want == "" {
	} else if url, err := url.Parse(want); err != nil {
		return arvados.URL{}, fmt.Errorf("$ARVADOS_SERVICE_INTERNAL_URL (%q): %s", want, err)
	} else {
		if url.Path == "" {
			url.Path = "/"
		}
		return arvados.URL(*url), nil
	}

	errors := []string{}
	for url := range svc.InternalURLs {
		listener, err := net.Listen("tcp", url.Host)
		if err == nil {
			listener.Close()
			return url, nil
		} else if strings.Contains(err.Error(), "cannot assign requested address") {
			// If 'Host' specifies a different server than
			// the current one, it'll resolve the hostname
			// to IP address, and then fail because it
			// can't bind an IP address it doesn't own.
			continue
		} else {
			errors = append(errors, fmt.Sprintf("tried %v, got %v", url, err))
		}
	}
	if len(errors) > 0 {
		return arvados.URL{}, fmt.Errorf("could not enable the %q service on this host: %s", prog, strings.Join(errors, "; "))
	}
	return arvados.URL{}, fmt.Errorf("configuration does not enable the %q service on this host", prog)
}

type contextKeyURL struct{}

func URLFromContext(ctx context.Context) (arvados.URL, bool) {
	u, ok := ctx.Value(contextKeyURL{}).(arvados.URL)
	return u, ok
}
