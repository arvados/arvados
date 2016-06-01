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
	"sync"
	"time"
)

type Squeue struct {
	sync.Mutex
	squeueContents []string
	SqueueDone     chan struct{}
}

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%q", err)
	}
}

var (
	crunchRunCommand *string
	squeueUpdater    Squeue
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

	squeueUpdater.SqueueDone = make(chan struct{})
	go squeueUpdater.SyncSqueue(time.Duration(*pollInterval) * time.Second)

	err = dispatcher.RunDispatcher()
	if err != nil {
		return err
	}

	squeueUpdater.SqueueDone <- struct{}{}
	close(squeueUpdater.SqueueDone)

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

// squeueFunc
func squeueFunc() *exec.Cmd {
	return exec.Command("squeue", "--format=%j")
}

// Wrap these so that they can be overridden by tests
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

func (squeue *Squeue) runSqueue() ([]string, error) {
	var newSqueueContents []string

	cmd := squeueCmd()
	sq, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Start()
	scanner := bufio.NewScanner(sq)
	for scanner.Scan() {
		newSqueueContents = append(newSqueueContents, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		cmd.Wait()
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return newSqueueContents, nil
}

func (squeue *Squeue) CheckSqueue(uuid string, check bool) (bool, error) {
	if check {
		n, err := squeue.runSqueue()
		if err != nil {
			return false, err
		}
		squeue.Lock()
		squeue.squeueContents = n
		squeue.Unlock()
	}

	if uuid != "" {
		squeue.Lock()
		defer squeue.Unlock()
		for _, k := range squeue.squeueContents {
			if k == uuid {
				return true, nil
			}
		}
	}
	return false, nil
}

func (squeue *Squeue) SyncSqueue(pollInterval time.Duration) {
	// TODO: considering using "squeue -i" instead of polling squeue.
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-squeueUpdater.SqueueDone:
			return
		case <-ticker.C:
			squeue.CheckSqueue("", true)
		}
	}
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
		if inQ, err := squeueUpdater.CheckSqueue(container.UUID, true); err != nil {
			// maybe squeue is broken, put it back in the queue
			log.Printf("Error running squeue: %v", err)
			dispatcher.UpdateState(container.UUID, dispatch.Queued)
		} else if !inQ {
			log.Printf("About to submit queued container %v", container.UUID)

			if _, err := submit(dispatcher, container, *crunchRunCommand); err != nil {
				log.Printf("Error submitting container %s to slurm: %v",
					container.UUID, err)
				// maybe sbatch is broken, put it back to queued
				dispatcher.UpdateState(container.UUID, dispatch.Queued)
			}
		}
	}

	log.Printf("Monitoring container %v started", uuid)

	// periodically check squeue
	doneSqueue := make(chan struct{})
	go func() {
		squeueUpdater.CheckSqueue(container.UUID, true)
		ticker := time.NewTicker(dispatcher.PollInterval)
		for {
			select {
			case <-ticker.C:
				if inQ, err := squeueUpdater.CheckSqueue(container.UUID, false); err != nil {
					log.Printf("Error running squeue: %v", err)
					// don't cancel, just leave it the way it is
				} else if !inQ {
					var con dispatch.Container
					err := dispatcher.Arv.Get("containers", uuid, nil, &con)
					if err != nil {
						log.Printf("Error getting final container state: %v", err)
					}

					var st string
					switch con.State {
					case dispatch.Locked:
						st = dispatch.Queued
					case dispatch.Running:
						st = dispatch.Cancelled
					default:
						st = ""
					}

					if st != "" {
						log.Printf("Container %s in state %v but missing from slurm queue, changing to %v.",
							uuid, con.State, st)
						dispatcher.UpdateState(uuid, st)
					}
				}
			case <-doneSqueue:
				close(doneSqueue)
				ticker.Stop()
				return
			}
		}
	}()

	for container = range status {
		if container.State == dispatch.Locked || container.State == dispatch.Running {
			if container.Priority == 0 {
				log.Printf("Canceling container %s", container.UUID)

				err := exec.Command("scancel", "--name="+container.UUID).Run()
				if err != nil {
					log.Printf("Error stopping container %s with scancel: %v",
						container.UUID, err)
					if inQ, err := squeueUpdater.CheckSqueue(container.UUID, true); err != nil {
						log.Printf("Error running squeue: %v", err)
						continue
					} else if inQ {
						log.Printf("Container %s is still in squeue after scancel.",
							container.UUID)
						continue
					}
				}

				err = dispatcher.UpdateState(container.UUID, dispatch.Cancelled)
			}
		}
	}

	doneSqueue <- struct{}{}

	log.Printf("Monitoring container %v finished", uuid)
}
