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

type IArvadosClient interface {
	Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error
	Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error
	Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error)
}

var ErrCancelled = errors.New("Cancelled")

type IKeepClient interface {
	PutHB(hash string, buf []byte) (string, int, error)
	ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error)
}

type Mount struct{}

type Collection struct {
	ManifestText string `json:"manifest_text"`
}

type ContainerRecord struct {
	Uuid               string                 `json:"uuid"`
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

type NewLogWriter func(name string) io.WriteCloser

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

type ContainerRunner struct {
	Docker ThinDockerClient
	Api    IArvadosClient
	Kc     IKeepClient
	ContainerRecord
	dockerclient.ContainerConfig
	ContainerId string
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

func (this *ContainerRunner) SetupSignals() error {
	this.SigChan = make(chan os.Signal, 1)
	signal.Notify(this.SigChan, syscall.SIGTERM)
	signal.Notify(this.SigChan, syscall.SIGINT)
	signal.Notify(this.SigChan, syscall.SIGQUIT)

	go func(sig <-chan os.Signal) {
		for _ = range sig {
			if !this.Cancelled {
				this.CancelLock.Lock()
				this.Cancelled = true
				if this.ContainerId != "" {
					this.Docker.StopContainer(this.ContainerId, 10)
				}
				this.CancelLock.Unlock()
			}
		}
	}(this.SigChan)

	return nil
}

func (this *ContainerRunner) LoadImage() (err error) {

	this.CrunchLog.Printf("Fetching Docker image from collection '%s'", this.ContainerRecord.ContainerImage)

	var collection Collection
	err = this.Api.Get("collections", this.ContainerRecord.ContainerImage, nil, &collection)
	if err != nil {
		return err
	}
	manifest := manifest.Manifest{Text: collection.ManifestText}
	var img, imageId string
	for ms := range manifest.StreamIter() {
		img = ms.FileStreamSegments[0].Name
		if !strings.HasSuffix(img, ".tar") {
			return errors.New("First file in the collection does not end in .tar")
		}
		imageId = img[:len(img)-4]
	}

	this.CrunchLog.Printf("Using Docker image id '%s'", imageId)

	_, err = this.Docker.InspectImage(imageId)
	if err != nil {
		this.CrunchLog.Print("Loading Docker image from keep")

		var readCloser io.ReadCloser
		readCloser, err = this.Kc.ManifestFileReader(manifest, img)
		if err != nil {
			return err
		}

		err = this.Docker.LoadImage(readCloser)
		if err != nil {
			return err
		}
	} else {
		this.CrunchLog.Print("Docker image is available")
	}

	this.ContainerConfig.Image = imageId

	return nil
}

func (this *ContainerRunner) StartContainer() (err error) {
	this.CrunchLog.Print("Creating Docker container")

	this.CancelLock.Lock()
	defer this.CancelLock.Unlock()

	if this.Cancelled {
		return ErrCancelled
	}

	this.ContainerConfig.Cmd = this.ContainerRecord.Command
	if this.ContainerRecord.Cwd != "." {
		this.ContainerConfig.WorkingDir = this.ContainerRecord.Cwd
	}
	for k, v := range this.ContainerRecord.Environment {
		this.ContainerConfig.Env = append(this.ContainerConfig.Env, k+"="+v)
	}
	this.ContainerId, err = this.Docker.CreateContainer(&this.ContainerConfig, "", nil)
	if err != nil {
		return
	}
	hostConfig := &dockerclient.HostConfig{}

	this.CrunchLog.Printf("Starting Docker container id '%s'", this.ContainerId)
	err = this.Docker.StartContainer(this.ContainerId, hostConfig)
	if err != nil {
		return
	}

	return nil
}

func (this *ContainerRunner) AttachLogs() (err error) {

	this.CrunchLog.Print("Attaching container logs")

	var stderrReader, stdoutReader io.Reader
	stderrReader, err = this.Docker.ContainerLogs(this.ContainerId, &dockerclient.LogOptions{Follow: true, Stderr: true})
	if err != nil {
		return
	}
	stdoutReader, err = this.Docker.ContainerLogs(this.ContainerId, &dockerclient.LogOptions{Follow: true, Stdout: true})
	if err != nil {
		return
	}

	this.loggingDone = make(chan bool)

	this.Stdout = NewThrottledLogger(this.NewLogWriter("stdout"))
	this.Stderr = NewThrottledLogger(this.NewLogWriter("stderr"))
	go CopyReaderToLog(stdoutReader, this.Stdout.Logger, this.loggingDone)
	go CopyReaderToLog(stderrReader, this.Stderr.Logger, this.loggingDone)

	return nil
}

func (this *ContainerRunner) WaitFinish() error {
	result := this.Docker.Wait(this.ContainerId)
	wr := <-result
	if wr.Error != nil {
		return wr.Error
	}
	this.ExitCode = &wr.ExitCode

	// drain stdout/stderr
	<-this.loggingDone
	<-this.loggingDone

	this.Stdout.Close()
	this.Stderr.Close()

	return nil
}

func (this *ContainerRunner) CommitLogs() error {
	this.CrunchLog.Print(this.finalState)
	this.CrunchLog.Close()
	this.CrunchLog = NewThrottledLogger(&ArvLogWriter{this.Api, this.ContainerRecord.Uuid,
		"crunchexec", nil})

	mt, err := this.LogCollection.ManifestText()
	if err != nil {
		return err
	}

	response := make(map[string]string)
	err = this.Api.Create("collections",
		arvadosclient.Dict{"name": "logs for " + this.ContainerRecord.Uuid,
			"manifest_text": mt},
		response)
	if err != nil {
		return err
	}

	this.LogsPDH = new(string)
	*this.LogsPDH = response["portable_data_hash"]

	return nil
}

func (this *ContainerRunner) UpdateContainerRecordRunning() error {
	update := arvadosclient.Dict{"state": "Running"}
	return this.Api.Update("containers", this.ContainerRecord.Uuid, update, nil)
}

func (this *ContainerRunner) UpdateContainerRecordComplete() error {
	update := arvadosclient.Dict{}
	if this.LogsPDH != nil {
		update["log"] = *this.LogsPDH
	}
	if this.ExitCode != nil {
		update["exit_code"] = *this.ExitCode
	}

	update["state"] = this.finalState

	return this.Api.Update("containers", this.ContainerRecord.Uuid, update, nil)
}

func (this *ContainerRunner) NewArvLogWriter(name string) io.WriteCloser {
	return &ArvLogWriter{this.Api, this.ContainerRecord.Uuid, name, this.LogCollection.Open(name + ".txt")}
}

func (this *ContainerRunner) Run(containerUuid string) (err error) {
	this.CrunchLog.Printf("Executing container '%s'", containerUuid)

	var runerr, waiterr error

	defer func() {
		if err != nil {
			this.CrunchLog.Print(err)
		}

		if this.Cancelled {
			this.finalState = "Cancelled"
		} else {
			this.finalState = "Complete"
		}

		// (6) write logs
		logerr := this.CommitLogs()
		if logerr != nil {
			this.CrunchLog.Print(logerr)
		}

		// (7) update container record with results
		updateerr := this.UpdateContainerRecordComplete()
		if updateerr != nil {
			this.CrunchLog.Print(updateerr)
		}

		this.CrunchLog.Close()

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

	err = this.Api.Get("containers", containerUuid, nil, &this.ContainerRecord)
	if err != nil {
		return
	}

	// (0) setup signal handling
	err = this.SetupSignals()
	if err != nil {
		return
	}

	// (1) check for and/or load image
	err = this.LoadImage()
	if err != nil {
		return
	}

	// (2) start container
	err = this.StartContainer()
	if err != nil {
		if err == ErrCancelled {
			err = nil
		}
		return
	}

	// (3) update container record state
	err = this.UpdateContainerRecordRunning()
	if err != nil {
		this.CrunchLog.Print(err)
	}

	// (4) attach container logs
	runerr = this.AttachLogs()
	if runerr != nil {
		this.CrunchLog.Print(runerr)
	}

	// (5) wait for container to finish
	waiterr = this.WaitFinish()

	return
}

func NewContainerRunner(api IArvadosClient,
	kc IKeepClient,
	docker ThinDockerClient) *ContainerRunner {

	cr := &ContainerRunner{Api: api, Kc: kc, Docker: docker}
	cr.NewLogWriter = cr.NewArvLogWriter
	cr.LogCollection = &CollectionWriter{kc, nil}
	cr.CrunchLog = NewThrottledLogger(cr.NewLogWriter("crunchexec"))
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
