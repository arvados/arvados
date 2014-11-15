package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"git.curoverse.com/arvados.git/services/keep"
)

// ======================
// Configuration settings
//
// TODO(twp): make all of these configurable via command line flags
// and/or configuration file settings.

// Default TCP address on which to listen for requests.
// Initialized by the --listen flag.
const DEFAULT_ADDR = ":25107"

// A Keep "block" is 64MB.
const BLOCKSIZE = 64 * 1024 * 1024

// A Keep volume must have at least MIN_FREE_KILOBYTES available
// in order to permit writes.
const MIN_FREE_KILOBYTES = BLOCKSIZE / 1024

var PROC_MOUNTS = "/proc/mounts"

// enforce_permissions controls whether permission signatures
// should be enforced (affecting GET and DELETE requests).
// Initialized by the --enforce-permissions flag.
var enforce_permissions bool

// permission_ttl is the time duration for which new permission
// signatures (returned by PUT requests) will be valid.
// Initialized by the --permission-ttl flag.
var permission_ttl time.Duration

// data_manager_token represents the API token used by the
// Data Manager, and is required on certain privileged operations.
// Initialized by the --data-manager-token-file flag.
var data_manager_token string

// never_delete can be used to prevent the DELETE handler from
// actually deleting anything.
var never_delete = false

// ==========
// Error types.
//
type KeepError struct {
	HTTPCode int
	ErrMsg   string
}

var (
	BadRequestError     = &KeepError{400, "Bad Request"}
	UnauthorizedError   = &KeepError{401, "Unauthorized"}
	CollisionError      = &KeepError{500, "Collision"}
	RequestHashError    = &KeepError{422, "Hash mismatch in request"}
	PermissionError     = &KeepError{403, "Forbidden"}
	DiskHashError       = &KeepError{500, "Hash mismatch in stored data"}
	ExpiredError        = &KeepError{401, "Expired permission signature"}
	NotFoundError       = &KeepError{404, "Not Found"}
	GenericError        = &KeepError{500, "Fail"}
	FullError           = &KeepError{503, "Full"}
	TooLongError        = &KeepError{504, "Timeout"}
	MethodDisabledError = &KeepError{405, "Method disabled"}
)

func (e *KeepError) Error() string {
	return e.ErrMsg
}

// ========================
// Internal data structures
//
// These global variables are used by multiple parts of the
// program. They are good candidates for moving into their own
// packages.

// The Keep VolumeManager maintains a list of available volumes.
// Initialized by the --volumes flag (or by FindKeepVolumes).
var KeepVM VolumeManager

// The pull list manager and trash queue are threadsafe queues which
// support atomic update operations. The PullHandler and TrashHandler
// store results from Data Manager /pull and /trash requests here.
//
// See the Keep and Data Manager design documents for more details:
// https://arvados.org/projects/arvados/wiki/Keep_Design_Doc
// https://arvados.org/projects/arvados/wiki/Data_Manager_Design_Doc
//
var pullq *WorkQueue
var trashq *WorkQueue

// TODO(twp): continue moving as much code as possible out of main
// so it can be effectively tested. Esp. handling and postprocessing
// of command line flags (identifying Keep volumes and initializing
// permission arguments).

func main() {
	log.Println("Keep started: pid", os.Getpid())

	// Parse command-line flags:
	//
	// -listen=ipaddr:port
	//    Interface on which to listen for requests. Use :port without
	//    an ipaddr to listen on all network interfaces.
	//    Examples:
	//      -listen=127.0.0.1:4949
	//      -listen=10.0.1.24:8000
	//      -listen=:25107 (to listen to port 25107 on all interfaces)
	//
	// -volumes
	//    A comma-separated list of directories to use as Keep volumes.
	//    Example:
	//      -volumes=/var/keep01,/var/keep02,/var/keep03/subdir
	//
	//    If -volumes is empty or is not present, Keep will select volumes
	//    by looking at currently mounted filesystems for /keep top-level
	//    directories.

	var (
		data_manager_token_file string
		listen                  string
		permission_key_file     string
		permission_ttl_sec      int
		serialize_io            bool
		volumearg               string
		pidfile                 string
	)
	flag.StringVar(
		&data_manager_token_file,
		"data-manager-token-file",
		"",
		"File with the API token used by the Data Manager. All DELETE "+
			"requests or GET /index requests must carry this token.")
	flag.BoolVar(
		&enforce_permissions,
		"enforce-permissions",
		false,
		"Enforce permission signatures on requests.")
	flag.StringVar(
		&listen,
		"listen",
		DEFAULT_ADDR,
		"Interface on which to listen for requests, in the format "+
			"ipaddr:port. e.g. -listen=10.0.1.24:8000. Use -listen=:port "+
			"to listen on all network interfaces.")
	flag.BoolVar(
		&never_delete,
		"never-delete",
		false,
		"If set, nothing will be deleted. HTTP 405 will be returned "+
			"for valid DELETE requests.")
	flag.StringVar(
		&permission_key_file,
		"permission-key-file",
		"",
		"File containing the secret key for generating and verifying "+
			"permission signatures.")
	flag.IntVar(
		&permission_ttl_sec,
		"permission-ttl",
		1209600,
		"Expiration time (in seconds) for newly generated permission "+
			"signatures.")
	flag.BoolVar(
		&serialize_io,
		"serialize",
		false,
		"If set, all read and write operations on local Keep volumes will "+
			"be serialized.")
	flag.StringVar(
		&volumearg,
		"volumes",
		"",
		"Comma-separated list of directories to use for Keep volumes, "+
			"e.g. -volumes=/var/keep1,/var/keep2. If empty or not "+
			"supplied, Keep will scan mounted filesystems for volumes "+
			"with a /keep top-level directory.")

	flag.StringVar(
		&pidfile,
		"pid",
		"",
		"Path to write pid file")

	flag.Parse()

	// Look for local keep volumes.
	var keepvols []string
	if volumearg == "" {
		// TODO(twp): decide whether this is desirable default behavior.
		// In production we may want to require the admin to specify
		// Keep volumes explicitly.
		keepvols = FindKeepVolumes()
	} else {
		keepvols = strings.Split(volumearg, ",")
	}

	// Check that the specified volumes actually exist.
	var goodvols []Volume = nil
	for _, v := range keepvols {
		if _, err := os.Stat(v); err == nil {
			log.Println("adding Keep volume:", v)
			newvol := MakeUnixVolume(v, serialize_io)
			goodvols = append(goodvols, &newvol)
		} else {
			log.Printf("bad Keep volume: %s\n", err)
		}
	}

	if len(goodvols) == 0 {
		log.Fatal("could not find any keep volumes")
	}

	// Initialize data manager token and permission key.
	// If these tokens are specified but cannot be read,
	// raise a fatal error.
	if data_manager_token_file != "" {
		if buf, err := ioutil.ReadFile(data_manager_token_file); err == nil {
			data_manager_token = strings.TrimSpace(string(buf))
		} else {
			log.Fatalf("reading data manager token: %s\n", err)
		}
	}
	if permission_key_file != "" {
		if buf, err := ioutil.ReadFile(permission_key_file); err == nil {
			PermissionSecret = bytes.TrimSpace(buf)
		} else {
			log.Fatalf("reading permission key: %s\n", err)
		}
	}

	// Initialize permission TTL
	permission_ttl = time.Duration(permission_ttl_sec) * time.Second

	// If --enforce-permissions is true, we must have a permission key
	// to continue.
	if PermissionSecret == nil {
		if enforce_permissions {
			log.Fatal("--enforce-permissions requires a permission key")
		} else {
			log.Println("Running without a PermissionSecret. Block locators " +
				"returned by this server will not be signed, and will be rejected " +
				"by a server that enforces permissions.")
			log.Println("To fix this, run Keep with --permission-key-file=<path> " +
				"to define the location of a file containing the permission key.")
		}
	}

	// Start a round-robin VolumeManager with the volumes we have found.
	KeepVM = MakeRRVolumeManager(goodvols)

	// Tell the built-in HTTP server to direct all requests to the REST router.
  routerWrapper := keep_utils.MakeRESTRouterWrapper(MakeRESTRouter())
  http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
    routerWrapper.ServeHTTP(resp, req)
  })

	// Set up a TCP listener.
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
	}

	// Shut down the server gracefully (by closing the listener)
	// if SIGTERM is received.
	term := make(chan os.Signal, 1)
	go func(sig <-chan os.Signal) {
		s := <-sig
		log.Println("caught signal:", s)
		listener.Close()
	}(term)
	signal.Notify(term, syscall.SIGTERM)

	if pidfile != "" {
		f, err := os.Create(pidfile)
		if err == nil {
			fmt.Fprint(f, os.Getpid())
			f.Close()
		} else {
			log.Printf("Error writing pid file (%s): %s", pidfile, err.Error())
		}
	}

	// Start listening for requests.
	srv := &http.Server{Addr: listen}
	srv.Serve(listener)

	log.Println("shutting down")

	if pidfile != "" {
		os.Remove(pidfile)
	}
}
