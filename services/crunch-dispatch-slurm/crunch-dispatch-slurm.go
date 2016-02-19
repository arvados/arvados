package main

import (
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%q", err)
	}
}

var (
	arv              arvadosclient.ArvadosClient
	runningCmds      map[string]*exec.Cmd
	runningCmdsMutex sync.Mutex
	waitGroup        sync.WaitGroup
	doneProcessing   chan bool
	sigChan          chan os.Signal
)

func doMain() error {
	flags := flag.NewFlagSet("crunch-dispatch-slurm", flag.ExitOnError)

	pollInterval := flags.Int(
		"poll-interval",
		10,
		"Interval in seconds to poll for queued containers")

	priorityPollInterval := flags.Int(
		"container-priority-poll-interval",
		60,
		"Interval in seconds to check priority of a dispatched container")

	crunchRunCommand := flags.String(
		"crunch-run-command",
		"/usr/bin/crunch-run",
		"Crunch command to run container")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	var err error
	arv, err = arvadosclient.MakeArvadosClient()
	if err != nil {
		return err
	}

	// Channel to terminate
	doneProcessing = make(chan bool)

	// Graceful shutdown
	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func(sig <-chan os.Signal) {
		for sig := range sig {
			log.Printf("Caught signal: %v", sig)
			doneProcessing <- true
		}
	}(sigChan)

	// Run all queued containers
	runQueuedContainers(*pollInterval, *priorityPollInterval, *crunchRunCommand)

	// Wait for all running crunch jobs to complete / terminate
	waitGroup.Wait()

	return nil
}

// Poll for queued containers using pollInterval.
// Invoke dispatchLocal for each ticker cycle, which will run all the queued containers.
//
// Any errors encountered are logged but the program would continue to run (not exit).
// This is because, once one or more crunch jobs are running,
// we would need to wait for them complete.
func runQueuedContainers(pollInterval, priorityPollInterval int, crunchRunCommand string) {
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)

	for {
		select {
		case <-ticker.C:
			dispatchSlurm(priorityPollInterval, crunchRunCommand)
		case <-doneProcessing:
			ticker.Stop()
			return
		}
	}
}

// Container data
type Container struct {
	UUID     string `json:"uuid"`
	State    string `json:"state"`
	Priority int    `json:"priority"`
}

// ContainerList is a list of the containers from api
type ContainerList struct {
	Items []Container `json:"items"`
}

// Get the list of queued containers from API server and invoke run for each container.
func dispatchSlurm(priorityPollInterval int, crunchRunCommand string) {
	params := arvadosclient.Dict{
		"filters": [][]string{[]string{"state", "=", "Queued"}},
	}

	var containers ContainerList
	err := arv.List("containers", params, &containers)
	if err != nil {
		log.Printf("Error getting list of queued containers: %q", err)
		return
	}

	for i := 0; i < len(containers.Items); i++ {
		log.Printf("About to submit queued container %v", containers.Items[i].UUID)
		// Run the container
		go run(containers.Items[i].UUID, crunchRunCommand, priorityPollInterval)
	}
}

// Run queued container:
// Set container state to locked (TBD)
// Run container using the given crunch-run command
// Set the container state to Running
// If the container priority becomes zero while crunch job is still running, terminate it.
func run(uuid string, crunchRunCommand string, priorityPollInterval int) {
	stdinReader, stdinWriter := io.Pipe()

	cmd := exec.Command("sbatch", "--job-name="+uuid)
	cmd.Stdin = stdinReader
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("Error running container for %v: %q", uuid, err)
		return
	}

	fmt.Fprintf(stdinWriter, "#!/bin/sh\nexec %s %s\n", crunchRunCommand, uuid)

	stdinWriter.Close()
	cmd.Wait()

	// Update container status to Running
	err := arv.Update("containers", uuid,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": "Running"}},
		nil)
	if err != nil {
		log.Printf("Error updating container state to 'Running' for %v: %q", uuid, err)
	}

	log.Printf("Submitted container run for %v", uuid)

	// A goroutine to terminate the runner if container priority becomes zero
	priorityTicker := time.NewTicker(time.Duration(priorityPollInterval) * time.Second)
	go func() {
		for _ = range priorityTicker.C {
			var container Container
			err := arv.Get("containers", uuid, nil, &container)
			if err != nil {
				log.Printf("Error getting container info for %v: %q", uuid, err)
			} else {
				if container.Priority == 0 {
					priorityTicker.Stop()
					cancelcmd := exec.Command("scancel", "--name="+uuid)
					cancelcmd.Run()
				}
			}
		}
	}()

}
