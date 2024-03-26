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
	"net/http/httptest"
	_ "net/http/pprof"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

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

var requestQueueDumpCheckInterval = time.Minute

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

	listenURL, internalURL, err := getListenAddr(cluster.Services, c.svcName, log)
	if err != nil {
		return 1
	}
	ctx = context.WithValue(ctx, contextKeyURL{}, internalURL)

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
							c.requestLimiter(handler, cluster, reg)))))))
	srv := &httpserver.Server{
		Server: http.Server{
			Handler:     ifCollectionInHost(instrumented, instrumented.ServeAPI(cluster.ManagementToken, instrumented)),
			BaseContext: func(net.Listener) context.Context { return ctx },
		},
		Addr: listenURL.Host,
	}
	if listenURL.Scheme == "https" || listenURL.Scheme == "wss" {
		tlsconfig, err := makeTLSConfig(cluster, logger)
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
	go c.requestQueueDumpCheck(cluster, prog, reg, &srv.Server, logger)
	err = srv.Wait()
	if err != nil {
		return 1
	}
	return 0
}

// If SystemLogs.RequestQueueDumpDirectory is set, monitor the
// server's incoming HTTP request limiters. When the number of
// concurrent requests in any queue ("api" or "tunnel") exceeds 90% of
// its maximum slots, write the /_inspect/requests data to a JSON file
// in the specified directory.
func (c *command) requestQueueDumpCheck(cluster *arvados.Cluster, prog string, reg *prometheus.Registry, srv *http.Server, logger logrus.FieldLogger) {
	outdir := cluster.SystemLogs.RequestQueueDumpDirectory
	if outdir == "" || cluster.ManagementToken == "" {
		return
	}
	logger = logger.WithField("worker", "RequestQueueDump")
	outfile := outdir + "/" + prog + "-requests.json"
	for range time.NewTicker(requestQueueDumpCheckInterval).C {
		mfs, err := reg.Gather()
		if err != nil {
			logger.WithError(err).Warn("error getting metrics")
			continue
		}
		cur := map[string]int{} // queue label => current
		max := map[string]int{} // queue label => max
		for _, mf := range mfs {
			for _, m := range mf.GetMetric() {
				for _, ml := range m.GetLabel() {
					if ml.GetName() == "queue" {
						n := int(m.GetGauge().GetValue())
						if name := mf.GetName(); name == "arvados_concurrent_requests" {
							cur[*ml.Value] = n
						} else if name == "arvados_max_concurrent_requests" {
							max[*ml.Value] = n
						}
					}
				}
			}
		}
		dump := false
		for queue, n := range cur {
			if n > 0 && max[queue] > 0 && n >= max[queue]*9/10 {
				dump = true
				break
			}
		}
		if dump {
			req, err := http.NewRequest("GET", "/_inspect/requests", nil)
			if err != nil {
				logger.WithError(err).Warn("error in http.NewRequest")
				continue
			}
			req.Header.Set("Authorization", "Bearer "+cluster.ManagementToken)
			resp := httptest.NewRecorder()
			srv.Handler.ServeHTTP(resp, req)
			if code := resp.Result().StatusCode; code != http.StatusOK {
				logger.WithField("StatusCode", code).Warn("error getting /_inspect/requests")
				continue
			}
			err = os.WriteFile(outfile, resp.Body.Bytes(), 0777)
			if err != nil {
				logger.WithError(err).Warn("error writing file")
				continue
			}
		}
	}
}

// Set up a httpserver.RequestLimiter with separate queues/streams for
// API requests (obeying MaxConcurrentRequests etc) and gateway tunnel
// requests (obeying MaxGatewayTunnels).
func (c *command) requestLimiter(handler http.Handler, cluster *arvados.Cluster, reg *prometheus.Registry) http.Handler {
	maxReqs := cluster.API.MaxConcurrentRequests
	if maxRails := cluster.API.MaxConcurrentRailsRequests; maxRails > 0 &&
		(maxRails < maxReqs || maxReqs == 0) &&
		c.svcName == arvados.ServiceNameController {
		// Ideally, we would accept up to
		// MaxConcurrentRequests, and apply the
		// MaxConcurrentRailsRequests limit only for requests
		// that require calling upstream to RailsAPI. But for
		// now we make the simplifying assumption that every
		// controller request causes an upstream RailsAPI
		// request.
		maxReqs = maxRails
	}
	rqAPI := &httpserver.RequestQueue{
		Label:                      "api",
		MaxConcurrent:              maxReqs,
		MaxQueue:                   cluster.API.MaxQueuedRequests,
		MaxQueueTimeForMinPriority: cluster.API.MaxQueueTimeForLockRequests.Duration(),
	}
	rqTunnel := &httpserver.RequestQueue{
		Label:         "tunnel",
		MaxConcurrent: cluster.API.MaxGatewayTunnels,
		MaxQueue:      0,
	}
	return &httpserver.RequestLimiter{
		Handler:  handler,
		Priority: c.requestPriority,
		Registry: reg,
		Queue: func(req *http.Request) *httpserver.RequestQueue {
			if req.Method == http.MethodPost && reTunnelPath.MatchString(req.URL.Path) {
				return rqTunnel
			} else {
				return rqAPI
			}
		},
	}
}

// reTunnelPath matches paths of API endpoints that go in the "tunnel"
// queue.
var reTunnelPath = regexp.MustCompile(func() string {
	rePathVar := regexp.MustCompile(`{.*?}`)
	out := ""
	for _, endpoint := range []arvados.APIEndpoint{
		arvados.EndpointContainerGatewayTunnel,
		arvados.EndpointContainerGatewayTunnelCompat,
		arvados.EndpointContainerSSH,
		arvados.EndpointContainerSSHCompat,
	} {
		if out != "" {
			out += "|"
		}
		out += `\Q/` + rePathVar.ReplaceAllString(endpoint.Path, `\E[^/]*\Q`) + `\E`
	}
	return "^(" + out + ")$"
}())

func (c *command) requestPriority(req *http.Request, queued time.Time) int64 {
	switch {
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/arvados/v1/containers/") && strings.HasSuffix(req.URL.Path, "/lock"):
		// Return 503 immediately instead of queueing. We want
		// to send feedback to dispatchcloud ASAP to stop
		// bringing up new containers.
		return httpserver.MinPriority
	case req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/arvados/v1/logs"):
		// "Create log entry" is the most harmless kind of
		// request to drop. Negative priority is called "low"
		// in aggregate metrics.
		return -1
	case req.Header.Get("Origin") != "":
		// Handle interactive requests first. Positive
		// priority is called "high" in aggregate metrics.
		return 1
	default:
		// Zero priority is called "normal" in aggregate
		// metrics.
		return 0
	}
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

// Determine listenURL (addr:port where server should bind) and
// internalURL (target url that client should connect to) for a
// service.
//
// If the config does not specify ListenURL, we check all of the
// configured InternalURLs. If there is exactly one that matches our
// hostname, or exactly one that matches a local interface address,
// then we use that as listenURL.
//
// Note that listenURL and internalURL may use different protocols
// (e.g., listenURL is http, but the service sits behind a proxy, so
// clients connect using https).
func getListenAddr(svcs arvados.Services, prog arvados.ServiceName, log logrus.FieldLogger) (arvados.URL, arvados.URL, error) {
	svc, ok := svcs.Map()[prog]
	if !ok {
		return arvados.URL{}, arvados.URL{}, fmt.Errorf("unknown service name %q", prog)
	}

	if want := os.Getenv("ARVADOS_SERVICE_INTERNAL_URL"); want != "" {
		url, err := url.Parse(want)
		if err != nil {
			return arvados.URL{}, arvados.URL{}, fmt.Errorf("$ARVADOS_SERVICE_INTERNAL_URL (%q): %s", want, err)
		}
		if url.Path == "" {
			url.Path = "/"
		}
		for internalURL, conf := range svc.InternalURLs {
			if internalURL.String() == url.String() {
				listenURL := conf.ListenURL
				if listenURL.Host == "" {
					listenURL = internalURL
				}
				return listenURL, internalURL, nil
			}
		}
		log.Warnf("possible configuration error: listening on %s (from $ARVADOS_SERVICE_INTERNAL_URL) even though configuration does not have a matching InternalURLs entry", url)
		internalURL := arvados.URL(*url)
		return internalURL, internalURL, nil
	}

	errors := []string{}
	for internalURL, conf := range svc.InternalURLs {
		listenURL := conf.ListenURL
		if listenURL.Host == "" {
			// If ListenURL is not specified, assume
			// InternalURL is also usable as the listening
			// proto/addr/port (i.e., simple case with no
			// intermediate proxy/routing)
			listenURL = internalURL
		}
		listenAddr := listenURL.Host
		if _, _, err := net.SplitHostPort(listenAddr); err != nil {
			// url "https://foo.example/" (with no
			// explicit port name/number) means listen on
			// the well-known port for the specified
			// protocol, "foo.example:https".
			port := listenURL.Scheme
			if port == "ws" || port == "wss" {
				port = "http" + port[2:]
			}
			listenAddr = net.JoinHostPort(listenAddr, port)
		}
		listener, err := net.Listen("tcp", listenAddr)
		if err == nil {
			listener.Close()
			return listenURL, internalURL, nil
		} else if strings.Contains(err.Error(), "cannot assign requested address") {
			// If 'Host' specifies a different server than
			// the current one, it'll resolve the hostname
			// to IP address, and then fail because it
			// can't bind an IP address it doesn't own.
			continue
		} else {
			errors = append(errors, fmt.Sprintf("%s: %s", listenURL, err))
		}
	}
	if len(errors) > 0 {
		return arvados.URL{}, arvados.URL{}, fmt.Errorf("could not enable the %q service on this host: %s", prog, strings.Join(errors, "; "))
	}
	return arvados.URL{}, arvados.URL{}, fmt.Errorf("configuration does not enable the %q service on this host", prog)
}

type contextKeyURL struct{}

func URLFromContext(ctx context.Context) (arvados.URL, bool) {
	u, ok := ctx.Value(contextKeyURL{}).(arvados.URL)
	return u, ok
}
