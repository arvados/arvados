package main

import (
	"errors"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Default TCP address on which to listen for requests.
// Initialized by the -listen flag.
const DEFAULT_ADDR = ":25107"

var listener net.Listener

func main() {
	var (
		listen           string
		no_get           bool
		no_put           bool
		default_replicas int
		timeout          int64
		pidfile          string
	)

	flagset := flag.NewFlagSet("default", flag.ExitOnError)

	flagset.StringVar(
		&listen,
		"listen",
		DEFAULT_ADDR,
		"Interface on which to listen for requests, in the format "+
			"ipaddr:port. e.g. -listen=10.0.1.24:8000. Use -listen=:port "+
			"to listen on all network interfaces.")

	flagset.BoolVar(
		&no_get,
		"no-get",
		false,
		"If set, disable GET operations")

	flagset.BoolVar(
		&no_put,
		"no-put",
		false,
		"If set, disable PUT operations")

	flagset.IntVar(
		&default_replicas,
		"default-replicas",
		2,
		"Default number of replicas to write if not specified by the client.")

	flagset.Int64Var(
		&timeout,
		"timeout",
		15,
		"Timeout on requests to internal Keep services (default 15 seconds)")

	flagset.StringVar(
		&pidfile,
		"pid",
		"",
		"Path to write pid file")

	flagset.Parse(os.Args[1:])

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %s", err.Error())
	}

	kc, err := keepclient.MakeKeepClient(&arv)
	if err != nil {
		log.Fatalf("Error setting up keep client %s", err.Error())
	}

	if pidfile != "" {
		f, err := os.Create(pidfile)
		if err != nil {
			log.Fatalf("Error writing pid file (%s): %s", pidfile, err.Error())
		}
		fmt.Fprint(f, os.Getpid())
		f.Close()
		defer os.Remove(pidfile)
	}

	kc.Want_replicas = default_replicas

	kc.Client.Timeout = time.Duration(timeout) * time.Second

	listener, err = net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("Could not listen on %v", listen)
	}

	go RefreshServicesList(kc)

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

	log.Printf("Arvados Keep proxy started listening on %v", listener.Addr())

	// Start listening for requests.
	http.Serve(listener, MakeRESTRouter(!no_get, !no_put, kc))

	log.Println("shutting down")
}

type ApiTokenCache struct {
	tokens     map[string]int64
	lock       sync.Mutex
	expireTime int64
}

// Refresh the keep service list every five minutes.
func RefreshServicesList(kc *keepclient.KeepClient) {
	previousRoots := ""
	for {
		if err := kc.DiscoverKeepServers(); err != nil {
			log.Println("Error retrieving services list:", err)
			time.Sleep(3*time.Second)
			previousRoots = ""
		} else if len(kc.LocalRoots()) == 0 {
			log.Println("Received empty services list")
			time.Sleep(3*time.Second)
			previousRoots = ""
		} else {
			newRoots := fmt.Sprint("Locals ", kc.LocalRoots(), ", gateways ", kc.GatewayRoots())
			if newRoots != previousRoots {
				log.Println("Updated services list:", newRoots)
				previousRoots = newRoots
			}
			time.Sleep(300*time.Second)
		}
	}
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
	if realip := req.Header.Get("X-Real-IP"); realip != "" {
		if forwarded := req.Header.Get("X-Forwarded-For"); forwarded != realip {
			return fmt.Sprintf("%s (X-Forwarded-For %s)", realip, forwarded)
		} else {
			return realip
		}
	}
	return req.RemoteAddr
}

func CheckAuthorizationHeader(kc keepclient.KeepClient, cache *ApiTokenCache, req *http.Request) (pass bool, tok string) {
	var auth string
	if auth = req.Header.Get("Authorization"); auth == "" {
		return false, ""
	}

	_, err := fmt.Sscanf(auth, "OAuth2 %s", &tok)
	if err != nil {
		// Scanning error
		return false, ""
	}

	if cache.RecallToken(tok) {
		// Valid in the cache, short circut
		return true, tok
	}

	arv := *kc.Arvados
	arv.ApiToken = tok
	if err := arv.Call("HEAD", "users", "", "current", nil, nil); err != nil {
		log.Printf("%s: CheckAuthorizationHeader error: %v", GetRemoteAddress(req), err)
		return false, ""
	}

	// Success!  Update cache
	cache.RememberToken(tok)

	return true, tok
}

type GetBlockHandler struct {
	*keepclient.KeepClient
	*ApiTokenCache
}

type PutBlockHandler struct {
	*keepclient.KeepClient
	*ApiTokenCache
}

type InvalidPathHandler struct{}

type OptionsHandler struct{}

// MakeRESTRouter
//     Returns a mux.Router that passes GET and PUT requests to the
//     appropriate handlers.
//
func MakeRESTRouter(
	enable_get bool,
	enable_put bool,
	kc *keepclient.KeepClient) *mux.Router {

	t := &ApiTokenCache{tokens: make(map[string]int64), expireTime: 300}

	rest := mux.NewRouter()

	if enable_get {
		rest.Handle(`/{locator:[0-9a-f]{32}\+.*}`,
			GetBlockHandler{kc, t}).Methods("GET", "HEAD")
		rest.Handle(`/{locator:[0-9a-f]{32}}`, GetBlockHandler{kc, t}).Methods("GET", "HEAD")
	}

	if enable_put {
		rest.Handle(`/{locator:[0-9a-f]{32}\+.*}`, PutBlockHandler{kc, t}).Methods("PUT")
		rest.Handle(`/{locator:[0-9a-f]{32}}`, PutBlockHandler{kc, t}).Methods("PUT")
		rest.Handle(`/`, PutBlockHandler{kc, t}).Methods("POST")
		rest.Handle(`/{any}`, OptionsHandler{}).Methods("OPTIONS")
		rest.Handle(`/`, OptionsHandler{}).Methods("OPTIONS")
	}

	rest.NotFoundHandler = InvalidPathHandler{}

	return rest
}

func SetCorsHeaders(resp http.ResponseWriter) {
	resp.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, OPTIONS")
	resp.Header().Set("Access-Control-Allow-Origin", "*")
	resp.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Length, Content-Type, X-Keep-Desired-Replicas")
	resp.Header().Set("Access-Control-Max-Age", "86486400")
}

func (this InvalidPathHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Printf("%s: %s %s unroutable", GetRemoteAddress(req), req.Method, req.URL.Path)
	http.Error(resp, "Bad request", http.StatusBadRequest)
}

func (this OptionsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	log.Printf("%s: %s %s", GetRemoteAddress(req), req.Method, req.URL.Path)
	SetCorsHeaders(resp)
}

var BadAuthorizationHeader = errors.New("Missing or invalid Authorization header")
var ContentLengthMismatch = errors.New("Actual length != expected content length")
var MethodNotSupported = errors.New("Method not supported")

func (this GetBlockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	SetCorsHeaders(resp)

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

	kc := *this.KeepClient

	var pass bool
	var tok string
	if pass, tok = CheckAuthorizationHeader(kc, this.ApiTokenCache, req); !pass {
		status, err = http.StatusForbidden, BadAuthorizationHeader
		return
	}

	// Copy ArvadosClient struct and use the client's API token
	arvclient := *kc.Arvados
	arvclient.ApiToken = tok
	kc.Arvados = &arvclient

	var reader io.ReadCloser

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

	switch err {
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
	case keepclient.BlockNotFound:
		status = http.StatusNotFound
	default:
		status = http.StatusBadGateway
	}
}

var LengthRequiredError = errors.New(http.StatusText(http.StatusLengthRequired))
var LengthMismatchError = errors.New("Locator size hint does not match Content-Length header")

func (this PutBlockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	SetCorsHeaders(resp)

	kc := *this.KeepClient
	var err error
	var expectLength int64 = -1
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

	if req.Header.Get("Content-Length") != "" {
		_, err := fmt.Sscanf(req.Header.Get("Content-Length"), "%d", &expectLength)
		if err != nil {
			resp.Header().Set("Content-Length", fmt.Sprintf("%d", expectLength))
		}

	}

	if expectLength < 0 {
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
	if pass, tok = CheckAuthorizationHeader(kc, this.ApiTokenCache, req); !pass {
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
		if err != nil {
			kc.Want_replicas = r
		}
	}

	// Now try to put the block through
	if locatorIn == "" {
		if bytes, err := ioutil.ReadAll(req.Body); err != nil {
			err = errors.New(fmt.Sprintf("Error reading request body: %s", err))
			status = http.StatusInternalServerError
			return
		} else {
			locatorOut, wroteReplicas, err = kc.PutB(bytes)
		}
	} else {
		locatorOut, wroteReplicas, err = kc.PutHR(locatorIn, req.Body, expectLength)
	}

	// Tell the client how many successful PUTs we accomplished
	resp.Header().Set(keepclient.X_Keep_Replicas_Stored, fmt.Sprintf("%d", wroteReplicas))

	switch err {
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
