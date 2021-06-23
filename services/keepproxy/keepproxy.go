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

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/coreos/go-systemd/daemon"
	"github.com/ghodss/yaml"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var version = "dev"

var (
	listener net.Listener
	router   http.Handler
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

func configure(logger log.FieldLogger, args []string) (*arvados.Cluster, error) {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)

	dumpConfig := flags.Bool("dump-config", false, "write current configuration to stdout and exit")
	getVersion := flags.Bool("version", false, "Print version information and exit.")

	loader := config.NewLoader(os.Stdin, logger)
	loader.SetupFlags(flags)

	args = loader.MungeLegacyConfigArgs(logger, args[1:], "-legacy-keepproxy-config")
	flags.Parse(args)

	// Print version information if requested
	if *getVersion {
		fmt.Printf("keepproxy %s\n", version)
		return nil, nil
	}

	cfg, err := loader.Load()
	if err != nil {
		return nil, err
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return nil, err
	}

	if *dumpConfig {
		out, err := yaml.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stdout.Write(out); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return cluster, nil
}

func main() {
	logger := log.New()
	logger.Formatter = &log.JSONFormatter{
		TimestampFormat: rfc3339NanoFixed,
	}

	cluster, err := configure(logger, os.Args)
	if err != nil {
		log.Fatal(err)
	}
	if cluster == nil {
		return
	}

	log.Printf("keepproxy %s started", version)

	if err := run(logger, cluster); err != nil {
		log.Fatal(err)
	}

	log.Println("shutting down")
}

func run(logger log.FieldLogger, cluster *arvados.Cluster) error {
	client, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return err
	}
	client.AuthToken = cluster.SystemRootToken

	arv, err := arvadosclient.New(client)
	if err != nil {
		return fmt.Errorf("Error setting up arvados client %v", err)
	}

	// If a config file is available, use the keepstores defined there
	// instead of the legacy autodiscover mechanism via the API server
	for k := range cluster.Services.Keepstore.InternalURLs {
		arv.KeepServiceURIs = append(arv.KeepServiceURIs, strings.TrimRight(k.String(), "/"))
	}

	if cluster.SystemLogs.LogLevel == "debug" {
		keepclient.DebugPrintf = log.Printf
	}
	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		return fmt.Errorf("Error setting up keep client %v", err)
	}
	keepclient.RefreshServiceDiscoveryOnSIGHUP()

	if cluster.Collections.DefaultReplication > 0 {
		kc.Want_replicas = cluster.Collections.DefaultReplication
	}

	var listen arvados.URL
	for listen = range cluster.Services.Keepproxy.InternalURLs {
		break
	}

	var lErr error
	listener, lErr = net.Listen("tcp", listen.Host)
	if lErr != nil {
		return fmt.Errorf("listen(%s): %v", listen.Host, lErr)
	}

	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}
	log.Println("listening at", listener.Addr())

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
	router = MakeRESTRouter(kc, time.Duration(keepclient.DefaultProxyRequestTimeout), cluster.ManagementToken)
	return http.Serve(listener, httpserver.AddRequestIDs(httpserver.LogRequests(router)))
}

type APITokenCache struct {
	tokens     map[string]int64
	lock       sync.Mutex
	expireTime int64
}

// RememberToken caches the token and set an expire time.  If we already have
// an expire time on the token, it is not updated.
func (cache *APITokenCache) RememberToken(token string) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	now := time.Now().Unix()
	if cache.tokens[token] == 0 {
		cache.tokens[token] = now + cache.expireTime
	}
}

// RecallToken checks if the cached token is known and still believed to be
// valid.
func (cache *APITokenCache) RecallToken(token string) bool {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	now := time.Now().Unix()
	if cache.tokens[token] == 0 {
		// Unknown token
		return false
	} else if now < cache.tokens[token] {
		// Token is known and still valid
		return true
	} else {
		// Token is expired
		cache.tokens[token] = 0
		return false
	}
}

// GetRemoteAddress returns a string with the remote address for the request.
// If the X-Forwarded-For header is set and has a non-zero length, it returns a
// string made from a comma separated list of all the remote addresses,
// starting with the one(s) from the X-Forwarded-For header.
func GetRemoteAddress(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		return xff + "," + req.RemoteAddr
	}
	return req.RemoteAddr
}

func CheckAuthorizationHeader(kc *keepclient.KeepClient, cache *APITokenCache, req *http.Request) (pass bool, tok string) {
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
	*APITokenCache
	timeout   time.Duration
	transport *http.Transport
}

// MakeRESTRouter returns an http.Handler that passes GET and PUT
// requests to the appropriate handlers.
func MakeRESTRouter(kc *keepclient.KeepClient, timeout time.Duration, mgmtToken string) http.Handler {
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
		APITokenCache: &APITokenCache{
			tokens:     make(map[string]int64),
			expireTime: 300,
		},
	}

	rest.HandleFunc(`/{locator:[0-9a-f]{32}\+.*}`, h.Get).Methods("GET", "HEAD")
	rest.HandleFunc(`/{locator:[0-9a-f]{32}}`, h.Get).Methods("GET", "HEAD")

	// List all blocks
	rest.HandleFunc(`/index`, h.Index).Methods("GET")

	// List blocks whose hash has the given prefix
	rest.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, h.Index).Methods("GET")

	rest.HandleFunc(`/{locator:[0-9a-f]{32}\+.*}`, h.Put).Methods("PUT")
	rest.HandleFunc(`/{locator:[0-9a-f]{32}}`, h.Put).Methods("PUT")
	rest.HandleFunc(`/`, h.Put).Methods("POST")
	rest.HandleFunc(`/{any}`, h.Options).Methods("OPTIONS")
	rest.HandleFunc(`/`, h.Options).Methods("OPTIONS")

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

var errBadAuthorizationHeader = errors.New("Missing or invalid Authorization header")
var errContentLengthMismatch = errors.New("Actual length != expected content length")
var errMethodNotSupported = errors.New("Method not supported")

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
	if pass, tok = CheckAuthorizationHeader(kc, h.APITokenCache, req); !pass {
		status, err = http.StatusForbidden, errBadAuthorizationHeader
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
		status, err = http.StatusNotImplemented, errMethodNotSupported
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
				err = errContentLengthMismatch
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

var errLengthRequired = errors.New(http.StatusText(http.StatusLengthRequired))
var errLengthMismatch = errors.New("Locator size hint does not match Content-Length header")

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
		err = errLengthRequired
		status = http.StatusLengthRequired
		return
	}

	if locatorIn != "" {
		var loc *keepclient.Locator
		if loc, err = keepclient.MakeLocator(locatorIn); err != nil {
			status = http.StatusBadRequest
			return
		} else if loc.Size > 0 && int64(loc.Size) != expectLength {
			err = errLengthMismatch
			status = http.StatusBadRequest
			return
		}
	}

	var pass bool
	var tok string
	if pass, tok = CheckAuthorizationHeader(kc, h.APITokenCache, req); !pass {
		err = errBadAuthorizationHeader
		status = http.StatusForbidden
		return
	}

	// Copy ArvadosClient struct and use the client's API token
	arvclient := *kc.Arvados
	arvclient.ApiToken = tok
	kc.Arvados = &arvclient

	// Check if the client specified the number of replicas
	if desiredReplicas := req.Header.Get(keepclient.XKeepDesiredReplicas); desiredReplicas != "" {
		var r int
		_, err := fmt.Sscanf(desiredReplicas, "%d", &r)
		if err == nil {
			kc.Want_replicas = r
		}
	}

	// Now try to put the block through
	if locatorIn == "" {
		bytes, err2 := ioutil.ReadAll(req.Body)
		if err2 != nil {
			err = fmt.Errorf("Error reading request body: %s", err2)
			status = http.StatusInternalServerError
			return
		}
		locatorOut, wroteReplicas, err = kc.PutB(bytes)
	} else {
		locatorOut, wroteReplicas, err = kc.PutHR(locatorIn, req.Body, expectLength)
	}

	// Tell the client how many successful PUTs we accomplished
	resp.Header().Set(keepclient.XKeepReplicasStored, fmt.Sprintf("%d", wroteReplicas))

	switch err.(type) {
	case nil:
		status = http.StatusOK
		if len(kc.StorageClasses) > 0 {
			// A successful PUT request with storage classes means that all
			// storage classes were fulfilled, so the client will get a
			// confirmation via the X-Storage-Classes-Confirmed header.
			hdr := ""
			isFirst := true
			for _, sc := range kc.StorageClasses {
				if isFirst {
					hdr = fmt.Sprintf("%s=%d", sc, wroteReplicas)
					isFirst = false
				} else {
					hdr += fmt.Sprintf(", %s=%d", sc, wroteReplicas)
				}
			}
			resp.Header().Set(keepclient.XKeepStorageClassesConfirmed, hdr)
		}
		_, err = io.WriteString(resp, locatorOut)
	case keepclient.OversizeBlockError:
		// Too much data
		status = http.StatusRequestEntityTooLarge
	case keepclient.InsufficientReplicasError:
		status = http.StatusServiceUnavailable
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
	ok, token := CheckAuthorizationHeader(kc, h.APITokenCache, req)
	if !ok {
		status, err = http.StatusForbidden, errBadAuthorizationHeader
		return
	}

	// Copy ArvadosClient struct and use the client's API token
	arvclient := *kc.Arvados
	arvclient.ApiToken = token
	kc.Arvados = &arvclient

	// Only GET method is supported
	if req.Method != "GET" {
		status, err = http.StatusNotImplemented, errMethodNotSupported
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
