package main

// Dispatcher service for Crunch that submits containers to the slurm queue.

import (
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	"github.com/coreos/go-systemd/daemon"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Config used by crunch-dispatch-slurm
type Config struct {
	Client arvados.Client

	SbatchArguments []string
	PollPeriod      arvados.Duration

	// crunch-run command to invoke. The container UUID will be
	// appended. If nil, []string{"crunch-run"} will be used.
	//
	// Example: []string{"crunch-run", "--cgroup-parent-subsystem=memory"}
	CrunchRunCommand []string
}

func main() {
	err := doMain()
	if err != nil {
		log.Fatal(err)
	}
}

var (
	theConfig     Config
	squeueUpdater Squeue
)

const defaultConfigPath = "/etc/arvados/crunch-dispatch-slurm/crunch-dispatch-slurm.yml"

func doMain() error {
	flags := flag.NewFlagSet("crunch-dispatch-slurm", flag.ExitOnError)
	flags.Usage = func() { usage(flags) }

	configPath := flags.String(
		"config",
		defaultConfigPath,
		"`path` to JSON or YAML configuration file")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	err := readConfig(&theConfig, *configPath)
	if err != nil {
		return err
	}

	if theConfig.CrunchRunCommand == nil {
		theConfig.CrunchRunCommand = []string{"crunch-run"}
	}

	if theConfig.PollPeriod == 0 {
		theConfig.PollPeriod = arvados.Duration(10 * time.Second)
	}

	if theConfig.Client.APIHost != "" || theConfig.Client.AuthToken != "" {
		// Copy real configs into env vars so [a]
		// MakeArvadosClient() uses them, and [b] they get
		// propagated to crunch-run via SLURM.
		os.Setenv("ARVADOS_API_HOST", theConfig.Client.APIHost)
		os.Setenv("ARVADOS_API_TOKEN", theConfig.Client.AuthToken)
		os.Setenv("ARVADOS_API_HOST_INSECURE", "")
		if theConfig.Client.Insecure {
			os.Setenv("ARVADOS_API_HOST_INSECURE", "1")
		}
		os.Setenv("ARVADOS_KEEP_SERVICES", strings.Join(theConfig.Client.KeepServiceURIs, " "))
		os.Setenv("ARVADOS_EXTERNAL_CLIENT", "")
	} else {
		log.Printf("warning: Client credentials missing from config, so falling back on environment variables (deprecated).")
	}

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Printf("Error making Arvados client: %v", err)
		return err
	}
	arv.Retries = 25

	squeueUpdater.StartMonitor(time.Duration(theConfig.PollPeriod))
	defer squeueUpdater.Done()

	dispatcher := dispatch.Dispatcher{
		Arv:            arv,
		RunContainer:   run,
		PollInterval:   time.Duration(theConfig.PollPeriod),
		DoneProcessing: make(chan struct{})}

	if _, err := daemon.SdNotify("READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}

	err = dispatcher.RunDispatcher()
	if err != nil {
		return err
	}

	return nil
}

// sbatchCmd
func sbatchFunc(container arvados.Container) *exec.Cmd {
	memPerCPU := math.Ceil(float64(container.RuntimeConstraints.RAM) / (float64(container.RuntimeConstraints.VCPUs) * 1048576))

	var sbatchArgs []string
	sbatchArgs = append(sbatchArgs, "--share")
	sbatchArgs = append(sbatchArgs, theConfig.SbatchArguments...)
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--job-name=%s", container.UUID))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--mem-per-cpu=%d", int(memPerCPU)))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--cpus-per-task=%d", container.RuntimeConstraints.VCPUs))
	if container.RuntimeConstraints.Partition != nil {
		sbatchArgs = append(sbatchArgs, fmt.Sprintf("--partition=%s", strings.Join(container.RuntimeConstraints.Partition, ",")))
	}

	return exec.Command("sbatch", sbatchArgs...)
}

// scancelCmd
func scancelFunc(container arvados.Container) *exec.Cmd {
	return exec.Command("scancel", "--name="+container.UUID)
}

// Wrap these so that they can be overridden by tests
var sbatchCmd = sbatchFunc
var scancelCmd = scancelFunc

// Submit job to slurm using sbatch.
func submit(dispatcher *dispatch.Dispatcher,
	container arvados.Container, crunchRunCommand []string) (submitErr error) {
	defer func() {
		// If we didn't get as far as submitting a slurm job,
		// unlock the container and return it to the queue.
		if submitErr == nil {
			// OK, no cleanup needed
			return
		}
		err := dispatcher.Unlock(container.UUID)
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

	// Mutex between squeue sync and running sbatch or scancel.
	squeueUpdater.SlurmLock.Lock()
	defer squeueUpdater.SlurmLock.Unlock()

	log.Printf("sbatch starting: %+q", cmd.Args)
	err := cmd.Start()
	if err != nil {
		submitErr = fmt.Errorf("Error starting sbatch: %v", err)
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
	io.WriteString(stdinWriter, execScript(append(crunchRunCommand, container.UUID)))
	stdinWriter.Close()

	err = cmd.Wait()

	stdoutMsg := <-stdoutChan
	stderrmsg := <-stderrChan

	close(stdoutChan)
	close(stderrChan)

	if err != nil {
		submitErr = fmt.Errorf("Container submission failed: %v: %v (stderr: %q)", cmd.Args, err, stderrmsg)
		return
	}

	log.Printf("sbatch succeeded: %s", strings.TrimSpace(string(stdoutMsg)))
	return
}

// If the container is marked as Locked, check if it is already in the slurm
// queue.  If not, submit it.
//
// If the container is marked as Running, check if it is in the slurm queue.
// If not, mark it as Cancelled.
func monitorSubmitOrCancel(dispatcher *dispatch.Dispatcher, container arvados.Container, monitorDone *bool) {
	submitted := false
	for !*monitorDone {
		if squeueUpdater.CheckSqueue(container.UUID) {
			// Found in the queue, so continue monitoring
			submitted = true
		} else if container.State == dispatch.Locked && !submitted {
			// Not in queue but in Locked state and we haven't
			// submitted it yet, so submit it.

			log.Printf("About to submit queued container %v", container.UUID)

			if err := submit(dispatcher, container, theConfig.CrunchRunCommand); err != nil {
				log.Printf("Error submitting container %s to slurm: %v",
					container.UUID, err)
				// maybe sbatch is broken, put it back to queued
				dispatcher.Unlock(container.UUID)
			}
			submitted = true
		} else {
			// Not in queue and we are not going to submit it.
			// Refresh the container state. If it is
			// Complete/Cancelled, do nothing, if it is Locked then
			// release it back to the Queue, if it is Running then
			// clean up the record.

			var con arvados.Container
			err := dispatcher.Arv.Get("containers", container.UUID, nil, &con)
			if err != nil {
				log.Printf("Error getting final container state: %v", err)
			}

			switch con.State {
			case dispatch.Locked:
				log.Printf("Container %s in state %v but missing from slurm queue, changing to %v.",
					container.UUID, con.State, dispatch.Queued)
				dispatcher.Unlock(container.UUID)
			case dispatch.Running:
				st := dispatch.Cancelled
				log.Printf("Container %s in state %v but missing from slurm queue, changing to %v.",
					container.UUID, con.State, st)
				dispatcher.UpdateState(container.UUID, st)
			default:
				// Container state is Queued, Complete or Cancelled so stop monitoring it.
				return
			}
		}
	}
}

// Run or monitor a container.
//
// Monitor status updates.  If the priority changes to zero, cancel the
// container using scancel.
func run(dispatcher *dispatch.Dispatcher,
	container arvados.Container,
	status chan arvados.Container) {

	log.Printf("Monitoring container %v started", container.UUID)
	defer log.Printf("Monitoring container %v finished", container.UUID)

	monitorDone := false
	go monitorSubmitOrCancel(dispatcher, container, &monitorDone)

	for container = range status {
		if container.State == dispatch.Locked || container.State == dispatch.Running {
			if container.Priority == 0 {
				log.Printf("Canceling container %s", container.UUID)

				// Mutex between squeue sync and running sbatch or scancel.
				squeueUpdater.SlurmLock.Lock()
				err := scancelCmd(container).Run()
				squeueUpdater.SlurmLock.Unlock()

				if err != nil {
					log.Printf("Error stopping container %s with scancel: %v",
						container.UUID, err)
					if squeueUpdater.CheckSqueue(container.UUID) {
						log.Printf("Container %s is still in squeue after scancel.",
							container.UUID)
						continue
					}
				}

				err = dispatcher.UpdateState(container.UUID, dispatch.Cancelled)
			}
		}
	}
	monitorDone = true
}

func readConfig(dst interface{}, path string) error {
	err := config.LoadFile(dst, path)
	if err != nil && os.IsNotExist(err) && path == defaultConfigPath {
		log.Printf("Config not specified. Continue with default configuration.")
		err = nil
	}
	return err
}
