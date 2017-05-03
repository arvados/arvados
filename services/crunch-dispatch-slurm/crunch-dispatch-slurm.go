package main

// Dispatcher service for Crunch that submits containers to the slurm queue.

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	"github.com/coreos/go-systemd/daemon"
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

	// Minimum time between two attempts to run the same container
	MinRetryPeriod arvados.Duration
}

func main() {
	err := doMain()
	if err != nil {
		log.Fatal(err)
	}
}

var (
	theConfig Config
	sqCheck   = &SqueueChecker{}
)

const defaultConfigPath = "/etc/arvados/crunch-dispatch-slurm/crunch-dispatch-slurm.yml"

func doMain() error {
	flags := flag.NewFlagSet("crunch-dispatch-slurm", flag.ExitOnError)
	flags.Usage = func() { usage(flags) }

	configPath := flags.String(
		"config",
		defaultConfigPath,
		"`path` to JSON or YAML configuration file")
	dumpConfig := flag.Bool(
		"dump-config",
		false,
		"write current configuration to stdout and exit")

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

	if *dumpConfig {
		log.Fatal(config.DumpAndExit(theConfig))
	}

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Printf("Error making Arvados client: %v", err)
		return err
	}
	arv.Retries = 25

	sqCheck = &SqueueChecker{Period: time.Duration(theConfig.PollPeriod)}
	defer sqCheck.Stop()

	dispatcher := &dispatch.Dispatcher{
		Arv:            arv,
		RunContainer:   run,
		PollPeriod:     time.Duration(theConfig.PollPeriod),
		MinRetryPeriod: time.Duration(theConfig.MinRetryPeriod),
	}

	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}

	go checkSqueueForOrphans(dispatcher, sqCheck)

	return dispatcher.Run(context.Background())
}

var containerUuidPattern = regexp.MustCompile(`^[a-z0-9]{5}-dz642-[a-z0-9]{15}$`)

// Check the next squeue report, and invoke TrackContainer for all the
// containers in the report. This gives us a chance to cancel slurm
// jobs started by a previous dispatch process that never released
// their slurm allocations even though their container states are
// Cancelled or Complete. See https://dev.arvados.org/issues/10979
func checkSqueueForOrphans(dispatcher *dispatch.Dispatcher, sqCheck *SqueueChecker) {
	for _, uuid := range sqCheck.All() {
		if !containerUuidPattern.MatchString(uuid) {
			continue
		}
		err := dispatcher.TrackContainer(uuid)
		if err != nil {
			log.Printf("checkSqueueForOrphans: TrackContainer(%s): %s", uuid, err)
		}
	}
}

// sbatchCmd
func sbatchFunc(container arvados.Container) *exec.Cmd {
	mem := int64(math.Ceil(float64(container.RuntimeConstraints.RAM+container.RuntimeConstraints.KeepCacheRAM) / float64(1048576)))

	var disk int64
	for _, m := range container.Mounts {
		if m.Kind == "tmp" {
			disk += m.Capacity
		}
	}
	disk = int64(math.Ceil(float64(disk) / float64(1048576)))

	var sbatchArgs []string
	sbatchArgs = append(sbatchArgs, theConfig.SbatchArguments...)
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--job-name=%s", container.UUID))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--mem=%d", mem))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--cpus-per-task=%d", container.RuntimeConstraints.VCPUs))
	sbatchArgs = append(sbatchArgs, fmt.Sprintf("--tmp=%d", disk))
	if len(container.SchedulingParameters.Partitions) > 0 {
		sbatchArgs = append(sbatchArgs, fmt.Sprintf("--partition=%s", strings.Join(container.SchedulingParameters.Partitions, ",")))
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
func submit(dispatcher *dispatch.Dispatcher, container arvados.Container, crunchRunCommand []string) error {
	cmd := sbatchCmd(container)

	// Send a tiny script on stdin to execute the crunch-run
	// command (slurm requires this to be a #! script)
	cmd.Stdin = strings.NewReader(execScript(append(crunchRunCommand, container.UUID)))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Mutex between squeue sync and running sbatch or scancel.
	sqCheck.L.Lock()
	defer sqCheck.L.Unlock()

	log.Printf("exec sbatch %+q", cmd.Args)
	err := cmd.Run()

	switch err.(type) {
	case nil:
		log.Printf("sbatch succeeded: %q", strings.TrimSpace(stdout.String()))
		return nil

	case *exec.ExitError:
		dispatcher.Unlock(container.UUID)
		return fmt.Errorf("sbatch %+q failed: %v (stderr: %q)", cmd.Args, err, stderr.Bytes())

	default:
		dispatcher.Unlock(container.UUID)
		return fmt.Errorf("exec failed: %v", err)
	}
}

// Submit a container to the slurm queue (or resume monitoring if it's
// already in the queue).  Cancel the slurm job if the container's
// priority changes to zero or its state indicates it's no longer
// running.
func run(disp *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if ctr.State == dispatch.Locked && !sqCheck.HasUUID(ctr.UUID) {
		log.Printf("Submitting container %s to slurm", ctr.UUID)
		if err := submit(disp, ctr, theConfig.CrunchRunCommand); err != nil {
			log.Printf("Error submitting container %s to slurm: %s", ctr.UUID, err)
			disp.Unlock(ctr.UUID)
			return
		}
	}

	log.Printf("Start monitoring container %s", ctr.UUID)
	defer log.Printf("Done monitoring container %s", ctr.UUID)

	// If the container disappears from the slurm queue, there is
	// no point in waiting for further dispatch updates: just
	// clean up and return.
	go func(uuid string) {
		for ctx.Err() == nil && sqCheck.HasUUID(uuid) {
		}
		cancel()
	}(ctr.UUID)

	for {
		select {
		case <-ctx.Done():
			// Disappeared from squeue
			if err := disp.Arv.Get("containers", ctr.UUID, nil, &ctr); err != nil {
				log.Printf("Error getting final container state for %s: %s", ctr.UUID, err)
			}
			switch ctr.State {
			case dispatch.Running:
				disp.UpdateState(ctr.UUID, dispatch.Cancelled)
			case dispatch.Locked:
				disp.Unlock(ctr.UUID)
			}
			return
		case updated, ok := <-status:
			if !ok {
				log.Printf("Dispatcher says container %s is done: cancel slurm job", ctr.UUID)
				scancel(ctr)
			} else if updated.Priority == 0 {
				log.Printf("Container %s has state %q, priority %d: cancel slurm job", ctr.UUID, updated.State, updated.Priority)
				scancel(ctr)
			}
		}
	}
}

func scancel(ctr arvados.Container) {
	sqCheck.L.Lock()
	cmd := scancelCmd(ctr)
	msg, err := cmd.CombinedOutput()
	sqCheck.L.Unlock()

	if err != nil {
		log.Printf("%q %q: %s %q", cmd.Path, cmd.Args, err, msg)
		time.Sleep(time.Second)
	} else if sqCheck.HasUUID(ctr.UUID) {
		log.Printf("container %s is still in squeue after scancel", ctr.UUID)
		time.Sleep(time.Second)
	}
}

func readConfig(dst interface{}, path string) error {
	err := config.LoadFile(dst, path)
	if err != nil && os.IsNotExist(err) && path == defaultConfigPath {
		log.Printf("Config not specified. Continue with default configuration.")
		err = nil
	}
	return err
}
