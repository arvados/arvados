package main

// Dispatcher service for Crunch that runs containers locally.

import (
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%q", err)
	}
}

var (
	runningCmds      map[string]*exec.Cmd
	runningCmdsMutex sync.Mutex
	waitGroup        sync.WaitGroup
	crunchRunCommand *string
)

func doMain() error {
	flags := flag.NewFlagSet("crunch-dispatch-local", flag.ExitOnError)

	pollInterval := flags.Int(
		"poll-interval",
		10,
		"Interval in seconds to poll for queued containers")

	crunchRunCommand = flags.String(
		"crunch-run-command",
		"/usr/bin/crunch-run",
		"Crunch command to run container")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	runningCmds = make(map[string]*exec.Cmd)

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Printf("Error making Arvados client: %v", err)
		return err
	}
	arv.Retries = 25

	dispatcher := dispatch.Dispatcher{
		Arv:            arv,
		RunContainer:   run,
		PollInterval:   time.Duration(*pollInterval) * time.Second,
		DoneProcessing: make(chan struct{})}

	err = dispatcher.RunDispatcher()
	if err != nil {
		return err
	}

	runningCmdsMutex.Lock()
	// Finished dispatching; interrupt any crunch jobs that are still running
	for _, cmd := range runningCmds {
		cmd.Process.Signal(os.Interrupt)
	}
	runningCmdsMutex.Unlock()

	// Wait for all running crunch jobs to complete / terminate
	waitGroup.Wait()

	return nil
}

func startFunc(container arvados.Container, cmd *exec.Cmd) error {
	return cmd.Start()
}

var startCmd = startFunc

// Run a container.
//
// If the container is Locked, start a new crunch-run process and wait until
// crunch-run completes.  If the priority is set to zero, set an interrupt
// signal to the crunch-run process.
//
// If the container is in any other state, or is not Complete/Cancelled after
// crunch-run terminates, mark the container as Cancelled.
func run(dispatcher *dispatch.Dispatcher,
	container arvados.Container,
	status chan arvados.Container) {

	uuid := container.UUID

	if container.State == dispatch.Locked {
		waitGroup.Add(1)

		cmd := exec.Command(*crunchRunCommand, uuid)
		cmd.Stdin = nil
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stderr

		log.Printf("Starting container %v", uuid)

		// Add this crunch job to the list of runningCmds only if we
		// succeed in starting crunch-run.

		runningCmdsMutex.Lock()
		if err := startCmd(container, cmd); err != nil {
			runningCmdsMutex.Unlock()
			log.Printf("Error starting %v for %v: %q", *crunchRunCommand, uuid, err)
			dispatcher.UpdateState(uuid, dispatch.Cancelled)
		} else {
			runningCmds[uuid] = cmd
			runningCmdsMutex.Unlock()

			// Need to wait for crunch-run to exit
			done := make(chan struct{})

			go func() {
				if _, err := cmd.Process.Wait(); err != nil {
					log.Printf("Error while waiting for crunch job to finish for %v: %q", uuid, err)
				}
				log.Printf("sending done")
				done <- struct{}{}
			}()

		Loop:
			for {
				select {
				case <-done:
					break Loop
				case c := <-status:
					// Interrupt the child process if priority changes to 0
					if (c.State == dispatch.Locked || c.State == dispatch.Running) && c.Priority == 0 {
						log.Printf("Sending SIGINT to pid %d to cancel container %v", cmd.Process.Pid, uuid)
						cmd.Process.Signal(os.Interrupt)
					}
				}
			}
			close(done)

			log.Printf("Finished container run for %v", uuid)

			// Remove the crunch job from runningCmds
			runningCmdsMutex.Lock()
			delete(runningCmds, uuid)
			runningCmdsMutex.Unlock()
		}
		waitGroup.Done()
	}

	// If the container is not finalized, then change it to "Cancelled".
	err := dispatcher.Arv.Get("containers", uuid, nil, &container)
	if err != nil {
		log.Printf("Error getting final container state: %v", err)
	}
	if container.LockedByUUID == dispatcher.Auth.UUID &&
		(container.State == dispatch.Locked || container.State == dispatch.Running) {
		log.Printf("After %s process termination, container state for %v is %q.  Updating it to %q",
			*crunchRunCommand, container.State, uuid, dispatch.Cancelled)
		dispatcher.UpdateState(uuid, dispatch.Cancelled)
	}

	// drain any subsequent status changes
	for range status {
	}

	log.Printf("Finalized container %v", uuid)
}
