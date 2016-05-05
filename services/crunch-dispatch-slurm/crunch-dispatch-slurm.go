package main

import (
	"bufio"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
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

type apiClientAuthorization struct {
	UUID     string `json:"uuid"`
	APIToken string `json:"api_token"`
}

type apiClientAuthorizationList struct {
	Items []apiClientAuthorization `json:"items"`
}

// Poll for queued containers using pollInterval.
// Invoke dispatchSlurm for each ticker cycle, which will run all the queued containers.
//
// Any errors encountered are logged but the program would continue to run (not exit).
// This is because, once one or more crunch jobs are running,
// we would need to wait for them complete.
func runQueuedContainers(pollInterval, priorityPollInterval int, crunchRunCommand, finishCommand string) {
	var authList apiClientAuthorizationList
	err := arv.List("api_client_authorizations", map[string]interface{}{
		"filters": [][]interface{}{{"api_token", "=", arv.ApiToken}},
	}, &authList)
	if err != nil || len(authList.Items) != 1 {
		log.Printf("Error getting my token UUID: %v (%d)", err, len(authList.Items))
		return
	}
	auth := authList.Items[0]

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	for {
		select {
		case <-ticker.C:
			dispatchSlurm(auth, time.Duration(priorityPollInterval)*time.Second, crunchRunCommand, finishCommand)
		case <-doneProcessing:
			ticker.Stop()
			return
		}
	}
}

// Container data
type Container struct {
	UUID               string           `json:"uuid"`
	State              string           `json:"state"`
	Priority           int              `json:"priority"`
	RuntimeConstraints map[string]int64 `json:"runtime_constraints"`
	LockedByUUID       string           `json:"locked_by_uuid"`
}

// ContainerList is a list of the containers from api
type ContainerList struct {
	Items []Container `json:"items"`
}

// Get the list of queued containers from API server and invoke run
// for each container.
func dispatchSlurm(auth apiClientAuthorization, pollInterval time.Duration, crunchRunCommand, finishCommand string) {
	params := arvadosclient.Dict{
		"filters": [][]interface{}{{"state", "in", []string{"Queued", "Locked"}}},
	}

	var containers ContainerList
	err := arv.List("containers", params, &containers)
	if err != nil {
		log.Printf("Error getting list of queued containers: %q", err)
		return
	}

	for _, container := range containers.Items {
		if container.State == "Locked" {
			if container.LockedByUUID != auth.UUID {
				// Locked by a different dispatcher
				continue
			} else if checkMine(container.UUID) {
				// I already have a goroutine running
				// for this container: it just hasn't
				// gotten past Locked state yet.
				continue
			}
			log.Printf("WARNING: found container %s already locked by my token %s, but I didn't submit it. "+
				"Assuming it was left behind by a previous dispatch process, and waiting for it to finish.",
				container.UUID, auth.UUID)
			setMine(container.UUID, true)
			go func() {
				waitContainer(container, pollInterval)
				setMine(container.UUID, false)
			}()
		}
		go run(container, crunchRunCommand, finishCommand, pollInterval)
	}
}

// sbatchCmd
func sbatchFunc(container Container) *exec.Cmd {
	memPerCPU := math.Ceil((float64(container.RuntimeConstraints["ram"])) / (float64(container.RuntimeConstraints["vcpus"] * 1048576)))
	return exec.Command("sbatch", "--share", "--parsable",
		"--job-name="+container.UUID,
		"--mem-per-cpu="+strconv.Itoa(int(memPerCPU)),
		"--cpus-per-task="+strconv.Itoa(int(container.RuntimeConstraints["vcpus"])))
}

var sbatchCmd = sbatchFunc

// striggerCmd
func striggerFunc(jobid, containerUUID, finishCommand, apiHost, apiToken, apiInsecure string) *exec.Cmd {
	return exec.Command("strigger", "--set", "--jobid="+jobid, "--fini",
		fmt.Sprintf("--program=%s %s %s %s %s", finishCommand, apiHost, apiToken, apiInsecure, containerUUID))
}

var striggerCmd = striggerFunc

// Submit job to slurm using sbatch.
func submit(container Container, crunchRunCommand string) (jobid string, submitErr error) {
	submitErr = nil

	defer func() {
		// If we didn't get as far as submitting a slurm job,
		// unlock the container and return it to the queue.
		if submitErr == nil {
			// OK, no cleanup needed
			return
		}
		err := arv.Update("containers", container.UUID,
			arvadosclient.Dict{
				"container": arvadosclient.Dict{"state": "Queued"}},
			nil)
		if err != nil {
			log.Printf("Error unlocking container %s: %v", container.UUID, err)
		}
	}()

	// Create the command and attach to stdin/stdout
	cmd := sbatchCmd(container)
	stdinWriter, stdinerr := cmd.StdinPipe()
	if stdinerr != nil {
		submitErr = fmt.Errorf("Error creating stdin pipe %v: %q", container.UUID, stdinerr)
		return
	}

	stdoutReader, stdoutErr := cmd.StdoutPipe()
	if stdoutErr != nil {
		submitErr = fmt.Errorf("Error creating stdout pipe %v: %q", container.UUID, stdoutErr)
		return
	}

	stderrReader, stderrErr := cmd.StderrPipe()
	if stderrErr != nil {
		submitErr = fmt.Errorf("Error creating stderr pipe %v: %q", container.UUID, stderrErr)
		return
	}

	err := cmd.Start()
	if err != nil {
		submitErr = fmt.Errorf("Error starting %v: %v", cmd.Args, err)
		return
	}

	stdoutChan := make(chan []byte)
	go func() {
		b, _ := ioutil.ReadAll(stdoutReader)
		stdoutReader.Close()
		stdoutChan <- b
		close(stdoutChan)
	}()

	stderrChan := make(chan []byte)
	go func() {
		b, _ := ioutil.ReadAll(stderrReader)
		stderrReader.Close()
		stderrChan <- b
		close(stderrChan)
	}()

	// Send a tiny script on stdin to execute the crunch-run command
	// slurm actually enforces that this must be a #! script
	fmt.Fprintf(stdinWriter, "#!/bin/sh\nexec '%s' '%s'\n", crunchRunCommand, container.UUID)
	stdinWriter.Close()

	err = cmd.Wait()

	stdoutMsg := <-stdoutChan
	stderrmsg := <-stderrChan

	if err != nil {
		submitErr = fmt.Errorf("Container submission failed %v: %v %v", cmd.Args, err, stderrmsg)
		return
	}

	// If everything worked out, got the jobid on stdout
	jobid = string(stdoutMsg)

	return
}

// finalizeRecordOnFinish uses 'strigger' command to register a script that will run on
// the slurm controller when the job finishes.
func finalizeRecordOnFinish(jobid, containerUUID, finishCommand, apiHost, apiToken, apiInsecure string) {
	cmd := striggerCmd(jobid, containerUUID, finishCommand, apiHost, apiToken, apiInsecure)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("While setting up strigger: %v", err)
		// BUG: we drop the error here and forget about it. A
		// human has to notice the container is stuck in
		// Running state, and fix it manually.
	}
}

// Run a queued container: [1] Set container state to locked. [2]
// Execute crunch-run as a slurm batch job. [3] waitContainer().
func run(container Container, crunchRunCommand, finishCommand string, pollInterval time.Duration) {
	setMine(container.UUID, true)
	defer setMine(container.UUID, false)

	// Update container status to Locked. This will fail if
	// another dispatcher (token) has already locked it. It will
	// succeed if *this* dispatcher has already locked it.
	err := arv.Update("containers", container.UUID,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": "Locked"}},
		nil)
	if err != nil {
		log.Printf("Error updating container state to 'Locked' for %v: %q", container.UUID, err)
		return
	}

	log.Printf("About to submit queued container %v", container.UUID)

	jobid, err := submit(container, crunchRunCommand)
	if err != nil {
		log.Printf("Error submitting container %s to slurm: %v", container.UUID, err)
		return
	}

	insecure := "0"
	if arv.ApiInsecure {
		insecure = "1"
	}
	finalizeRecordOnFinish(jobid, container.UUID, finishCommand, arv.ApiServer, arv.ApiToken, insecure)

	// Update container status to Running. This will fail if
	// another dispatcher (token) has already locked it. It will
	// succeed if *this* dispatcher has already locked it.
	err = arv.Update("containers", container.UUID,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": "Running"}},
		nil)
	if err != nil {
		log.Printf("Error updating container state to 'Running' for %v: %q", container.UUID, err)
	}
	log.Printf("Submitted container %v to slurm", container.UUID)
	waitContainer(container, pollInterval)
}

// Wait for a container to finish. Cancel the slurm job if the
// container priority changes to zero before it ends.
func waitContainer(container Container, pollInterval time.Duration) {
	log.Printf("Monitoring container %v started", container.UUID)
	defer log.Printf("Monitoring container %v finished", container.UUID)

	pollTicker := time.NewTicker(pollInterval)
	defer pollTicker.Stop()
	for _ = range pollTicker.C {
		var updated Container
		err := arv.Get("containers", container.UUID, nil, &updated)
		if err != nil {
			log.Printf("Error getting container %s: %q", container.UUID, err)
			continue
		}
		if updated.State == "Complete" || updated.State == "Cancelled" {
			return
		}
		if updated.Priority != 0 {
			continue
		}

		// Priority is zero, but state is Running or Locked
		log.Printf("Canceling container %s", container.UUID)

		err = exec.Command("scancel", "--name="+container.UUID).Run()
		if err != nil {
			log.Printf("Error stopping container %s with scancel: %v", container.UUID, err)
			if inQ, err := checkSqueue(container.UUID); err != nil {
				log.Printf("Error running squeue: %v", err)
				continue
			} else if inQ {
				log.Printf("Container %s is still in squeue; will retry", container.UUID)
				continue
			}
		}

		err = arv.Update("containers", container.UUID,
			arvadosclient.Dict{
				"container": arvadosclient.Dict{"state": "Cancelled"}},
			nil)
		if err != nil {
			log.Printf("Error updating state for container %s: %s", container.UUID, err)
			continue
		}

		return
	}
}

func checkSqueue(uuid string) (bool, error) {
	cmd := exec.Command("squeue", "--format=%j")
	sq, err := cmd.StdoutPipe()
	if err != nil {
		return false, err
	}
	cmd.Start()
	defer cmd.Wait()
	scanner := bufio.NewScanner(sq)
	found := false
	for scanner.Scan() {
		if scanner.Text() == uuid {
			found = true
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return found, nil
}

var mineMutex sync.RWMutex
var mineMap = make(map[string]bool)

// Goroutine-safely add/remove uuid to the set of "my" containers,
// i.e., ones for which this process has a goroutine running.
func setMine(uuid string, t bool) {
	mineMutex.Lock()
	if t {
		mineMap[uuid] = true
	} else {
		delete(mineMap, uuid)
	}
	mineMutex.Unlock()
}

// Check whether there is already a goroutine running for this
// container.
func checkMine(uuid string) bool {
	mineMutex.RLocker().Lock()
	defer mineMutex.RLocker().Unlock()
	return mineMap[uuid]
}
