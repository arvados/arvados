package main

import (
	"arvados.org/keepclient"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
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
		pidfile          string
	)

	flag.StringVar(
		&listen,
		"listen",
		DEFAULT_ADDR,
		"Interface on which to listen for requests, in the format "+
			"ipaddr:port. e.g. -listen=10.0.1.24:8000. Use -listen=:port "+
			"to listen on all network interfaces.")

	flag.BoolVar(
		&no_get,
		"no-get",
		false,
		"If set, disable GET operations")

	flag.BoolVar(
		&no_get,
		"no-put",
		false,
		"If set, disable PUT operations")

	flag.IntVar(
		&default_replicas,
		"default-replicas",
		2,
		"Default number of replicas to write if not specified by the client.")

	flag.StringVar(
		&pidfile,
		"pid",
		"",
		"Path to write pid file")

	flag.Parse()

	/*if no_get == false {
		log.Print("Must specify -no-get")
		return
	}*/

	kc, err := keepclient.MakeKeepClient()
	if err != nil {
		log.Print(err)
		return
	}

	if pidfile != "" {
		f, err := os.Create(pidfile)
		if err == nil {
			fmt.Fprint(f, os.Getpid())
			f.Close()
		} else {
			log.Printf("Error writing pid file (%s): %s", pidfile, err.Error())
		}
	}

	kc.Want_replicas = default_replicas

	listener, err = net.Listen("tcp", listen)
	if err != nil {
		log.Printf("Could not listen on %v", listen)
		return
	}

	// Start listening for requests.
	http.Serve(listener, MakeRESTRouter(!no_get, !no_put, kc))
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

func CheckAuthorizationHeader(kc keepclient.KeepClient, cache *ApiTokenCache, req *http.Request) bool {
	if req.Header.Get("Authorization") == "" {
		return false
	}

	var tok string
	_, err := fmt.Sscanf(req.Header.Get("Authorization"), "OAuth2 %s", &tok)
	if err != nil {
		// Scanning error
		return false
	}

	if cache.RecallToken(tok) {
		// Valid in the cache, short circut
		return true
	}

	var usersreq *http.Request

	if usersreq, err = http.NewRequest("GET", fmt.Sprintf("https://%s/arvados/v1/users/current", kc.ApiServer), nil); err != nil {
		// Can't construct the request
		log.Print("CheckAuthorizationHeader error: %v", err)
		return false
	}

	// Add api token header
	usersreq.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", tok))

	// Actually make the request
	var resp *http.Response
	if resp, err = kc.Client.Do(usersreq); err != nil {
		// Something else failed
		log.Print("CheckAuthorizationHeader error: %v", err)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		// Bad status
		return false
	}

	// Success!  Update cache
	cache.RememberToken(tok)

	return true
}

type GetBlockHandler struct {
	keepclient.KeepClient
	*ApiTokenCache
}

type PutBlockHandler struct {
	keepclient.KeepClient
	*ApiTokenCache
}

// MakeRESTRouter
//     Returns a mux.Router that passes GET and PUT requests to the
//     appropriate handlers.
//
func MakeRESTRouter(
	enable_get bool,
	enable_put bool,
	kc keepclient.KeepClient) *mux.Router {

	t := &ApiTokenCache{tokens: make(map[string]int64), expireTime: 300}

	rest := mux.NewRouter()
	gh := rest.Handle(`/{hash:[0-9a-f]{32}}`, GetBlockHandler{kc, t})
	ghsig := rest.Handle(
		`/{hash:[0-9a-f]{32}}+A{signature:[0-9a-f]+}@{timestamp:[0-9a-f]+}`,
		GetBlockHandler{kc, t})
	ph := rest.Handle(`/{hash:[0-9a-f]{32}}`, PutBlockHandler{kc, t})

	if enable_get {
		gh.Methods("GET", "HEAD")
		ghsig.Methods("GET", "HEAD")
	}

	if enable_put {
		ph.Methods("PUT")
	}

	return rest
}

func (this GetBlockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	if !CheckAuthorizationHeader(this.KeepClient, this.ApiTokenCache, req) {
		http.Error(resp, "Missing or invalid Authorization header", http.StatusForbidden)
	}

	hash := mux.Vars(req)["hash"]
	signature := mux.Vars(req)["signature"]
	timestamp := mux.Vars(req)["timestamp"]

	var reader io.ReadCloser
	var err error
	var blocklen int64

	if req.Method == "GET" {
		reader, blocklen, _, err = this.KeepClient.AuthorizedGet(hash, signature, timestamp)
		defer reader.Close()
	} else if req.Method == "HEAD" {
		blocklen, _, err = this.KeepClient.AuthorizedAsk(hash, signature, timestamp)
	}

	resp.Header().Set("Content-Length", fmt.Sprint(blocklen))

	switch err {
	case nil:
		if reader != nil {
			io.Copy(resp, reader)
		}
	case keepclient.BlockNotFound:
		http.Error(resp, "Not found", http.StatusNotFound)
	default:
		http.Error(resp, err.Error(), http.StatusBadGateway)
	}
}

func (this PutBlockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {

	log.Print("PutBlockHandler start")

	if !CheckAuthorizationHeader(this.KeepClient, this.ApiTokenCache, req) {
		http.Error(resp, "Missing or invalid Authorization header", http.StatusForbidden)
	}

	hash := mux.Vars(req)["hash"]

	var contentLength int64 = -1
	if req.Header.Get("Content-Length") != "" {
		_, err := fmt.Sscanf(req.Header.Get("Content-Length"), "%d", &contentLength)
		if err != nil {
			resp.Header().Set("Content-Length", fmt.Sprintf("%d", contentLength))
		}

	}

	if contentLength < 1 {
		http.Error(resp, "Must include Content-Length header", http.StatusLengthRequired)
		return
	}

	// Check if the client specified the number of replicas
	if req.Header.Get("X-Keep-Desired-Replicas") != "" {
		var r int
		_, err := fmt.Sscanf(req.Header.Get("X-Keep-Desired-Replicas"), "%d", &r)
		if err != nil {
			this.KeepClient.Want_replicas = r
		}
	}

	// Now try to put the block through
	replicas, err := this.KeepClient.PutHR(hash, req.Body, contentLength)

	log.Printf("Replicas stored: %v err: %v", replicas, err)

	// Tell the client how many successful PUTs we accomplished
	resp.Header().Set("X-Keep-Replicas-Stored", fmt.Sprintf("%d", replicas))

	switch err {
	case nil:
		// Default will return http.StatusOK

	case keepclient.OversizeBlockError:
		// Too much data
		http.Error(resp, fmt.Sprintf("Exceeded maximum blocksize %d", keepclient.BLOCKSIZE), http.StatusRequestEntityTooLarge)

	case keepclient.InsufficientReplicasError:
		if replicas > 0 {
			// At least one write is considered success.  The
			// client can decide if getting less than the number of
			// replications it asked for is a fatal error.
			// Default will return http.StatusOK
		} else {
			http.Error(resp, "", http.StatusServiceUnavailable)
		}

	default:
		http.Error(resp, err.Error(), http.StatusBadGateway)
	}

}
