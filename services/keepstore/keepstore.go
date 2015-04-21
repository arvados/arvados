package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
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

var (
	flagSerializeIO bool
	flagReadonly    bool
)
type volumeSet []Volume

func (vs *volumeSet) Set(value string) error {
	if dirs := strings.Split(value, ","); len(dirs) > 1 {
		log.Print("DEPRECATED: using comma-separated volume list.")
		for _, dir := range dirs {
			if err := vs.Set(dir); err != nil {
				return err
			}
		}
		return nil
	}
	if len(value) == 0 || value[0] != '/' {
		return errors.New("Invalid volume: must begin with '/'.")
	}
	if _, err := os.Stat(value); err != nil {
		return err
	}
	*vs = append(*vs, MakeUnixVolume(value, flagSerializeIO, flagReadonly))
	return nil
}

func (vs *volumeSet) String() string {
	s := "["
	for i, v := range *vs {
		if i > 0 {
			s = s + " "
		}
		s = s + v.String()
	}
	return s + "]"
}

// Discover adds a volume for every directory named "keep" that is
// located at the top level of a device- or tmpfs-backed mount point
// other than "/". It returns the number of volumes added.
func (vs *volumeSet) Discover() int {
	added := 0
	f, err := os.Open(PROC_MOUNTS)
	if err != nil {
		log.Fatalf("opening %s: %s", PROC_MOUNTS, err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		if err := scanner.Err(); err != nil {
			log.Fatalf("reading %s: %s", PROC_MOUNTS, err)
		}
		dev, mount := args[0], args[1]
		if mount == "/" {
			continue
		}
		if dev != "tmpfs" && !strings.HasPrefix(dev, "/dev/") {
			continue
		}
		keepdir := mount + "/keep"
		if st, err := os.Stat(keepdir); err != nil || !st.IsDir() {
			continue
		}
		// Set the -readonly flag (but only for this volume)
		// if the filesystem is mounted readonly.
		flagReadonlyWas := flagReadonly
		for _, fsopt := range strings.Split(args[3], ",") {
			if fsopt == "ro" {
				flagReadonly = true
				break
			}
			if fsopt == "rw" {
				break
			}
		}
		vs.Set(keepdir)
		flagReadonly = flagReadonlyWas
		added++
	}
	return added
}

// TODO(twp): continue moving as much code as possible out of main
// so it can be effectively tested. Esp. handling and postprocessing
// of command line flags (identifying Keep volumes and initializing
// permission arguments).

func main() {
	log.Println("Keep started: pid", os.Getpid())

	var (
		data_manager_token_file string
		listen                  string
		permission_key_file     string
		permission_ttl_sec      int
		volumes                 volumeSet
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
		"Listening address, in the form \"host:port\". e.g., 10.0.1.24:8000. Omit the host part to listen on all interfaces.")
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
		&flagSerializeIO,
		"serialize",
		false,
		"Serialize read and write operations on the following volumes.")
	flag.BoolVar(
		&flagReadonly,
		"readonly",
		false,
		"Do not write, delete, or touch anything on the following volumes.")
	flag.Var(
		&volumes,
		"volumes",
		"Deprecated synonym for -volume.")
	flag.Var(
		&volumes,
		"volume",
		"Local storage directory. Can be given more than once to add multiple directories. If none are supplied, the default is to use all directories named \"keep\" that exist in the top level directory of a mount point at startup time. Can be a comma-separated list, but this is deprecated: use multiple -volume arguments instead.")
	flag.StringVar(
		&pidfile,
		"pid",
		"",
		"Path to write pid file")

	flag.Parse()

	if len(volumes) == 0 {
		if volumes.Discover() == 0 {
			log.Fatal("No volumes found.")
		}
	}

	for _, v := range volumes {
		log.Printf("Using volume %v (writable=%v)", v, v.Writable())
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
	KeepVM = MakeRRVolumeManager(volumes)

	// Tell the built-in HTTP server to direct all requests to the REST router.
	loggingRouter := MakeLoggingRESTRouter()
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		loggingRouter.ServeHTTP(resp, req)
	})

	// Set up a TCP listener.
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Pull queue and worker
	keepClient := &keepclient.KeepClient{
		Arvados:       nil,
		Want_replicas: 1,
		Using_proxy:   true,
		Client:        &http.Client{},
	}

	// Initialize the pullq and worker
	pullq = NewWorkQueue()
	go RunPullWorker(pullq, keepClient)

	// Initialize the trashq and worker
	trashq = NewWorkQueue()
	go RunTrashWorker(trashq)

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
