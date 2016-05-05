package main

import (
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
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
	flags := flag.NewFlagSet("crunch-dispatch-local", flag.ExitOnError)

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

	// Map of running crunch jobs
	runningCmds = make(map[string]*exec.Cmd)

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
	runQueuedContainers(time.Duration(*pollInterval)*time.Second, time.Duration(*priorityPollInterval)*time.Second, *crunchRunCommand)

	// Finished dispatching; interrupt any crunch jobs that are still running
	for _, cmd := range runningCmds {
		cmd.Process.Signal(os.Interrupt)
	}

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
func runQueuedContainers(pollInterval, priorityPollInterval time.Duration, crunchRunCommand string) {
	ticker := time.NewTicker(pollInterval)

	for {
		select {
		case <-ticker.C:
			dispatchLocal(priorityPollInterval, crunchRunCommand)
		case <-doneProcessing:
			ticker.Stop()
			return
		}
	}
}

// Container data
type Container struct {
	UUID         string `json:"uuid"`
	State        string `json:"state"`
	Priority     int    `json:"priority"`
	LockedByUUID string `json:"locked_by_uuid"`
}

// ContainerList is a list of the containers from api
type ContainerList struct {
	Items []Container `json:"items"`
}

// Get the list of queued containers from API server and invoke run for each container.
func dispatchLocal(pollInterval time.Duration, crunchRunCommand string) {
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
		log.Printf("About to run queued container %v", containers.Items[i].UUID)
		// Run the container
		go run(containers.Items[i].UUID, crunchRunCommand, pollInterval)
	}
}

func updateState(uuid, newState string) error {
	err := arv.Update("containers", uuid,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": newState}},
		nil)
	if err != nil {
		log.Printf("Error updating container %s to '%s' state: %q", uuid, newState, err)
	}
	return err
}

// Run queued container:
// Set container state to Locked
// Run container using the given crunch-run command
// Set the container state to Running
// If the container priority becomes zero while crunch job is still running, terminate it.
func run(uuid string, crunchRunCommand string, pollInterval time.Duration) {
	if err := updateState(uuid, "Locked"); err != nil {
		return
	}

	cmd := exec.Command(crunchRunCommand, uuid)
	cmd.Stdin = nil
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr

	// Add this crunch job to the list of runningCmds only if we
	// succeed in starting crunch-run.
	runningCmdsMutex.Lock()
	if err := cmd.Start(); err != nil {
		log.Printf("Error starting crunch-run for %v: %q", uuid, err)
		runningCmdsMutex.Unlock()
		updateState(uuid, "Queued")
		return
	}
	runningCmds[uuid] = cmd
	runningCmdsMutex.Unlock()

	defer func() {
		setFinalState(uuid)

		// Remove the crunch job from runningCmds
		runningCmdsMutex.Lock()
		delete(runningCmds, uuid)
		runningCmdsMutex.Unlock()
	}()

	log.Printf("Starting container %v", uuid)

	// Add this crunch job to waitGroup
	waitGroup.Add(1)
	defer waitGroup.Done()

	updateState(uuid, "Running")

	cmdExited := make(chan struct{})

	// Kill the child process if container priority changes to zero
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-cmdExited:
				return
			case <-ticker.C:
			}
			var container Container
			err := arv.Get("containers", uuid, nil, &container)
			if err != nil {
				log.Printf("Error getting container %v: %q", uuid, err)
				continue
			}
			if container.Priority == 0 {
				log.Printf("Sending SIGINT to pid %d to cancel container %v", cmd.Process.Pid, uuid)
				cmd.Process.Signal(os.Interrupt)
			}
		}
	}()

	// Wait for crunch-run to exit
	if _, err := cmd.Process.Wait(); err != nil {
		log.Printf("Error while waiting for crunch job to finish for %v: %q", uuid, err)
	}
	close(cmdExited)

	log.Printf("Finished container run for %v", uuid)
}

func setFinalState(uuid string) {
	// The container state should now be 'Complete' if everything
	// went well. If it started but crunch-run didn't change its
	// final state to 'Running', fix that now. If it never even
	// started, cancel it as unrunnable. (TODO: Requeue instead,
	// and fix tests so they can tell something happened even if
	// the final state is Queued.)
	var container Container
	err := arv.Get("containers", uuid, nil, &container)
	if err != nil {
		log.Printf("Error getting final container state: %v", err)
	}
	fixState := map[string]string{
		"Running": "Complete",
		"Locked": "Cancelled",
	}
	if newState, ok := fixState[container.State]; ok {
		log.Printf("After crunch-run process termination, the state is still '%s' for %v. Updating it to '%s'", container.State, uuid, newState)
		updateState(uuid, newState)
	}
}
