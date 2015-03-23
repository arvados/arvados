package main

import (
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

	go RefreshServicesList(&kc)

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

	log.Printf("Arvados Keep proxy started listening on %v with server list %v", listener.Addr(), kc.ServiceRoots())

	// Start listening for requests.
	http.Serve(listener, MakeRESTRouter(!no_get, !no_put, &kc))

	log.Println("shutting down")
}

type ApiTokenCache struct {
	tokens     map[string]int64
	lock       sync.Mutex
	expireTime int64
}

// Refresh the keep service list every five minutes.
func RefreshServicesList(kc *keepclient.KeepClient) {
	for {
		time.Sleep(300 * time.Second)
		oldservices := kc.ServiceRoots()
		kc.DiscoverKeepServers()
		newservices := kc.ServiceRoots()
		s1 := fmt.Sprint(oldservices)
		s2 := fmt.Sprint(newservices)
		if s1 != s2 {
			log.Printf("Updated server list to %v", s2)
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
		rest.Handle(`/{hash:[0-9a-f]{32}}+{hints}`,
			GetBlockHandler{kc, t}).Methods("GET", "HEAD")
		rest.Handle(`/{hash:[0-9a-f]{32}}`, GetBlockHandler{kc, t}).Methods("GET", "HEAD")
	}

	if enable_put {
		rest.Handle(`/{hash:[0-9a-f]{32}}+{hints}`, PutBlockHandler{kc, t}).Methods("PUT")
		rest.Handle(`/{hash:[0-9a-f]{32}}`, PutBlockHandler{kc, t}).Methods("PUT")
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

func (this GetBlockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	SetCorsHeaders(resp)

	kc := *this.KeepClient

	hash := mux.Vars(req)["hash"]
	hints := mux.Vars(req)["hints"]

	locator := keepclient.MakeLocator2(hash, hints)

	log.Printf("%s: %s %s begin", GetRemoteAddress(req), req.Method, hash)

	var pass bool
	var tok string
	if pass, tok = CheckAuthorizationHeader(kc, this.ApiTokenCache, req); !pass {
		http.Error(resp, "Missing or invalid Authorization header", http.StatusForbidden)
		return
	}

	// Copy ArvadosClient struct and use the client's API token
	arvclient := *kc.Arvados
	arvclient.ApiToken = tok
	kc.Arvados = &arvclient

	var reader io.ReadCloser
	var err error
	var blocklen int64

	if req.Method == "GET" {
		reader, blocklen, _, err = kc.AuthorizedGet(hash, locator.Signature, locator.Timestamp)
		if reader != nil {
			defer reader.Close()
		}
	} else if req.Method == "HEAD" {
		blocklen, _, err = kc.AuthorizedAsk(hash, locator.Signature, locator.Timestamp)
	}

	if blocklen == -1 {
		log.Printf("%s: %s %s Keep server did not return Content-Length",
			GetRemoteAddress(req), req.Method, hash)
	}

	var status = 0
	switch err {
	case nil:
		status = http.StatusOK
		resp.Header().Set("Content-Length", fmt.Sprint(blocklen))
		if reader != nil {
			n, err2 := io.Copy(resp, reader)
			if blocklen > -1 && n != blocklen {
				log.Printf("%s: %s %s %v %v mismatched copy size expected Content-Length: %v",
					GetRemoteAddress(req), req.Method, hash, status, n, blocklen)
			} else if err2 == nil {
				log.Printf("%s: %s %s %v %v",
					GetRemoteAddress(req), req.Method, hash, status, n)
			} else {
				log.Printf("%s: %s %s %v %v copy error: %v",
					GetRemoteAddress(req), req.Method, hash, status, n, err2.Error())
			}
		} else {
			log.Printf("%s: %s %s %v 0", GetRemoteAddress(req), req.Method, hash, status)
		}
	case keepclient.BlockNotFound:
		status = http.StatusNotFound
		http.Error(resp, "Not Found", http.StatusNotFound)
	default:
		status = http.StatusBadGateway
		http.Error(resp, err.Error(), http.StatusBadGateway)
	}

	if err != nil {
		log.Printf("%s: %s %s %v error: %v",
			GetRemoteAddress(req), req.Method, hash, status, err.Error())
	}
}

func (this PutBlockHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	SetCorsHeaders(resp)

	kc := *this.KeepClient

	hash := mux.Vars(req)["hash"]
	hints := mux.Vars(req)["hints"]

	locator := keepclient.MakeLocator2(hash, hints)

	var contentLength int64 = -1
	if req.Header.Get("Content-Length") != "" {
		_, err := fmt.Sscanf(req.Header.Get("Content-Length"), "%d", &contentLength)
		if err != nil {
			resp.Header().Set("Content-Length", fmt.Sprintf("%d", contentLength))
		}

	}

	log.Printf("%s: %s %s Content-Length %v", GetRemoteAddress(req), req.Method, hash, contentLength)

	if contentLength < 0 {
		http.Error(resp, "Must include Content-Length header", http.StatusLengthRequired)
		return
	}

	if locator.Size > 0 && int64(locator.Size) != contentLength {
		http.Error(resp, "Locator size hint does not match Content-Length header", http.StatusBadRequest)
		return
	}

	var pass bool
	var tok string
	if pass, tok = CheckAuthorizationHeader(kc, this.ApiTokenCache, req); !pass {
		http.Error(resp, "Missing or invalid Authorization header", http.StatusForbidden)
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
	var replicas int
	var put_err error
	if hash == "" {
		if bytes, err := ioutil.ReadAll(req.Body); err != nil {
			msg := fmt.Sprintf("Error reading request body: %s", err)
			log.Printf(msg)
			http.Error(resp, msg, http.StatusInternalServerError)
			return
		} else {
			hash, replicas, put_err = kc.PutB(bytes)
		}
	} else {
		hash, replicas, put_err = kc.PutHR(hash, req.Body, contentLength)
	}

	// Tell the client how many successful PUTs we accomplished
	resp.Header().Set(keepclient.X_Keep_Replicas_Stored, fmt.Sprintf("%d", replicas))

	switch put_err {
	case nil:
		// Default will return http.StatusOK
		log.Printf("%s: %s %s finished, stored %v replicas (desired %v)", GetRemoteAddress(req), req.Method, hash, replicas, kc.Want_replicas)
		n, err2 := io.WriteString(resp, hash)
		if err2 != nil {
			log.Printf("%s: wrote %v bytes to response body and got error %v", n, err2.Error())
		}

	case keepclient.OversizeBlockError:
		// Too much data
		http.Error(resp, fmt.Sprintf("Exceeded maximum blocksize %d", keepclient.BLOCKSIZE), http.StatusRequestEntityTooLarge)

	case keepclient.InsufficientReplicasError:
		if replicas > 0 {
			// At least one write is considered success.  The
			// client can decide if getting less than the number of
			// replications it asked for is a fatal error.
			// Default will return http.StatusOK
			n, err2 := io.WriteString(resp, hash)
			if err2 != nil {
				log.Printf("%s: wrote %v bytes to response body and got error %v", n, err2.Error())
			}
		} else {
			http.Error(resp, put_err.Error(), http.StatusServiceUnavailable)
		}

	default:
		http.Error(resp, put_err.Error(), http.StatusBadGateway)
	}

	if put_err != nil {
		log.Printf("%s: %s %s stored %v replicas (desired %v) got error %v", GetRemoteAddress(req), req.Method, hash, replicas, kc.Want_replicas, put_err.Error())
	}

}
