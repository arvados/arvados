package main

import (
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"io/ioutil"
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

	finishCommand := flags.String(
		"finish-command",
		"/usr/bin/crunch-finish-slurm.sh",
		"Command to run from strigger when job is finished")

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
	runQueuedContainers(*pollInterval, *priorityPollInterval, *crunchRunCommand, *finishCommand)

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
func runQueuedContainers(pollInterval, priorityPollInterval int, crunchRunCommand, finishCommand string) {
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)

	for {
		select {
		case <-ticker.C:
			dispatchSlurm(priorityPollInterval, crunchRunCommand, finishCommand)
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
func dispatchSlurm(priorityPollInterval int, crunchRunCommand, finishCommand string) {
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
		go run(containers.Items[i], crunchRunCommand, finishCommand, priorityPollInterval)
	}
}

func submit(container Container, crunchRunCommand string) (jobid string, submiterr error) {
	submiterr = nil

	defer func() {
		if submiterr != nil {
			// This really should be an "Error" state, see #8018
			updateErr := arv.Update("containers", container.UUID,
				arvadosclient.Dict{
					"container": arvadosclient.Dict{"state": "Complete"}},
				nil)
			if updateErr != nil {
				log.Printf("Error updating container state to 'Complete' for %v: %q", container.UUID, updateErr)
			}
		}
	}()

	cmd := exec.Command("sbatch", "--job-name="+container.UUID, "--share", "--parsable")
	stdinWriter, stdinerr := cmd.StdinPipe()
	if stdinerr != nil {
		submiterr = fmt.Errorf("Error creating stdin pipe %v: %q", container.UUID, stdinerr)
		return
	}

	stdoutReader, stdouterr := cmd.StdoutPipe()
	if stdouterr != nil {
		submiterr = fmt.Errorf("Error creating stdout pipe %v: %q", container.UUID, stdouterr)
		return
	}

	stderrReader, stderrerr := cmd.StderrPipe()
	if stderrerr != nil {
		submiterr = fmt.Errorf("Error creating stderr pipe %v: %q", container.UUID, stderrerr)
		return
	}

	err := cmd.Start()
	if err != nil {
		submiterr = fmt.Errorf("Error starting %v: %v", cmd.Args, err)
		return
	}

	stdoutchan := make(chan []byte)
	go func() {
		b, _ := ioutil.ReadAll(stdoutReader)
		stdoutchan <- b
		close(stdoutchan)
	}()

	stderrchan := make(chan []byte)
	go func() {
		b, _ := ioutil.ReadAll(stderrReader)
		stderrchan <- b
		close(stderrchan)
	}()

	fmt.Fprintf(stdinWriter, "#!/bin/sh\nexec '%s' '%s'\n", crunchRunCommand, container.UUID)
	stdinWriter.Close()

	err = cmd.Wait()

	stdoutmsg := <-stdoutchan
	stderrmsg := <-stderrchan

	if err != nil {
		submiterr = fmt.Errorf("Container submission failed %v: %v %v", cmd.Args, err, stderrmsg)
		return
	}

	jobid = string(stdoutmsg)

	return
}

func strigger(jobid, containerUUID, finishCommand string) {
	cmd := exec.Command("strigger", "--set", "--jobid="+jobid, "--fini", fmt.Sprintf("--program=%s", finishCommand))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("While setting up strigger: %v", err)
	}
}

// Run queued container:
// Set container state to locked (TBD)
// Run container using the given crunch-run command
// Set the container state to Running
// If the container priority becomes zero while crunch job is still running, terminate it.
func run(container Container, crunchRunCommand, finishCommand string, priorityPollInterval int) {

	jobid, err := submit(container, crunchRunCommand)
	if err != nil {
		log.Printf("Error queuing container run: %v", err)
		return
	}

	strigger(jobid, container.UUID, finishCommand)

	// Update container status to Running
	err = arv.Update("containers", container.UUID,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": "Running"}},
		nil)
	if err != nil {
		log.Printf("Error updating container state to 'Running' for %v: %q", container.UUID, err)
	}

	log.Printf("Submitted container run for %v", container.UUID)

	containerUUID := container.UUID

	// A goroutine to terminate the runner if container priority becomes zero
	priorityTicker := time.NewTicker(time.Duration(priorityPollInterval) * time.Second)
	go func() {
		for _ = range priorityTicker.C {
			var container Container
			err := arv.Get("containers", containerUUID, nil, &container)
			if err != nil {
				log.Printf("Error getting container info for %v: %q", container.UUID, err)
			} else {
				if container.Priority == 0 {
					log.Printf("Canceling container %v", container.UUID)
					priorityTicker.Stop()
					cancelcmd := exec.Command("scancel", "--name="+container.UUID)
					cancelcmd.Run()
				}
				if container.State == "Complete" {
					priorityTicker.Stop()
				}
			}
		}
	}()

}
