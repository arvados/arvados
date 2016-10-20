package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"github.com/coreos/go-systemd/daemon"
	"github.com/ghodss/yaml"
)

// A Keep "block" is 64MB.
const BlockSize = 64 * 1024 * 1024

// A Keep volume must have at least MinFreeKilobytes available
// in order to permit writes.
const MinFreeKilobytes = BlockSize / 1024

// ProcMounts /proc/mounts
var ProcMounts = "/proc/mounts"

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

func main() {
	deprecated.beforeFlagParse(theConfig)

	dumpConfig := flag.Bool("dump-config", false, "write current configuration to stdout and exit (useful for migrating from command line flags to config file)")

	defaultConfigPath := "/etc/arvados/keepstore/keepstore.yml"
	var configPath string
	flag.StringVar(
		&configPath,
		"config",
		defaultConfigPath,
		"YAML or JSON configuration file `path`")
	flag.Usage = usage
	flag.Parse()

	deprecated.afterFlagParse(theConfig)

	err := config.LoadFile(theConfig, configPath)
	if err != nil && (!os.IsNotExist(err) || configPath != defaultConfigPath) {
		log.Fatal(err)
	}

	if *dumpConfig {
		y, err := yaml.Marshal(theConfig)
		if err != nil {
			log.Fatal(err)
		}
		os.Stdout.Write(y)
		os.Exit(0)
	}

	err = theConfig.Start()

	if pidfile := theConfig.PIDFile; pidfile != "" {
		f, err := os.OpenFile(pidfile, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			log.Fatalf("open pidfile (%s): %s", pidfile, err)
		}
		defer f.Close()
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			log.Fatalf("flock pidfile (%s): %s", pidfile, err)
		}
		defer os.Remove(pidfile)
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
	}

	log.Println("keepstore starting, pid", os.Getpid())
	defer log.Println("keepstore exiting, pid", os.Getpid())

	// Start a round-robin VolumeManager with the volumes we have found.
	KeepVM = MakeRRVolumeManager(theConfig.Volumes)

	// Middleware stack: logger, MaxRequests limiter, method handlers
	http.Handle("/", &LoggingRESTRouter{
		httpserver.NewRequestLimiter(theConfig.MaxRequests,
			MakeRESTRouter()),
	})

	// Set up a TCP listener.
	listener, err := net.Listen("tcp", theConfig.Listen)
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
	go emptyTrash(doneEmptyingTrash, theConfig.TrashCheckInterval.Duration())

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

	if _, err := daemon.SdNotify("READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}
	log.Println("listening at", listener.Addr())
	srv := &http.Server{}
	srv.Serve(listener)
}

// Periodically (once per interval) invoke EmptyTrash on all volumes.
func emptyTrash(done <-chan bool, interval time.Duration) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			for _, v := range theConfig.Volumes {
				if v.Writable() {
					v.EmptyTrash()
				}
			}
		case <-done:
			ticker.Stop()
			return
		}
	}
}
