package main

import (
	"errors"
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"github.com/curoverse/dockerclient"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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
type Mount struct{}

// Collection record returned by the API server.
type Collection struct {
	ManifestText string `json:"manifest_text"`
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
}

// NewLogWriter is a factory function to create a new log writer.
type NewLogWriter func(name string) io.WriteCloser

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
	CancelLock    sync.Mutex
	Cancelled     bool
	SigChan       chan os.Signal
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

	var collection Collection
	err = runner.ArvClient.Get("collections", runner.ContainerRecord.ContainerImage, nil, &collection)
	if err != nil {
		return err
	}
	manifest := manifest.Manifest{Text: collection.ManifestText}
	var img, imageID string
	for ms := range manifest.StreamIter() {
		img = ms.FileStreamSegments[0].Name
		if !strings.HasSuffix(img, ".tar") {
			return errors.New("First file in the collection does not end in .tar")
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
			return err
		}

		err = runner.Docker.LoadImage(readCloser)
		if err != nil {
			return err
		}
	} else {
		runner.CrunchLog.Print("Docker image is available")
	}

	runner.ContainerConfig.Image = imageID

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
	runner.ContainerID, err = runner.Docker.CreateContainer(&runner.ContainerConfig, "", nil)
	if err != nil {
		return
	}
	hostConfig := &dockerclient.HostConfig{}

	runner.CrunchLog.Printf("Starting Docker container id '%s'", runner.ContainerID)
	err = runner.Docker.StartContainer(runner.ContainerID, hostConfig)
	if err != nil {
		return
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
		return
	}
	stdoutReader, err = runner.Docker.ContainerLogs(runner.ContainerID, &dockerclient.LogOptions{Follow: true, Stdout: true})
	if err != nil {
		return
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
		return wr.Error
	}
	runner.ExitCode = &wr.ExitCode

	// drain stdout/stderr
	<-runner.loggingDone
	<-runner.loggingDone

	runner.Stdout.Close()
	runner.Stderr.Close()

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
		return err
	}

	response := make(map[string]string)
	err = runner.ArvClient.Create("collections",
		arvadosclient.Dict{"name": "logs for " + runner.ContainerRecord.UUID,
			"manifest_text": mt},
		response)
	if err != nil {
		return err
	}

	runner.LogsPDH = new(string)
	*runner.LogsPDH = response["portable_data_hash"]

	return nil
}

// UpdateContainerRecordRunning updates the container state to "Running"
func (runner *ContainerRunner) UpdateContainerRecordRunning() error {
	update := arvadosclient.Dict{"state": "Running"}
	return runner.ArvClient.Update("containers", runner.ContainerRecord.UUID, update, nil)
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

	update["state"] = runner.finalState

	return runner.ArvClient.Update("containers", runner.ContainerRecord.UUID, update, nil)
}

// NewArvLogWriter creates an ArvLogWriter
func (runner *ContainerRunner) NewArvLogWriter(name string) io.WriteCloser {
	return &ArvLogWriter{runner.ArvClient, runner.ContainerRecord.UUID, name, runner.LogCollection.Open(name + ".txt")}
}

// Run the full container lifecycle.
func (runner *ContainerRunner) Run(containerUUID string) (err error) {
	runner.CrunchLog.Printf("Executing container '%s'", containerUUID)

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

		// (6) write logs
		logerr := runner.CommitLogs()
		if logerr != nil {
			runner.CrunchLog.Print(logerr)
		}

		// (7) update container record with results
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

	err = runner.ArvClient.Get("containers", containerUUID, nil, &runner.ContainerRecord)
	if err != nil {
		return
	}

	// (0) setup signal handling
	err = runner.SetupSignals()
	if err != nil {
		return
	}

	// (1) check for and/or load image
	err = runner.LoadImage()
	if err != nil {
		return
	}

	// (2) start container
	err = runner.StartContainer()
	if err != nil {
		if err == ErrCancelled {
			err = nil
		}
		return
	}

	// (3) update container record state
	err = runner.UpdateContainerRecordRunning()
	if err != nil {
		runner.CrunchLog.Print(err)
	}

	// (4) attach container logs
	runerr = runner.AttachLogs()
	if runerr != nil {
		runner.CrunchLog.Print(runerr)
	}

	// (5) wait for container to finish
	waiterr = runner.WaitFinish()

	return
}

// NewContainerRunner creates a new container runner.
func NewContainerRunner(api IArvadosClient,
	kc IKeepClient,
	docker ThinDockerClient) *ContainerRunner {

	cr := &ContainerRunner{ArvClient: api, Kc: kc, Docker: docker}
	cr.NewLogWriter = cr.NewArvLogWriter
	cr.LogCollection = &CollectionWriter{kc, nil}
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

	cr := NewContainerRunner(api, kc, docker)

	err = cr.Run(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

}
