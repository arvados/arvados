package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"github.com/curoverse/dockerclient"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// IArvadosClient is the minimal Arvados API methods used by crunch-run.
type IArvadosClient interface {
	Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error
	Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error
	Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error)
}

// ErrCancelled is the error returned when the container is cancelled.
var ErrCancelled = errors.New("Cancelled")

// IKeepClient is the minimal Keep API methods used by crunch-run.
type IKeepClient interface {
	PutHB(hash string, buf []byte) (string, int, error)
	ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error)
}

// Mount describes the mount points to create inside the container.
type Mount struct {
	Kind             string `json:"kind"`
	Writable         bool   `json:"writable"`
	PortableDataHash string `json:"portable_data_hash"`
	UUID             string `json:"uuid"`
	DeviceType       string `json:"device_type"`
}

// Collection record returned by the API server.
type CollectionRecord struct {
	ManifestText     string `json:"manifest_text"`
	PortableDataHash string `json:"portable_data_hash"`
}

// ContainerRecord is the container record returned by the API server.
type ContainerRecord struct {
	UUID               string                 `json:"uuid"`
	Command            []string               `json:"command"`
	ContainerImage     string                 `json:"container_image"`
	Cwd                string                 `json:"cwd"`
	Environment        map[string]string      `json:"environment"`
	Mounts             map[string]Mount       `json:"mounts"`
	OutputPath         string                 `json:"output_path"`
	Priority           int                    `json:"priority"`
	RuntimeConstraints map[string]interface{} `json:"runtime_constraints"`
	State              string                 `json:"state"`
	Output             string                 `json:"output"`
}

// NewLogWriter is a factory function to create a new log writer.
type NewLogWriter func(name string) io.WriteCloser

type RunArvMount func([]string) (*exec.Cmd, error)

type MkTempDir func(string, string) (string, error)

// ThinDockerClient is the minimal Docker client interface used by crunch-run.
type ThinDockerClient interface {
	StopContainer(id string, timeout int) error
	InspectImage(id string) (*dockerclient.ImageInfo, error)
	LoadImage(reader io.Reader) error
	CreateContainer(config *dockerclient.ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (string, error)
	StartContainer(id string, config *dockerclient.HostConfig) error
	ContainerLogs(id string, options *dockerclient.LogOptions) (io.ReadCloser, error)
	Wait(id string) <-chan dockerclient.WaitResult
	RemoveImage(name string, force bool) ([]*dockerclient.ImageDelete, error)
}

// ContainerRunner is the main stateful struct used for a single execution of a
// container.
type ContainerRunner struct {
	Docker    ThinDockerClient
	ArvClient IArvadosClient
	Kc        IKeepClient
	ContainerRecord
	dockerclient.ContainerConfig
	ContainerID string
	ExitCode    *int
	NewLogWriter
	loggingDone   chan bool
	CrunchLog     *ThrottledLogger
	Stdout        *ThrottledLogger
	Stderr        *ThrottledLogger
	LogCollection *CollectionWriter
	LogsPDH       *string
	RunArvMount
	MkTempDir
	ArvMount      *exec.Cmd
	ArvMountPoint string
	HostOutputDir string
	Binds         []string
	OutputPDH     *string
	CancelLock    sync.Mutex
	Cancelled     bool
	SigChan       chan os.Signal
	ArvMountExit  chan error
	finalState    string
}

// SetupSignals sets up signal handling to gracefully terminate the underlying
// Docker container and update state when receiving a TERM, INT or QUIT signal.
func (runner *ContainerRunner) SetupSignals() error {
	runner.SigChan = make(chan os.Signal, 1)
	signal.Notify(runner.SigChan, syscall.SIGTERM)
	signal.Notify(runner.SigChan, syscall.SIGINT)
	signal.Notify(runner.SigChan, syscall.SIGQUIT)

	go func(sig <-chan os.Signal) {
		for _ = range sig {
			if !runner.Cancelled {
				runner.CancelLock.Lock()
				runner.Cancelled = true
				if runner.ContainerID != "" {
					runner.Docker.StopContainer(runner.ContainerID, 10)
				}
				runner.CancelLock.Unlock()
			}
		}
	}(runner.SigChan)

	return nil
}

// LoadImage determines the docker image id from the container record and
// checks if it is available in the local Docker image store.  If not, it loads
// the image from Keep.
func (runner *ContainerRunner) LoadImage() (err error) {

	runner.CrunchLog.Printf("Fetching Docker image from collection '%s'", runner.ContainerRecord.ContainerImage)

	var collection CollectionRecord
	err = runner.ArvClient.Get("collections", runner.ContainerRecord.ContainerImage, nil, &collection)
	if err != nil {
		return fmt.Errorf("While getting container image collection: %v", err)
	}
	manifest := manifest.Manifest{Text: collection.ManifestText}
	var img, imageID string
	for ms := range manifest.StreamIter() {
		img = ms.FileStreamSegments[0].Name
		if !strings.HasSuffix(img, ".tar") {
			return fmt.Errorf("First file in the container image collection does not end in .tar")
		}
		imageID = img[:len(img)-4]
	}

	runner.CrunchLog.Printf("Using Docker image id '%s'", imageID)

	_, err = runner.Docker.InspectImage(imageID)
	if err != nil {
		runner.CrunchLog.Print("Loading Docker image from keep")

		var readCloser io.ReadCloser
		readCloser, err = runner.Kc.ManifestFileReader(manifest, img)
		if err != nil {
			return fmt.Errorf("While creating ManifestFileReader for container image: %v", err)
		}

		err = runner.Docker.LoadImage(readCloser)
		if err != nil {
			return fmt.Errorf("While loading container image into Docker: %v", err)
		}
	} else {
		runner.CrunchLog.Print("Docker image is available")
	}

	runner.ContainerConfig.Image = imageID

	return nil
}

func (runner *ContainerRunner) ArvMountCmd(arvMountCmd []string) (c *exec.Cmd, err error) {
	c = exec.Command("arv-mount", arvMountCmd...)
	nt := NewThrottledLogger(runner.NewLogWriter("arv-mount"))
	c.Stdout = nt
	c.Stderr = nt

	err = c.Start()
	if err != nil {
		return nil, err
	}

	statReadme := make(chan bool)
	runner.ArvMountExit = make(chan error)

	keepStatting := true
	go func() {
		for keepStatting {
			time.Sleep(100 * time.Millisecond)
			_, err = os.Stat(fmt.Sprintf("%s/by_id/README", runner.ArvMountPoint))
			if err == nil {
				keepStatting = false
				statReadme <- true
			}
		}
		close(statReadme)
	}()

	go func() {
		runner.ArvMountExit <- c.Wait()
		close(runner.ArvMountExit)
	}()

	select {
	case <-statReadme:
		break
	case err := <-runner.ArvMountExit:
		runner.ArvMount = nil
		keepStatting = false
		return nil, err
	}

	return c, nil
}

func (runner *ContainerRunner) SetupMounts() (err error) {
	runner.ArvMountPoint, err = runner.MkTempDir("", "keep")
	if err != nil {
		return fmt.Errorf("While creating keep mount temp dir: %v", err)
	}

	pdhOnly := true
	tmpcount := 0
	arvMountCmd := []string{"--foreground"}
	collectionPaths := []string{}
	runner.Binds = nil

	for bind, mnt := range runner.ContainerRecord.Mounts {
		if mnt.Kind == "collection" {
			var src string
			if mnt.UUID != "" && mnt.PortableDataHash != "" {
				return fmt.Errorf("Cannot specify both 'uuid' and 'portable_data_hash' for a collection mount")
			}
			if mnt.UUID != "" {
				if mnt.Writable {
					return fmt.Errorf("Writing to existing collections currently not permitted.")
				}
				pdhOnly = false
				src = fmt.Sprintf("%s/by_id/%s", runner.ArvMountPoint, mnt.UUID)
			} else if mnt.PortableDataHash != "" {
				if mnt.Writable {
					return fmt.Errorf("Can never write to a collection specified by portable data hash")
				}
				src = fmt.Sprintf("%s/by_id/%s", runner.ArvMountPoint, mnt.PortableDataHash)
			} else {
				src = fmt.Sprintf("%s/tmp%d", runner.ArvMountPoint, tmpcount)
				arvMountCmd = append(arvMountCmd, "--mount-tmp")
				arvMountCmd = append(arvMountCmd, fmt.Sprintf("tmp%d", tmpcount))
				tmpcount += 1
			}
			if mnt.Writable {
				if bind == runner.ContainerRecord.OutputPath {
					runner.HostOutputDir = src
				}
				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s", src, bind))
			} else {
				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s:ro", src, bind))
			}
			collectionPaths = append(collectionPaths, src)
		} else if mnt.Kind == "tmp" {
			if bind == runner.ContainerRecord.OutputPath {
				runner.HostOutputDir, err = runner.MkTempDir("", "")
				if err != nil {
					return fmt.Errorf("While creating mount temp dir: %v", err)
				}

				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s", runner.HostOutputDir, bind))
			} else {
				runner.Binds = append(runner.Binds, bind)
			}
		} else {
			return fmt.Errorf("Unknown mount kind '%s'", mnt.Kind)
		}
	}

	if runner.HostOutputDir == "" {
		return fmt.Errorf("Output path does not correspond to a writable mount point")
	}

	if pdhOnly {
		arvMountCmd = append(arvMountCmd, "--mount-by-pdh", "by_id")
	} else {
		arvMountCmd = append(arvMountCmd, "--mount-by-id", "by_id")
	}
	arvMountCmd = append(arvMountCmd, runner.ArvMountPoint)

	runner.ArvMount, err = runner.RunArvMount(arvMountCmd)
	if err != nil {
		return fmt.Errorf("While trying to start arv-mount: %v", err)
	}

	for _, p := range collectionPaths {
		_, err = os.Stat(p)
		if err != nil {
			return fmt.Errorf("While checking that input files exist: %v", err)
		}
	}

	return nil
}

// StartContainer creates the container and runs it.
func (runner *ContainerRunner) StartContainer() (err error) {
	runner.CrunchLog.Print("Creating Docker container")

	runner.CancelLock.Lock()
	defer runner.CancelLock.Unlock()

	if runner.Cancelled {
		return ErrCancelled
	}

	runner.ContainerConfig.Cmd = runner.ContainerRecord.Command
	if runner.ContainerRecord.Cwd != "." {
		runner.ContainerConfig.WorkingDir = runner.ContainerRecord.Cwd
	}
	for k, v := range runner.ContainerRecord.Environment {
		runner.ContainerConfig.Env = append(runner.ContainerConfig.Env, k+"="+v)
	}
	runner.ContainerConfig.NetworkDisabled = true
	runner.ContainerID, err = runner.Docker.CreateContainer(&runner.ContainerConfig, "", nil)
	if err != nil {
		return fmt.Errorf("While creating container: %v", err)
	}
	hostConfig := &dockerclient.HostConfig{Binds: runner.Binds}

	runner.CrunchLog.Printf("Starting Docker container id '%s'", runner.ContainerID)
	err = runner.Docker.StartContainer(runner.ContainerID, hostConfig)
	if err != nil {
		return fmt.Errorf("While starting container: %v", err)
	}

	return nil
}

// AttachLogs connects the docker container stdout and stderr logs to the
// Arvados logger which logs to Keep and the API server logs table.
func (runner *ContainerRunner) AttachLogs() (err error) {

	runner.CrunchLog.Print("Attaching container logs")

	var stderrReader, stdoutReader io.Reader
	stderrReader, err = runner.Docker.ContainerLogs(runner.ContainerID, &dockerclient.LogOptions{Follow: true, Stderr: true})
	if err != nil {
		return fmt.Errorf("While getting container standard error: %v", err)
	}
	stdoutReader, err = runner.Docker.ContainerLogs(runner.ContainerID, &dockerclient.LogOptions{Follow: true, Stdout: true})
	if err != nil {
		return fmt.Errorf("While getting container standard output: %v", err)
	}

	runner.loggingDone = make(chan bool)

	runner.Stdout = NewThrottledLogger(runner.NewLogWriter("stdout"))
	runner.Stderr = NewThrottledLogger(runner.NewLogWriter("stderr"))
	go ReadWriteLines(stdoutReader, runner.Stdout, runner.loggingDone)
	go ReadWriteLines(stderrReader, runner.Stderr, runner.loggingDone)

	return nil
}

// WaitFinish waits for the container to terminate, capture the exit code, and
// close the stdout/stderr logging.
func (runner *ContainerRunner) WaitFinish() error {
	result := runner.Docker.Wait(runner.ContainerID)
	wr := <-result
	if wr.Error != nil {
		return fmt.Errorf("While waiting for container to finish: %v", wr.Error)
	}
	runner.ExitCode = &wr.ExitCode

	// drain stdout/stderr
	<-runner.loggingDone
	<-runner.loggingDone

	runner.Stdout.Close()
	runner.Stderr.Close()

	return nil
}

// HandleOutput sets the output and unmounts the FUSE mount.
func (runner *ContainerRunner) CaptureOutput() error {
	if runner.ArvMount != nil {
		defer func() {
			umount := exec.Command("fusermount", "-z", "-u", runner.ArvMountPoint)
			umnterr := umount.Run()
			if umnterr != nil {
				runner.CrunchLog.Print("While running fusermount: %v", umnterr)
			}

			mnterr := <-runner.ArvMountExit
			if mnterr != nil {
				runner.CrunchLog.Print("Arv-mount exit error: %v", mnterr)
			}
		}()
	}

	if runner.finalState != "Complete" {
		return nil
	}

	if runner.HostOutputDir == "" {
		return nil
	}

	_, err := os.Stat(runner.HostOutputDir)
	if err != nil {
		return fmt.Errorf("While checking host output path: %v", err)
	}

	var manifestText string

	collectionMetafile := fmt.Sprintf("%s/.arvados#collection", runner.HostOutputDir)
	_, err = os.Stat(collectionMetafile)
	if err != nil {
		// Regular directory
		cw := CollectionWriter{runner.Kc, nil, sync.Mutex{}}
		manifestText, err = cw.WriteTree(runner.HostOutputDir, runner.CrunchLog.Logger)
		if err != nil {
			return fmt.Errorf("While uploading output files: %v", err)
		}
	} else {
		// FUSE mount directory
		file, openerr := os.Open(collectionMetafile)
		if openerr != nil {
			return fmt.Errorf("While opening FUSE metafile: %v", err)
		}
		defer file.Close()

		rec := CollectionRecord{}
		err = json.NewDecoder(file).Decode(&rec)
		if err != nil {
			return fmt.Errorf("While reading FUSE metafile: %v", err)
		}
		manifestText = rec.ManifestText
	}

	var response CollectionRecord
	err = runner.ArvClient.Create("collections",
		arvadosclient.Dict{
			"collection": arvadosclient.Dict{
				"manifest_text": manifestText}},
		&response)
	if err != nil {
		return fmt.Errorf("While creating output collection: %v", err)
	}

	runner.OutputPDH = new(string)
	*runner.OutputPDH = response.PortableDataHash

	return nil
}

// CommitLogs posts the collection containing the final container logs.
func (runner *ContainerRunner) CommitLogs() error {
	runner.CrunchLog.Print(runner.finalState)
	runner.CrunchLog.Close()

	// Closing CrunchLog above allows it to be committed to Keep at this
	// point, but re-open crunch log with ArvClient in case there are any
	// other further (such as failing to write the log to Keep!) while
	// shutting down
	runner.CrunchLog = NewThrottledLogger(&ArvLogWriter{runner.ArvClient, runner.ContainerRecord.UUID,
		"crunch-run", nil})

	mt, err := runner.LogCollection.ManifestText()
	if err != nil {
		return fmt.Errorf("While creating log manifest: %v", err)
	}

	var response CollectionRecord
	err = runner.ArvClient.Create("collections",
		arvadosclient.Dict{
			"collection": arvadosclient.Dict{
				"name":          "logs for " + runner.ContainerRecord.UUID,
				"manifest_text": mt}},
		&response)
	if err != nil {
		return fmt.Errorf("While creating log collection: %v", err)
	}

	runner.LogsPDH = new(string)
	*runner.LogsPDH = response.PortableDataHash

	return nil
}

// UpdateContainerRecordRunning updates the container state to "Running"
func (runner *ContainerRunner) UpdateContainerRecordRunning() error {
	return runner.ArvClient.Update("containers", runner.ContainerRecord.UUID,
		arvadosclient.Dict{"container": arvadosclient.Dict{"state": "Running"}}, nil)
}

// UpdateContainerRecordComplete updates the container record state on API
// server to "Complete" or "Cancelled"
func (runner *ContainerRunner) UpdateContainerRecordComplete() error {
	update := arvadosclient.Dict{}
	if runner.LogsPDH != nil {
		update["log"] = *runner.LogsPDH
	}
	if runner.ExitCode != nil {
		update["exit_code"] = *runner.ExitCode
	}
	if runner.OutputPDH != nil {
		update["output"] = runner.OutputPDH
	}

	update["state"] = runner.finalState

	return runner.ArvClient.Update("containers", runner.ContainerRecord.UUID, arvadosclient.Dict{"container": update}, nil)
}

// NewArvLogWriter creates an ArvLogWriter
func (runner *ContainerRunner) NewArvLogWriter(name string) io.WriteCloser {
	return &ArvLogWriter{runner.ArvClient, runner.ContainerRecord.UUID, name, runner.LogCollection.Open(name + ".txt")}
}

// Run the full container lifecycle.
func (runner *ContainerRunner) Run() (err error) {
	runner.CrunchLog.Printf("Executing container '%s'", runner.ContainerRecord.UUID)

	var runerr, waiterr error

	defer func() {
		if err != nil {
			runner.CrunchLog.Print(err)
		}

		if runner.Cancelled {
			runner.finalState = "Cancelled"
		} else {
			runner.finalState = "Complete"
		}

		// (7) capture output
		outputerr := runner.CaptureOutput()
		if outputerr != nil {
			runner.CrunchLog.Print(outputerr)
		}

		// (8) write logs
		logerr := runner.CommitLogs()
		if logerr != nil {
			runner.CrunchLog.Print(logerr)
		}

		// (9) update container record with results
		updateerr := runner.UpdateContainerRecordComplete()
		if updateerr != nil {
			runner.CrunchLog.Print(updateerr)
		}

		runner.CrunchLog.Close()

		if err == nil {
			if runerr != nil {
				err = runerr
			} else if waiterr != nil {
				err = runerr
			} else if logerr != nil {
				err = logerr
			} else if updateerr != nil {
				err = updateerr
			}
		}
	}()

	err = runner.ArvClient.Get("containers", runner.ContainerRecord.UUID, nil, &runner.ContainerRecord)
	if err != nil {
		return fmt.Errorf("While getting container record: %v", err)
	}

	// (1) setup signal handling
	err = runner.SetupSignals()
	if err != nil {
		return fmt.Errorf("While setting up signal handling: %v", err)
	}

	// (2) check for and/or load image
	err = runner.LoadImage()
	if err != nil {
		return fmt.Errorf("While loading container image: %v", err)
	}

	// (3) set up FUSE mount and binds
	err = runner.SetupMounts()
	if err != nil {
		return fmt.Errorf("While setting up mounts: %v", err)
	}

	// (3) create and start container
	err = runner.StartContainer()
	if err != nil {
		if err == ErrCancelled {
			err = nil
		}
		return
	}

	// (4) update container record state
	err = runner.UpdateContainerRecordRunning()
	if err != nil {
		runner.CrunchLog.Print(err)
	}

	// (5) attach container logs
	runerr = runner.AttachLogs()
	if runerr != nil {
		runner.CrunchLog.Print(runerr)
	}

	// (6) wait for container to finish
	waiterr = runner.WaitFinish()

	return
}

// NewContainerRunner creates a new container runner.
func NewContainerRunner(api IArvadosClient,
	kc IKeepClient,
	docker ThinDockerClient,
	containerUUID string) *ContainerRunner {

	cr := &ContainerRunner{ArvClient: api, Kc: kc, Docker: docker}
	cr.NewLogWriter = cr.NewArvLogWriter
	cr.RunArvMount = cr.ArvMountCmd
	cr.MkTempDir = ioutil.TempDir
	cr.LogCollection = &CollectionWriter{kc, nil, sync.Mutex{}}
	cr.ContainerRecord.UUID = containerUUID
	cr.CrunchLog = NewThrottledLogger(cr.NewLogWriter("crunch-run"))
	return cr
}

func main() {
	flag.Parse()

	api, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatal(err)
	}
	api.Retries = 8

	var kc *keepclient.KeepClient
	kc, err = keepclient.MakeKeepClient(&api)
	if err != nil {
		log.Fatal(err)
	}
	kc.Retries = 4

	var docker *dockerclient.DockerClient
	docker, err = dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		log.Fatal(err)
	}

	cr := NewContainerRunner(api, kc, docker, flag.Arg(0))

	err = cr.Run()
	if err != nil {
		log.Fatal(err)
	}

}
