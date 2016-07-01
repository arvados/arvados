package main

import (
	"bytes"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
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
const DefaultAddr = ":25107"

// A Keep "block" is 64MB.
const BlockSize = 64 * 1024 * 1024

// A Keep volume must have at least MinFreeKilobytes available
// in order to permit writes.
const MinFreeKilobytes = BlockSize / 1024

// ProcMounts /proc/mounts
var ProcMounts = "/proc/mounts"

// enforcePermissions controls whether permission signatures
// should be enforced (affecting GET and DELETE requests).
// Initialized by the -enforce-permissions flag.
var enforcePermissions bool

// blobSignatureTTL is the time duration for which new permission
// signatures (returned by PUT requests) will be valid.
// Initialized by the -permission-ttl flag.
var blobSignatureTTL time.Duration

// dataManagerToken represents the API token used by the
// Data Manager, and is required on certain privileged operations.
// Initialized by the -data-manager-token-file flag.
var dataManagerToken string

// neverDelete can be used to prevent the DELETE handler from
// actually deleting anything.
var neverDelete = true

// trashLifetime is the time duration after a block is trashed
// during which it can be recovered using an /untrash request
// Use 10s or 10m or 10h to set as 10 seconds or minutes or hours respectively.
var trashLifetime time.Duration

// trashCheckInterval is the time duration at which the emptyTrash goroutine
// will check and delete expired trashed blocks. Default is one day.
// Use 10s or 10m or 10h to set as 10 seconds or minutes or hours respectively.
var trashCheckInterval time.Duration

var maxBuffers = 128
var bufs *bufferPool

// KeepError types.
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
	SizeRequiredError   = &KeepError{411, "Missing Content-Length"}
	TooLongError        = &KeepError{413, "Block is too large"}
	MethodDisabledError = &KeepError{405, "Method disabled"}
	ErrNotImplemented   = &KeepError{500, "Unsupported configuration"}
	ErrClientDisconnect = &KeepError{503, "Client disconnected"}
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

type volumeSet []Volume

var (
	flagSerializeIO bool
	flagReadonly    bool
	volumes         volumeSet
)

func (vs *volumeSet) String() string {
	return fmt.Sprintf("%+v", (*vs)[:])
}

// TODO(twp): continue moving as much code as possible out of main
// so it can be effectively tested. Esp. handling and postprocessing
// of command line flags (identifying Keep volumes and initializing
// permission arguments).

func main() {
	log.Println("keepstore starting, pid", os.Getpid())
	defer log.Println("keepstore exiting, pid", os.Getpid())

	var (
		dataManagerTokenFile string
		listen               string
		blobSigningKeyFile   string
		permissionTTLSec     int
		pidfile              string
		maxRequests          int
	)
	flag.StringVar(
		&dataManagerTokenFile,
		"data-manager-token-file",
		"",
		"File with the API token used by the Data Manager. All DELETE "+
			"requests or GET /index requests must carry this token.")
	flag.BoolVar(
		&enforcePermissions,
		"enforce-permissions",
		false,
		"Enforce permission signatures on requests.")
	flag.StringVar(
		&listen,
		"listen",
		DefaultAddr,
		"Listening address, in the form \"host:port\". e.g., 10.0.1.24:8000. Omit the host part to listen on all interfaces.")
	flag.IntVar(
		&maxRequests,
		"max-requests",
		0,
		"Maximum concurrent requests. When this limit is reached, new requests will receive 503 responses. Note: this limit does not include idle connections from clients using HTTP keepalive, so it does not strictly limit the number of concurrent connections. (default 2 * max-buffers)")
	flag.BoolVar(
		&neverDelete,
		"never-delete",
		true,
		"If true, nothing will be deleted. "+
			"Warning: the relevant features in keepstore and data manager have not been extensively tested. "+
			"You should leave this option alone unless you can afford to lose data.")
	flag.StringVar(
		&blobSigningKeyFile,
		"permission-key-file",
		"",
		"Synonym for -blob-signing-key-file.")
	flag.StringVar(
		&blobSigningKeyFile,
		"blob-signing-key-file",
		"",
		"File containing the secret key for generating and verifying "+
			"blob permission signatures.")
	flag.IntVar(
		&permissionTTLSec,
		"permission-ttl",
		0,
		"Synonym for -blob-signature-ttl.")
	flag.IntVar(
		&permissionTTLSec,
		"blob-signature-ttl",
		2*7*24*3600,
		"Lifetime of blob permission signatures in seconds. Modifying the ttl will invalidate all existing signatures. "+
			"See services/api/config/application.default.yml.")
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
	flag.StringVar(
		&pidfile,
		"pid",
		"",
		"Path to write pid file during startup. This file is kept open and locked with LOCK_EX until keepstore exits, so `fuser -k pidfile` is one way to shut down. Exit immediately if there is an error opening, locking, or writing the pid file.")
	flag.IntVar(
		&maxBuffers,
		"max-buffers",
		maxBuffers,
		fmt.Sprintf("Maximum RAM to use for data buffers, given in multiples of block size (%d MiB). When this limit is reached, HTTP requests requiring buffers (like GET and PUT) will wait for buffer space to be released.", BlockSize>>20))
	flag.DurationVar(
		&trashLifetime,
		"trash-lifetime",
		0*time.Second,
		"Time duration after a block is trashed during which it can be recovered using an /untrash request")
	flag.DurationVar(
		&trashCheckInterval,
		"trash-check-interval",
		24*time.Hour,
		"Time duration at which the emptyTrash goroutine will check and delete expired trashed blocks. Default is one day.")

	flag.Parse()

	if maxBuffers < 0 {
		log.Fatal("-max-buffers must be greater than zero.")
	}
	bufs = newBufferPool(maxBuffers, BlockSize)

	if pidfile != "" {
		f, err := os.OpenFile(pidfile, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			log.Fatalf("open pidfile (%s): %s", pidfile, err)
		}
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			log.Fatalf("flock pidfile (%s): %s", pidfile, err)
		}
		err = f.Truncate(0)
		if err != nil {
			log.Fatalf("truncate pidfile (%s): %s", pidfile, err)
		}
		_, err = fmt.Fprint(f, os.Getpid())
		if err != nil {
			log.Fatalf("write pidfile (%s): %s", pidfile, err)
		}
		err = f.Sync()
		if err != nil {
			log.Fatalf("sync pidfile (%s): %s", pidfile, err)
		}
		defer f.Close()
		defer os.Remove(pidfile)
	}

	if len(volumes) == 0 {
		if (&unixVolumeAdder{&volumes}).Discover() == 0 {
			log.Fatal("No volumes found.")
		}
	}

	for _, v := range volumes {
		log.Printf("Using volume %v (writable=%v)", v, v.Writable())
	}

	// Initialize data manager token and permission key.
	// If these tokens are specified but cannot be read,
	// raise a fatal error.
	if dataManagerTokenFile != "" {
		if buf, err := ioutil.ReadFile(dataManagerTokenFile); err == nil {
			dataManagerToken = strings.TrimSpace(string(buf))
		} else {
			log.Fatalf("reading data manager token: %s\n", err)
		}
	}

	if neverDelete != true {
		log.Print("never-delete is not set. Warning: the relevant features in keepstore and data manager have not " +
			"been extensively tested. You should leave this option alone unless you can afford to lose data.")
	}

	if blobSigningKeyFile != "" {
		if buf, err := ioutil.ReadFile(blobSigningKeyFile); err == nil {
			PermissionSecret = bytes.TrimSpace(buf)
		} else {
			log.Fatalf("reading permission key: %s\n", err)
		}
	}

	blobSignatureTTL = time.Duration(permissionTTLSec) * time.Second

	if PermissionSecret == nil {
		if enforcePermissions {
			log.Fatal("-enforce-permissions requires a permission key")
		} else {
			log.Println("Running without a PermissionSecret. Block locators " +
				"returned by this server will not be signed, and will be rejected " +
				"by a server that enforces permissions.")
			log.Println("To fix this, use the -blob-signing-key-file flag " +
				"to specify the file containing the permission key.")
		}
	}

	if maxRequests <= 0 {
		maxRequests = maxBuffers * 2
		log.Printf("-max-requests <1 or not specified; defaulting to maxBuffers * 2 == %d", maxRequests)
	}

	// Start a round-robin VolumeManager with the volumes we have found.
	KeepVM = MakeRRVolumeManager(volumes)

	// Middleware stack: logger, maxRequests limiter, method handlers
	http.Handle("/", &LoggingRESTRouter{
		httpserver.NewRequestLimiter(maxRequests,
			MakeRESTRouter()),
	})

	// Set up a TCP listener.
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Pull queue and worker
	keepClient := &keepclient.KeepClient{
		Arvados:       &arvadosclient.ArvadosClient{},
		Want_replicas: 1,
		Client:        &http.Client{},
	}

	// Initialize the pullq and worker
	pullq = NewWorkQueue()
	go RunPullWorker(pullq, keepClient)

	// Initialize the trashq and worker
	trashq = NewWorkQueue()
	go RunTrashWorker(trashq)

	// Start emptyTrash goroutine
	doneEmptyingTrash := make(chan bool)
	go emptyTrash(doneEmptyingTrash, trashCheckInterval)

	// Shut down the server gracefully (by closing the listener)
	// if SIGTERM is received.
	term := make(chan os.Signal, 1)
	go func(sig <-chan os.Signal) {
		s := <-sig
		log.Println("caught signal:", s)
		doneEmptyingTrash <- true
		listener.Close()
	}(term)
	signal.Notify(term, syscall.SIGTERM)
	signal.Notify(term, syscall.SIGINT)

	log.Println("listening at", listen)
	srv := &http.Server{Addr: listen}
	srv.Serve(listener)
}

// At every trashCheckInterval tick, invoke EmptyTrash on all volumes.
func emptyTrash(doneEmptyingTrash chan bool, trashCheckInterval time.Duration) {
	ticker := time.NewTicker(trashCheckInterval)

	for {
		select {
		case <-ticker.C:
			for _, v := range volumes {
				if v.Writable() {
					v.EmptyTrash()
				}
			}
		case <-doneEmptyingTrash:
			ticker.Stop()
			return
		}
	}
}
