// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-systemd/daemon"
	"github.com/ghodss/yaml"
	"github.com/gorilla/mux"
)

var version = "dev"

type Config struct {
	Client          arvados.Client
	Listen          string
	DisableGet      bool
	DisablePut      bool
	DefaultReplicas int
	Timeout         arvados.Duration
	PIDFile         string
	Debug           bool
	ManagementToken string
}

func DefaultConfig() *Config {
	return &Config{
		Listen:  ":25107",
		Timeout: arvados.Duration(15 * time.Second),
	}
}

var (
	listener net.Listener
	router   http.Handler
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

func main() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: rfc3339NanoFixed,
	})

	cfg := DefaultConfig()

	flagset := flag.NewFlagSet("keepproxy", flag.ExitOnError)
	flagset.Usage = usage

	const deprecated = " (DEPRECATED -- use config file instead)"
	flagset.StringVar(&cfg.Listen, "listen", cfg.Listen, "Local port to listen on."+deprecated)
	flagset.BoolVar(&cfg.DisableGet, "no-get", cfg.DisableGet, "Disable GET operations."+deprecated)
	flagset.BoolVar(&cfg.DisablePut, "no-put", cfg.DisablePut, "Disable PUT operations."+deprecated)
	flagset.IntVar(&cfg.DefaultReplicas, "default-replicas", cfg.DefaultReplicas, "Default number of replicas to write if not specified by the client. If 0, use site default."+deprecated)
	flagset.StringVar(&cfg.PIDFile, "pid", cfg.PIDFile, "Path to write pid file."+deprecated)
	timeoutSeconds := flagset.Int("timeout", int(time.Duration(cfg.Timeout)/time.Second), "Timeout (in seconds) on requests to internal Keep services."+deprecated)
	flagset.StringVar(&cfg.ManagementToken, "management-token", cfg.ManagementToken, "Authorization token to be included in all health check requests.")

	var cfgPath string
	const defaultCfgPath = "/etc/arvados/keepproxy/keepproxy.yml"
	flagset.StringVar(&cfgPath, "config", defaultCfgPath, "Configuration file `path`")
	dumpConfig := flagset.Bool("dump-config", false, "write current configuration to stdout and exit")
	getVersion := flagset.Bool("version", false, "Print version information and exit.")
	flagset.Parse(os.Args[1:])

	// Print version information if requested
	if *getVersion {
		fmt.Printf("keepproxy %s\n", version)
		return
	}

	err := config.LoadFile(cfg, cfgPath)
	if err != nil {
		h := os.Getenv("ARVADOS_API_HOST")
		t := os.Getenv("ARVADOS_API_TOKEN")
		if h == "" || t == "" || !os.IsNotExist(err) || cfgPath != defaultCfgPath {
			log.Fatal(err)
		}
		log.Print("DEPRECATED: No config file found, but ARVADOS_API_HOST and ARVADOS_API_TOKEN environment variables are set. Please use a config file instead.")
		cfg.Client.APIHost = h
		cfg.Client.AuthToken = t
		if regexp.MustCompile("^(?i:1|yes|true)$").MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE")) {
			cfg.Client.Insecure = true
		}
		if y, err := yaml.Marshal(cfg); err == nil && !*dumpConfig {
			log.Print("Current configuration:\n", string(y))
		}
		cfg.Timeout = arvados.Duration(time.Duration(*timeoutSeconds) * time.Second)
	}

	if *dumpConfig {
		log.Fatal(config.DumpAndExit(cfg))
	}

	log.Printf("keepproxy %s started", version)

	arv, err := arvadosclient.New(&cfg.Client)
	if err != nil {
		log.Fatalf("Error setting up arvados client %s", err.Error())
	}

	if cfg.Debug {
		keepclient.DebugPrintf = log.Printf
	}
	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		log.Fatalf("Error setting up keep client %s", err.Error())
	}
	keepclient.RefreshServiceDiscoveryOnSIGHUP()

	if cfg.PIDFile != "" {
		f, err := os.Create(cfg.PIDFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			log.Fatalf("flock(%s): %s", cfg.PIDFile, err)
		}
		defer os.Remove(cfg.PIDFile)
		err = f.Truncate(0)
		if err != nil {
			log.Fatalf("truncate(%s): %s", cfg.PIDFile, err)
		}
		_, err = fmt.Fprint(f, os.Getpid())
		if err != nil {
			log.Fatalf("write(%s): %s", cfg.PIDFile, err)
		}
		err = f.Sync()
		if err != nil {
			log.Fatal("sync(%s): %s", cfg.PIDFile, err)
		}
	}

	if cfg.DefaultReplicas > 0 {
		kc.Want_replicas = cfg.DefaultReplicas
	}

	listener, err = net.Listen("tcp", cfg.Listen)
	if err != nil {
		log.Fatalf("listen(%s): %s", cfg.Listen, err)
	}
	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}
	log.Println("Listening at", listener.Addr())

	// Shut down the server gracefully (by closing the listener)
	// if SIGTERM is received.
	term := make(chan os.Signal, 1)
	go func(sig <-chan os.Signal) {
		s := <-sig
		log.Println("caught signal:", s)
		listener.Close()
	}(term)
	signal.Notify(term, syscall.SIGTERM)
	signal.Notify(term, syscall.SIGINT)

	// Start serving requests.
	router = MakeRESTRouter(!cfg.DisableGet, !cfg.DisablePut, kc, time.Duration(cfg.Timeout), cfg.ManagementToken)
	http.Serve(listener, httpserver.AddRequestIDs(httpserver.LogRequests(nil, router)))

	log.Println("shutting down")
}

type ApiTokenCache struct {
	tokens     map[string]int64
	lock       sync.Mutex
	expireTime int64
}

// Cache the token and set an expire time.  If we already have an expire time
// on the token, it is not updated.
func (this *ApiTokenCache) RememberToken(token string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	now := time.Now().Unix()
	if this.tokens[token] == 0 {
		this.tokens[token] = now + this.expireTime
	}
}

// Check if the cached token is known and still believed to be valid.
func (this *ApiTokenCache) RecallToken(token string) bool {
	this.lock.Lock()
	defer this.lock.Unlock()

	now := time.Now().Unix()
	if this.tokens[token] == 0 {
		// Unknown token
		return false
	} else if now < this.tokens[token] {
		// Token is known and still valid
		return true
	} else {
		// Token is expired
		this.tokens[token] = 0
		return false
	}
}

func GetRemoteAddress(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		return xff + "," + req.RemoteAddr
	}
	return req.RemoteAddr
}

func CheckAuthorizationHeader(kc *keepclient.KeepClient, cache *ApiTokenCache, req *http.Request) (pass bool, tok string) {
	parts := strings.SplitN(req.Header.Get("Authorization"), " ", 2)
	if len(parts) < 2 || !(parts[0] == "OAuth2" || parts[0] == "Bearer") || len(parts[1]) == 0 {
		return false, ""
	}
	tok = parts[1]

	// Tokens are validated differently depending on what kind of
	// operation is being performed. For example, tokens in
	// collection-sharing links permit GET requests, but not
	// PUT requests.
	var op string
	if req.Method == "GET" || req.Method == "HEAD" {
		op = "read"
	} else {
		op = "write"
	}

	if cache.RecallToken(op + ":" + tok) {
		// Valid in the cache, short circuit
		return true, tok
	}

	var err error
	arv := *kc.Arvados
	arv.ApiToken = tok
	arv.RequestID = req.Header.Get("X-Request-Id")
	if op == "read" {
		err = arv.Call("HEAD", "keep_services", "", "accessible", nil, nil)
	} else {
		err = arv.Call("HEAD", "users", "", "current", nil, nil)
	}
	if err != nil {
		log.Printf("%s: CheckAuthorizationHeader error: %v", GetRemoteAddress(req), err)
		return false, ""
	}

	// Success!  Update cache
	cache.RememberToken(op + ":" + tok)

	return true, tok
}

// We need to make a private copy of the default http transport early
// in initialization, then make copies of our private copy later. It
// won't be safe to copy http.DefaultTransport itself later, because
// its private mutexes might have already been used. (Without this,
// the test suite sometimes panics "concurrent map writes" in
// net/http.(*Transport).removeIdleConnLocked().)
var defaultTransport = *(http.DefaultTransport.(*http.Transport))

type proxyHandler struct {
	http.Handler
	*keepclient.KeepClient
	*ApiTokenCache
	timeout   time.Duration
	transport *http.Transport
}

// MakeRESTRouter returns an http.Handler that passes GET and PUT
// requests to the appropriate handlers.
func MakeRESTRouter(enable_get bool, enable_put bool, kc *keepclient.KeepClient, timeout time.Duration, mgmtToken string) http.Handler {
	rest := mux.NewRouter()

	transport := defaultTransport
	transport.DialContext = (&net.Dialer{
		Timeout:   keepclient.DefaultConnectTimeout,
		KeepAlive: keepclient.DefaultKeepAlive,
		DualStack: true,
	}).DialContext
	transport.TLSClientConfig = arvadosclient.MakeTLSConfig(kc.Arvados.ApiInsecure)
	transport.TLSHandshakeTimeout = keepclient.DefaultTLSHandshakeTimeout

	h := &proxyHandler{
		Handler:    rest,
		KeepClient: kc,
		timeout:    timeout,
		transport:  &transport,
		ApiTokenCache: &ApiTokenCache{
			tokens:     make(map[string]int64),
			expireTime: 300,
		},
	}

	if enable_get {
		rest.HandleFunc(`/{locator:[0-9a-f]{32}\+.*}`, h.Get).Methods("GET", "HEAD")
		rest.HandleFunc(`/{locator:[0-9a-f]{32}}`, h.Get).Methods("GET", "HEAD")

		// List all blocks
		rest.HandleFunc(`/index`, h.Index).Methods("GET")

		// List blocks whose hash has the given prefix
		rest.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, h.Index).Methods("GET")
	}

	if enable_put {
		rest.HandleFunc(`/{locator:[0-9a-f]{32}\+.*}`, h.Put).Methods("PUT")
		rest.HandleFunc(`/{locator:[0-9a-f]{32}}`, h.Put).Methods("PUT")
		rest.HandleFunc(`/`, h.Put).Methods("POST")
		rest.HandleFunc(`/{any}`, h.Options).Methods("OPTIONS")
		rest.HandleFunc(`/`, h.Options).Methods("OPTIONS")
	}

	rest.Handle("/_health/{check}", &health.Handler{
		Token:  mgmtToken,
		Prefix: "/_health/",
	}).Methods("GET")

	rest.NotFoundHandler = InvalidPathHandler{}
	return h
}

var errLoopDetected = errors.New("loop detected")

func (*proxyHandler) checkLoop(resp http.ResponseWriter, req *http.Request) error {
	if via := req.Header.Get("Via"); strings.Index(via, " "+viaAlias) >= 0 {
		log.Printf("proxy loop detected (request has Via: %q): perhaps keepproxy is misidentified by gateway config as an external client, or its keep_services record does not have service_type=proxy?", via)
		http.Error(resp, errLoopDetected.Error(), http.StatusInternalServerError)
		return errLoopDetected
	}
	return nil
}

func SetCorsHeaders(resp http.ResponseWriter) {
	resp.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, OPTIONS")
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Length, Content-Type, X-Keep-Desired-Replicas")
	resp.Header().Set("Access-Control-Max-Age", "86486400")
}

type InvalidPathHandler struct{}

func (InvalidPathHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Printf("%s: %s %s unroutable", GetRemoteAddress(req), req.Method, req.URL.Path)
	http.Error(resp, "Bad request", http.StatusBadRequest)
}

func (h *proxyHandler) Options(resp http.ResponseWriter, req *http.Request) {
	log.Printf("%s: %s %s", GetRemoteAddress(req), req.Method, req.URL.Path)
	SetCorsHeaders(resp)
}

var BadAuthorizationHeader = errors.New("Missing or invalid Authorization header")
var ContentLengthMismatch = errors.New("Actual length != expected content length")
var MethodNotSupported = errors.New("Method not supported")

var removeHint, _ = regexp.Compile("\\+K@[a-z0-9]{5}(\\+|$)")

func (h *proxyHandler) Get(resp http.ResponseWriter, req *http.Request) {
	if err := h.checkLoop(resp, req); err != nil {
		return
	}
	SetCorsHeaders(resp)
	resp.Header().Set("Via", req.Proto+" "+viaAlias)

	locator := mux.Vars(req)["locator"]
	var err error
	var status int
	var expectLength, responseLength int64
	var proxiedURI = "-"

	defer func() {
		log.Println(GetRemoteAddress(req), req.Method, req.URL.Path, status, expectLength, responseLength, proxiedURI, err)
		if status != http.StatusOK {
			http.Error(resp, err.Error(), status)
		}
	}()

	kc := h.makeKeepClient(req)

	var pass bool
	var tok string
	if pass, tok = CheckAuthorizationHeader(kc, h.ApiTokenCache, req); !pass {
		status, err = http.StatusForbidden, BadAuthorizationHeader
		return
	}

	// Copy ArvadosClient struct and use the client's API token
	arvclient := *kc.Arvados
	arvclient.ApiToken = tok
	kc.Arvados = &arvclient

	var reader io.ReadCloser

	locator = removeHint.ReplaceAllString(locator, "$1")

	switch req.Method {
	case "HEAD":
		expectLength, proxiedURI, err = kc.Ask(locator)
	case "GET":
		reader, expectLength, proxiedURI, err = kc.Get(locator)
		if reader != nil {
			defer reader.Close()
		}
	default:
		status, err = http.StatusNotImplemented, MethodNotSupported
		return
	}

	if expectLength == -1 {
		log.Println("Warning:", GetRemoteAddress(req), req.Method, proxiedURI, "Content-Length not provided")
	}

	switch respErr := err.(type) {
	case nil:
		status = http.StatusOK
		resp.Header().Set("Content-Length", fmt.Sprint(expectLength))
		switch req.Method {
		case "HEAD":
			responseLength = 0
		case "GET":
			responseLength, err = io.Copy(resp, reader)
			if err == nil && expectLength > -1 && responseLength != expectLength {
				err = ContentLengthMismatch
			}
		}
	case keepclient.Error:
		if respErr == keepclient.BlockNotFound {
			status = http.StatusNotFound
		} else if respErr.Temporary() {
			status = http.StatusBadGateway
		} else {
			status = 422
		}
	default:
		status = http.StatusInternalServerError
	}
}

var LengthRequiredError = errors.New(http.StatusText(http.StatusLengthRequired))
var LengthMismatchError = errors.New("Locator size hint does not match Content-Length header")

func (h *proxyHandler) Put(resp http.ResponseWriter, req *http.Request) {
	if err := h.checkLoop(resp, req); err != nil {
		return
	}
	SetCorsHeaders(resp)
	resp.Header().Set("Via", "HTTP/1.1 "+viaAlias)

	kc := h.makeKeepClient(req)

	var err error
	var expectLength int64
	var status = http.StatusInternalServerError
	var wroteReplicas int
	var locatorOut string = "-"

	defer func() {
		log.Println(GetRemoteAddress(req), req.Method, req.URL.Path, status, expectLength, kc.Want_replicas, wroteReplicas, locatorOut, err)
		if status != http.StatusOK {
			http.Error(resp, err.Error(), status)
		}
	}()

	locatorIn := mux.Vars(req)["locator"]

	// Check if the client specified storage classes
	if req.Header.Get("X-Keep-Storage-Classes") != "" {
		var scl []string
		for _, sc := range strings.Split(req.Header.Get("X-Keep-Storage-Classes"), ",") {
			scl = append(scl, strings.Trim(sc, " "))
		}
		kc.StorageClasses = scl
	}

	_, err = fmt.Sscanf(req.Header.Get("Content-Length"), "%d", &expectLength)
	if err != nil || expectLength < 0 {
		err = LengthRequiredError
		status = http.StatusLengthRequired
		return
	}

	if locatorIn != "" {
		var loc *keepclient.Locator
		if loc, err = keepclient.MakeLocator(locatorIn); err != nil {
			status = http.StatusBadRequest
			return
		} else if loc.Size > 0 && int64(loc.Size) != expectLength {
			err = LengthMismatchError
			status = http.StatusBadRequest
			return
		}
	}

	var pass bool
	var tok string
	if pass, tok = CheckAuthorizationHeader(kc, h.ApiTokenCache, req); !pass {
		err = BadAuthorizationHeader
		status = http.StatusForbidden
		return
	}

	// Copy ArvadosClient struct and use the client's API token
	arvclient := *kc.Arvados
	arvclient.ApiToken = tok
	kc.Arvados = &arvclient

	// Check if the client specified the number of replicas
	if req.Header.Get("X-Keep-Desired-Replicas") != "" {
		var r int
		_, err := fmt.Sscanf(req.Header.Get(keepclient.X_Keep_Desired_Replicas), "%d", &r)
		if err == nil {
			kc.Want_replicas = r
		}
	}

	// Now try to put the block through
	if locatorIn == "" {
		bytes, err2 := ioutil.ReadAll(req.Body)
		if err2 != nil {
			_ = errors.New(fmt.Sprintf("Error reading request body: %s", err2))
			status = http.StatusInternalServerError
			return
		}
		locatorOut, wroteReplicas, err = kc.PutB(bytes)
	} else {
		locatorOut, wroteReplicas, err = kc.PutHR(locatorIn, req.Body, expectLength)
	}

	// Tell the client how many successful PUTs we accomplished
	resp.Header().Set(keepclient.X_Keep_Replicas_Stored, fmt.Sprintf("%d", wroteReplicas))

	switch err.(type) {
	case nil:
		status = http.StatusOK
		_, err = io.WriteString(resp, locatorOut)

	case keepclient.OversizeBlockError:
		// Too much data
		status = http.StatusRequestEntityTooLarge

	case keepclient.InsufficientReplicasError:
		if wroteReplicas > 0 {
			// At least one write is considered success.  The
			// client can decide if getting less than the number of
			// replications it asked for is a fatal error.
			status = http.StatusOK
			_, err = io.WriteString(resp, locatorOut)
		} else {
			status = http.StatusServiceUnavailable
		}

	default:
		status = http.StatusBadGateway
	}
}

// ServeHTTP implementation for IndexHandler
// Supports only GET requests for /index/{prefix:[0-9a-f]{0,32}}
// For each keep server found in LocalRoots:
//   Invokes GetIndex using keepclient
//   Expects "complete" response (terminating with blank new line)
//   Aborts on any errors
// Concatenates responses from all those keep servers and returns
func (h *proxyHandler) Index(resp http.ResponseWriter, req *http.Request) {
	SetCorsHeaders(resp)

	prefix := mux.Vars(req)["prefix"]
	var err error
	var status int

	defer func() {
		if status != http.StatusOK {
			http.Error(resp, err.Error(), status)
		}
	}()

	kc := h.makeKeepClient(req)
	ok, token := CheckAuthorizationHeader(kc, h.ApiTokenCache, req)
	if !ok {
		status, err = http.StatusForbidden, BadAuthorizationHeader
		return
	}

	// Copy ArvadosClient struct and use the client's API token
	arvclient := *kc.Arvados
	arvclient.ApiToken = token
	kc.Arvados = &arvclient

	// Only GET method is supported
	if req.Method != "GET" {
		status, err = http.StatusNotImplemented, MethodNotSupported
		return
	}

	// Get index from all LocalRoots and write to resp
	var reader io.Reader
	for uuid := range kc.LocalRoots() {
		reader, err = kc.GetIndex(uuid, prefix)
		if err != nil {
			status = http.StatusBadGateway
			return
		}

		_, err = io.Copy(resp, reader)
		if err != nil {
			status = http.StatusBadGateway
			return
		}
	}

	// Got index from all the keep servers and wrote to resp
	status = http.StatusOK
	resp.Write([]byte("\n"))
}

func (h *proxyHandler) makeKeepClient(req *http.Request) *keepclient.KeepClient {
	kc := *h.KeepClient
	kc.RequestID = req.Header.Get("X-Request-Id")
	kc.HTTPClient = &proxyClient{
		client: &http.Client{
			Timeout:   h.timeout,
			Transport: h.transport,
		},
		proto: req.Proto,
	}
	return &kc
}
