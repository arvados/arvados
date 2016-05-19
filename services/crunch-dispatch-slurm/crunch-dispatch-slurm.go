package main

// Dispatcher service for Crunch that submits containers to the slurm queue.

import (
	"bufio"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%q", err)
	}
}

var (
	crunchRunCommand *string
	finishCommand    *string
)

func doMain() error {
	flags := flag.NewFlagSet("crunch-dispatch-slurm", flag.ExitOnError)

	pollInterval := flags.Int(
		"poll-interval",
		10,
		"Interval in seconds to poll for queued containers")

	crunchRunCommand = flags.String(
		"crunch-run-command",
		"/usr/bin/crunch-run",
		"Crunch command to run container")

	finishCommand = flags.String(
		"finish-command",
		"/usr/bin/crunch-finish-slurm.sh",
		"Command to run from strigger when job is finished")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

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

	return nil
}

// sbatchCmd
func sbatchFunc(container dispatch.Container) *exec.Cmd {
	memPerCPU := math.Ceil((float64(container.RuntimeConstraints["ram"])) / (float64(container.RuntimeConstraints["vcpus"] * 1048576)))
	return exec.Command("sbatch", "--share", "--parsable",
		fmt.Sprintf("--job-name=%s", container.UUID),
		fmt.Sprintf("--mem-per-cpu=%d", int(memPerCPU)),
		fmt.Sprintf("--cpus-per-task=%d", int(container.RuntimeConstraints["vcpus"])),
		fmt.Sprintf("--priority=%d", container.Priority))
}

// striggerCmd
func striggerFunc(jobid, containerUUID, finishCommand, apiHost, apiToken, apiInsecure string) *exec.Cmd {
	return exec.Command("strigger", "--set", "--jobid="+jobid, "--fini",
		fmt.Sprintf("--program=%s %s %s %s %s", finishCommand, apiHost, apiToken, apiInsecure, containerUUID))
}

// squeueFunc
func squeueFunc() *exec.Cmd {
	return exec.Command("squeue", "--format=%j")
}

// Wrap these so that they can be overridden by tests
var striggerCmd = striggerFunc
var sbatchCmd = sbatchFunc
var squeueCmd = squeueFunc

// Submit job to slurm using sbatch.
func submit(dispatcher *dispatch.Dispatcher,
	container dispatch.Container, crunchRunCommand string) (jobid string, submitErr error) {
	submitErr = nil

	defer func() {
		// If we didn't get as far as submitting a slurm job,
		// unlock the container and return it to the queue.
		if submitErr == nil {
			// OK, no cleanup needed
			return
		}
		err := dispatcher.Arv.Update("containers", container.UUID,
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
	}()

	stderrChan := make(chan []byte)
	go func() {
		b, _ := ioutil.ReadAll(stderrReader)
		stderrReader.Close()
		stderrChan <- b
	}()

	// Send a tiny script on stdin to execute the crunch-run command
	// slurm actually enforces that this must be a #! script
	fmt.Fprintf(stdinWriter, "#!/bin/sh\nexec '%s' '%s'\n", crunchRunCommand, container.UUID)
	stdinWriter.Close()

	err = cmd.Wait()

	stdoutMsg := <-stdoutChan
	stderrmsg := <-stderrChan

	close(stdoutChan)
	close(stderrChan)

	if err != nil {
		submitErr = fmt.Errorf("Container submission failed %v: %v %v", cmd.Args, err, stderrmsg)
		return
	}

	// If everything worked out, got the jobid on stdout
	jobid = strings.TrimSpace(string(stdoutMsg))

	return
}

// finalizeRecordOnFinish uses 'strigger' command to register a script that will run on
// the slurm controller when the job finishes.
func finalizeRecordOnFinish(jobid, containerUUID, finishCommand string, arv arvadosclient.ArvadosClient) {
	insecure := "0"
	if arv.ApiInsecure {
		insecure = "1"
	}
	cmd := striggerCmd(jobid, containerUUID, finishCommand, arv.ApiServer, arv.ApiToken, insecure)
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

func checkSqueue(uuid string) (bool, error) {
	cmd := squeueCmd()
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

// Run or monitor a container.
//
// If the container is marked as Locked, check if it is already in the slurm
// queue.  If not, submit it.
//
// If the container is marked as Running, check if it is in the slurm queue.
// If not, mark it as Cancelled.
//
// Monitor status updates.  If the priority changes to zero, cancel the
// container using scancel.
func run(dispatcher *dispatch.Dispatcher,
	container dispatch.Container,
	status chan dispatch.Container) {

	uuid := container.UUID

	if container.State == dispatch.Locked {
		if inQ, err := checkSqueue(container.UUID); err != nil {
			log.Printf("Error running squeue: %v", err)
			dispatcher.UpdateState(container.UUID, dispatch.Cancelled)
		} else if !inQ {
			log.Printf("About to submit queued container %v", container.UUID)

			jobid, err := submit(dispatcher, container, *crunchRunCommand)
			if err != nil {
				log.Printf("Error submitting container %s to slurm: %v", container.UUID, err)
			} else {
				finalizeRecordOnFinish(jobid, container.UUID, *finishCommand, dispatcher.Arv)
			}
		}
	} else if container.State == dispatch.Running {
		if inQ, err := checkSqueue(container.UUID); err != nil {
			log.Printf("Error running squeue: %v", err)
			dispatcher.UpdateState(container.UUID, dispatch.Cancelled)
		} else if !inQ {
			log.Printf("Container %s in Running state but not in slurm queue, marking Cancelled.", container.UUID)
			dispatcher.UpdateState(container.UUID, dispatch.Cancelled)
		}
	}

	log.Printf("Monitoring container %v started", uuid)

	for container = range status {
		if (container.State == dispatch.Locked || container.State == dispatch.Running) && container.Priority == 0 {
			log.Printf("Canceling container %s", container.UUID)

			err := exec.Command("scancel", "--name="+container.UUID).Run()
			if err != nil {
				log.Printf("Error stopping container %s with scancel: %v", container.UUID, err)
				if inQ, err := checkSqueue(container.UUID); err != nil {
					log.Printf("Error running squeue: %v", err)
					continue
				} else if inQ {
					log.Printf("Container %s is still in squeue after scancel.", container.UUID)
					continue
				}
			}

			err = dispatcher.UpdateState(container.UUID, dispatch.Cancelled)
		}
	}

	log.Printf("Monitoring container %v finished", uuid)
}
