package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// Config specifies site configuration, like API credentials and the
// choice of which servers are to be balanced.
//
// Config is loaded from a JSON config file (see usage()).
type Config struct {
	// Arvados API endpoint and credentials.
	Client arvados.Client

	// List of service types (e.g., "disk") to balance.
	KeepServiceTypes []string

	KeepServiceList arvados.KeepServiceList

	// How often to check
	RunPeriod arvados.Duration

	// Number of collections to request in each API call
	CollectionBatchSize int

	// Max collections to buffer in memory (bigger values consume
	// more memory, but can reduce store-and-forward latency when
	// fetching pages)
	CollectionBuffers int
}

// RunOptions controls runtime behavior. The flags/options that belong
// here are the ones that are useful for interactive use. For example,
// "CommitTrash" is a runtime option rather than a config item because
// it invokes a troubleshooting feature rather than expressing how
// balancing is meant to be done at a given site.
//
// RunOptions fields are controlled by command line flags.
type RunOptions struct {
	Once        bool
	CommitPulls bool
	CommitTrash bool
	Logger      *log.Logger
	Dumper      *log.Logger

	// SafeRendezvousState from the most recent balance operation,
	// or "" if unknown. If this changes from one run to the next,
	// we need to watch out for races. See
	// (*Balancer)ClearTrashLists.
	SafeRendezvousState string
}

var debugf = func(string, ...interface{}) {}

func main() {
	var config Config
	var runOptions RunOptions

	configPath := flag.String("config", "",
		"`path` of json configuration file")
	serviceListPath := flag.String("config.KeepServiceList", "",
		"`path` of json file with list of keep services to balance, as given by \"arv keep_service list\" "+
			"(default: config[\"KeepServiceList\"], or if none given, get all available services and filter by config[\"KeepServiceTypes\"])")
	flag.BoolVar(&runOptions.Once, "once", false,
		"balance once and then exit")
	flag.BoolVar(&runOptions.CommitPulls, "commit-pulls", false,
		"send pull requests (make more replicas of blocks that are underreplicated or are not in optimal rendezvous probe order)")
	flag.BoolVar(&runOptions.CommitTrash, "commit-trash", false,
		"send trash requests (delete unreferenced old blocks, and excess replicas of overreplicated blocks)")
	dumpFlag := flag.Bool("dump", false, "dump details for each block to stdout")
	debugFlag := flag.Bool("debug", false, "enable debug messages")
	flag.Usage = usage
	flag.Parse()

	if *configPath == "" {
		log.Fatal("You must specify a config file (see `keep-balance -help`)")
	}
	mustReadJSON(&config, *configPath)
	if *serviceListPath != "" {
		mustReadJSON(&config.KeepServiceList, *serviceListPath)
	}

	if *debugFlag {
		debugf = log.Printf
		if j, err := json.Marshal(config); err != nil {
			log.Fatal(err)
		} else {
			log.Printf("config is %s", j)
		}
	}
	if *dumpFlag {
		runOptions.Dumper = log.New(os.Stdout, "", log.LstdFlags)
	}
	err := CheckConfig(config, runOptions)
	if err != nil {
		// (don't run)
	} else if runOptions.Once {
		_, err = (&Balancer{}).Run(config, runOptions)
	} else {
		err = RunForever(config, runOptions, nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func mustReadJSON(dst interface{}, path string) {
	if buf, err := ioutil.ReadFile(path); err != nil {
		log.Fatalf("Reading %q: %v", path, err)
	} else if err = json.Unmarshal(buf, dst); err != nil {
		log.Fatalf("Decoding %q: %v", path, err)
	}
}

// RunForever runs forever, or (for testing purposes) until the given
// stop channel is ready to receive.
func RunForever(config Config, runOptions RunOptions, stop <-chan interface{}) error {
	if runOptions.Logger == nil {
		runOptions.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	logger := runOptions.Logger

	ticker := time.NewTicker(time.Duration(config.RunPeriod))

	// The unbuffered channel here means we only hear SIGUSR1 if
	// it arrives while we're waiting in select{}.
	sigUSR1 := make(chan os.Signal)
	signal.Notify(sigUSR1, syscall.SIGUSR1)

	logger.Printf("starting up: will scan every %v and on SIGUSR1", config.RunPeriod)

	for {
		if !runOptions.CommitPulls && !runOptions.CommitTrash {
			logger.Print("WARNING: Will scan periodically, but no changes will be committed.")
			logger.Print("=======  Consider using -commit-pulls and -commit-trash flags.")
		}

		bal := &Balancer{}
		var err error
		runOptions, err = bal.Run(config, runOptions)
		if err != nil {
			logger.Print("run failed: ", err)
		} else {
			logger.Print("run succeeded")
		}

		select {
		case <-stop:
			signal.Stop(sigUSR1)
			return nil
		case <-ticker.C:
			logger.Print("timer went off")
		case <-sigUSR1:
			logger.Print("received SIGUSR1, resetting timer")
			// Reset the timer so we don't start the N+1st
			// run too soon after the Nth run is triggered
			// by SIGUSR1.
			ticker.Stop()
			ticker = time.NewTicker(time.Duration(config.RunPeriod))
		}
		logger.Print("starting next run")
	}
}
