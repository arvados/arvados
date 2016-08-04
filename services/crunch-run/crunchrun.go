package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/lib/crunchstat"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
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
	"path"
	"path/filepath"
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
	Call(method, resourceType, uuid, action string, parameters arvadosclient.Dict, output interface{}) (err error)
}

// ErrCancelled is the error returned when the container is cancelled.
var ErrCancelled = errors.New("Cancelled")

// IKeepClient is the minimal Keep API methods used by crunch-run.
type IKeepClient interface {
	PutHB(hash string, buf []byte) (string, int, error)
	ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error)
}

// NewLogWriter is a factory function to create a new log writer.
type NewLogWriter func(name string) io.WriteCloser

type RunArvMount func(args []string, tok string) (*exec.Cmd, error)

type MkTempDir func(string, string) (string, error)

// ThinDockerClient is the minimal Docker client interface used by crunch-run.
type ThinDockerClient interface {
	StopContainer(id string, timeout int) error
	InspectImage(id string) (*dockerclient.ImageInfo, error)
	LoadImage(reader io.Reader) error
	CreateContainer(config *dockerclient.ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (string, error)
	StartContainer(id string, config *dockerclient.HostConfig) error
	AttachContainer(id string, options *dockerclient.AttachOptions) (io.ReadCloser, error)
	Wait(id string) <-chan dockerclient.WaitResult
	RemoveImage(name string, force bool) ([]*dockerclient.ImageDelete, error)
}

// ContainerRunner is the main stateful struct used for a single execution of a
// container.
type ContainerRunner struct {
	Docker    ThinDockerClient
	ArvClient IArvadosClient
	Kc        IKeepClient
	arvados.Container
	dockerclient.ContainerConfig
	dockerclient.HostConfig
	token       string
	ContainerID string
	ExitCode    *int
	NewLogWriter
	loggingDone   chan bool
	CrunchLog     *ThrottledLogger
	Stdout        io.WriteCloser
	Stderr        *ThrottledLogger
	LogCollection *CollectionWriter
	LogsPDH       *string
	RunArvMount
	MkTempDir
	ArvMount       *exec.Cmd
	ArvMountPoint  string
	HostOutputDir  string
	CleanupTempDir []string
	Binds          []string
	OutputPDH      *string
	CancelLock     sync.Mutex
	Cancelled      bool
	SigChan        chan os.Signal
	ArvMountExit   chan error
	finalState     string

	statLogger   io.WriteCloser
	statReporter *crunchstat.Reporter
	statInterval time.Duration
	cgroupRoot   string
	// What we expect the container's cgroup parent to be.
	expectCgroupParent string
	// What we tell docker to use as the container's cgroup
	// parent. Note: Ideally we would use the same field for both
	// expectCgroupParent and setCgroupParent, and just make it
	// default to "docker". However, when using docker < 1.10 with
	// systemd, specifying a non-empty cgroup parent (even the
	// default value "docker") hits a docker bug
	// (https://github.com/docker/docker/issues/17126). Using two
	// separate fields makes it possible to use the "expect cgroup
	// parent to be X" feature even on sites where the "specify
	// cgroup parent" feature breaks.
	setCgroupParent string
}

// SetupSignals sets up signal handling to gracefully terminate the underlying
// Docker container and update state when receiving a TERM, INT or QUIT signal.
func (runner *ContainerRunner) SetupSignals() {
	runner.SigChan = make(chan os.Signal, 1)
	signal.Notify(runner.SigChan, syscall.SIGTERM)
	signal.Notify(runner.SigChan, syscall.SIGINT)
	signal.Notify(runner.SigChan, syscall.SIGQUIT)

	go func(sig <-chan os.Signal) {
		for range sig {
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
}

// LoadImage determines the docker image id from the container record and
// checks if it is available in the local Docker image store.  If not, it loads
// the image from Keep.
func (runner *ContainerRunner) LoadImage() (err error) {

	runner.CrunchLog.Printf("Fetching Docker image from collection '%s'", runner.Container.ContainerImage)

	var collection arvados.Collection
	err = runner.ArvClient.Get("collections", runner.Container.ContainerImage, nil, &collection)
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

func (runner *ContainerRunner) ArvMountCmd(arvMountCmd []string, token string) (c *exec.Cmd, err error) {
	c = exec.Command("arv-mount", arvMountCmd...)

	// Copy our environment, but override ARVADOS_API_TOKEN with
	// the container auth token.
	c.Env = nil
	for _, s := range os.Environ() {
		if !strings.HasPrefix(s, "ARVADOS_API_TOKEN=") {
			c.Env = append(c.Env, s)
		}
	}
	c.Env = append(c.Env, "ARVADOS_API_TOKEN="+token)

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

	runner.CleanupTempDir = append(runner.CleanupTempDir, runner.ArvMountPoint)

	pdhOnly := true
	tmpcount := 0
	arvMountCmd := []string{"--foreground", "--allow-other", "--read-write"}
	collectionPaths := []string{}
	runner.Binds = nil

	for bind, mnt := range runner.Container.Mounts {
		if bind == "stdout" {
			// Is it a "file" mount kind?
			if mnt.Kind != "file" {
				return fmt.Errorf("Unsupported mount kind '%s' for stdout. Only 'file' is supported.", mnt.Kind)
			}

			// Does path start with OutputPath?
			prefix := runner.Container.OutputPath
			if !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
			if !strings.HasPrefix(mnt.Path, prefix) {
				return fmt.Errorf("Stdout path does not start with OutputPath: %s, %s", mnt.Path, prefix)
			}
		}

		switch {
		case mnt.Kind == "collection":
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
				if bind == runner.Container.OutputPath {
					runner.HostOutputDir = src
				}
				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s", src, bind))
			} else {
				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s:ro", src, bind))
			}
			collectionPaths = append(collectionPaths, src)

		case mnt.Kind == "tmp" && bind == runner.Container.OutputPath:
			runner.HostOutputDir, err = runner.MkTempDir("", "")
			if err != nil {
				return fmt.Errorf("While creating mount temp dir: %v", err)
			}
			st, staterr := os.Stat(runner.HostOutputDir)
			if staterr != nil {
				return fmt.Errorf("While Stat on temp dir: %v", staterr)
			}
			err = os.Chmod(runner.HostOutputDir, st.Mode()|os.ModeSetgid|0777)
			if staterr != nil {
				return fmt.Errorf("While Chmod temp dir: %v", err)
			}
			runner.CleanupTempDir = append(runner.CleanupTempDir, runner.HostOutputDir)
			runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s", runner.HostOutputDir, bind))

		case mnt.Kind == "tmp":
			runner.Binds = append(runner.Binds, bind)

		case mnt.Kind == "json":
			jsondata, err := json.Marshal(mnt.Content)
			if err != nil {
				return fmt.Errorf("encoding json data: %v", err)
			}
			// Create a tempdir with a single file
			// (instead of just a tempfile): this way we
			// can ensure the file is world-readable
			// inside the container, without having to
			// make it world-readable on the docker host.
			tmpdir, err := runner.MkTempDir("", "")
			if err != nil {
				return fmt.Errorf("creating temp dir: %v", err)
			}
			runner.CleanupTempDir = append(runner.CleanupTempDir, tmpdir)
			tmpfn := filepath.Join(tmpdir, "mountdata.json")
			err = ioutil.WriteFile(tmpfn, jsondata, 0644)
			if err != nil {
				return fmt.Errorf("writing temp file: %v", err)
			}
			runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s:ro", tmpfn, bind))
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

	token, err := runner.ContainerToken()
	if err != nil {
		return fmt.Errorf("could not get container token: %s", err)
	}

	runner.ArvMount, err = runner.RunArvMount(arvMountCmd, token)
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

func (runner *ContainerRunner) ProcessDockerAttach(containerReader io.Reader) {
	// Handle docker log protocol
	// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.15/#attach-to-a-container

	header := make([]byte, 8)
	for {
		_, readerr := io.ReadAtLeast(containerReader, header, 8)

		if readerr == nil {
			readsize := int64(header[7]) | (int64(header[6]) << 8) | (int64(header[5]) << 16) | (int64(header[4]) << 24)
			if header[0] == 1 {
				// stdout
				_, readerr = io.CopyN(runner.Stdout, containerReader, readsize)
			} else {
				// stderr
				_, readerr = io.CopyN(runner.Stderr, containerReader, readsize)
			}
		}

		if readerr != nil {
			if readerr != io.EOF {
				runner.CrunchLog.Printf("While reading docker logs: %v", readerr)
			}

			closeerr := runner.Stdout.Close()
			if closeerr != nil {
				runner.CrunchLog.Printf("While closing stdout logs: %v", closeerr)
			}

			closeerr = runner.Stderr.Close()
			if closeerr != nil {
				runner.CrunchLog.Printf("While closing stderr logs: %v", closeerr)
			}

			if runner.statReporter != nil {
				runner.statReporter.Stop()
				closeerr = runner.statLogger.Close()
				if closeerr != nil {
					runner.CrunchLog.Printf("While closing crunchstat logs: %v", closeerr)
				}
			}

			runner.loggingDone <- true
			close(runner.loggingDone)
			return
		}
	}
}

func (runner *ContainerRunner) StartCrunchstat() {
	runner.statLogger = NewThrottledLogger(runner.NewLogWriter("crunchstat"))
	runner.statReporter = &crunchstat.Reporter{
		CID:          runner.ContainerID,
		Logger:       log.New(runner.statLogger, "", 0),
		CgroupParent: runner.expectCgroupParent,
		CgroupRoot:   runner.cgroupRoot,
		PollPeriod:   runner.statInterval,
	}
	runner.statReporter.Start()
}

// AttachLogs connects the docker container stdout and stderr logs to the
// Arvados logger which logs to Keep and the API server logs table.
func (runner *ContainerRunner) AttachStreams() (err error) {

	runner.CrunchLog.Print("Attaching container streams")

	var containerReader io.Reader
	containerReader, err = runner.Docker.AttachContainer(runner.ContainerID,
		&dockerclient.AttachOptions{Stream: true, Stdout: true, Stderr: true})
	if err != nil {
		return fmt.Errorf("While attaching container stdout/stderr streams: %v", err)
	}

	runner.loggingDone = make(chan bool)

	if stdoutMnt, ok := runner.Container.Mounts["stdout"]; ok {
		stdoutPath := stdoutMnt.Path[len(runner.Container.OutputPath):]
		index := strings.LastIndex(stdoutPath, "/")
		if index > 0 {
			subdirs := stdoutPath[:index]
			if subdirs != "" {
				st, err := os.Stat(runner.HostOutputDir)
				if err != nil {
					return fmt.Errorf("While Stat on temp dir: %v", err)
				}
				stdoutPath := path.Join(runner.HostOutputDir, subdirs)
				err = os.MkdirAll(stdoutPath, st.Mode()|os.ModeSetgid|0777)
				if err != nil {
					return fmt.Errorf("While MkdirAll %q: %v", stdoutPath, err)
				}
			}
		}
		stdoutFile, err := os.Create(path.Join(runner.HostOutputDir, stdoutPath))
		if err != nil {
			return fmt.Errorf("While creating stdout file: %v", err)
		}
		runner.Stdout = stdoutFile
	} else {
		runner.Stdout = NewThrottledLogger(runner.NewLogWriter("stdout"))
	}
	runner.Stderr = NewThrottledLogger(runner.NewLogWriter("stderr"))

	go runner.ProcessDockerAttach(containerReader)

	return nil
}

// CreateContainer creates the docker container.
func (runner *ContainerRunner) CreateContainer() error {
	runner.CrunchLog.Print("Creating Docker container")

	runner.ContainerConfig.Cmd = runner.Container.Command
	if runner.Container.Cwd != "." {
		runner.ContainerConfig.WorkingDir = runner.Container.Cwd
	}

	for k, v := range runner.Container.Environment {
		runner.ContainerConfig.Env = append(runner.ContainerConfig.Env, k+"="+v)
	}
	if wantAPI := runner.Container.RuntimeConstraints.API; wantAPI != nil && *wantAPI {
		tok, err := runner.ContainerToken()
		if err != nil {
			return err
		}
		runner.ContainerConfig.Env = append(runner.ContainerConfig.Env,
			"ARVADOS_API_TOKEN="+tok,
			"ARVADOS_API_HOST="+os.Getenv("ARVADOS_API_HOST"),
			"ARVADOS_API_HOST_INSECURE="+os.Getenv("ARVADOS_API_HOST_INSECURE"),
		)
		runner.ContainerConfig.NetworkDisabled = false
	} else {
		runner.ContainerConfig.NetworkDisabled = true
	}

	var err error
	runner.ContainerID, err = runner.Docker.CreateContainer(&runner.ContainerConfig, "", nil)
	if err != nil {
		return fmt.Errorf("While creating container: %v", err)
	}

	runner.HostConfig = dockerclient.HostConfig{
		Binds:        runner.Binds,
		CgroupParent: runner.setCgroupParent,
		LogConfig: dockerclient.LogConfig{
			Type: "none",
		},
	}

	return runner.AttachStreams()
}

// StartContainer starts the docker container created by CreateContainer.
func (runner *ContainerRunner) StartContainer() error {
	runner.CrunchLog.Printf("Starting Docker container id '%s'", runner.ContainerID)
	err := runner.Docker.StartContainer(runner.ContainerID, &runner.HostConfig)
	if err != nil {
		return fmt.Errorf("could not start container: %v", err)
	}
	return nil
}

// WaitFinish waits for the container to terminate, capture the exit code, and
// close the stdout/stderr logging.
func (runner *ContainerRunner) WaitFinish() error {
	runner.CrunchLog.Print("Waiting for container to finish")

	result := runner.Docker.Wait(runner.ContainerID)
	wr := <-result
	if wr.Error != nil {
		return fmt.Errorf("While waiting for container to finish: %v", wr.Error)
	}
	runner.ExitCode = &wr.ExitCode

	// wait for stdout/stderr to complete
	<-runner.loggingDone

	return nil
}

// HandleOutput sets the output, unmounts the FUSE mount, and deletes temporary directories
func (runner *ContainerRunner) CaptureOutput() error {
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

		var rec arvados.Collection
		err = json.NewDecoder(file).Decode(&rec)
		if err != nil {
			return fmt.Errorf("While reading FUSE metafile: %v", err)
		}
		manifestText = rec.ManifestText
	}

	var response arvados.Collection
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

func (runner *ContainerRunner) CleanupDirs() {
	if runner.ArvMount != nil {
		umount := exec.Command("fusermount", "-z", "-u", runner.ArvMountPoint)
		umnterr := umount.Run()
		if umnterr != nil {
			runner.CrunchLog.Printf("While running fusermount: %v", umnterr)
		}

		mnterr := <-runner.ArvMountExit
		if mnterr != nil {
			runner.CrunchLog.Printf("Arv-mount exit error: %v", mnterr)
		}
	}

	for _, tmpdir := range runner.CleanupTempDir {
		rmerr := os.RemoveAll(tmpdir)
		if rmerr != nil {
			runner.CrunchLog.Printf("While cleaning up temporary directory %s: %v", tmpdir, rmerr)
		}
	}
}

// CommitLogs posts the collection containing the final container logs.
func (runner *ContainerRunner) CommitLogs() error {
	runner.CrunchLog.Print(runner.finalState)
	runner.CrunchLog.Close()

	// Closing CrunchLog above allows it to be committed to Keep at this
	// point, but re-open crunch log with ArvClient in case there are any
	// other further (such as failing to write the log to Keep!) while
	// shutting down
	runner.CrunchLog = NewThrottledLogger(&ArvLogWriter{runner.ArvClient, runner.Container.UUID,
		"crunch-run", nil})

	if runner.LogsPDH != nil {
		// If we have already assigned something to LogsPDH,
		// we must be closing the re-opened log, which won't
		// end up getting attached to the container record and
		// therefore doesn't need to be saved as a collection
		// -- it exists only to send logs to other channels.
		return nil
	}

	mt, err := runner.LogCollection.ManifestText()
	if err != nil {
		return fmt.Errorf("While creating log manifest: %v", err)
	}

	var response arvados.Collection
	err = runner.ArvClient.Create("collections",
		arvadosclient.Dict{
			"collection": arvadosclient.Dict{
				"name":          "logs for " + runner.Container.UUID,
				"manifest_text": mt}},
		&response)
	if err != nil {
		return fmt.Errorf("While creating log collection: %v", err)
	}

	runner.LogsPDH = &response.PortableDataHash

	return nil
}

// UpdateContainerRunning updates the container state to "Running"
func (runner *ContainerRunner) UpdateContainerRunning() error {
	runner.CancelLock.Lock()
	defer runner.CancelLock.Unlock()
	if runner.Cancelled {
		return ErrCancelled
	}
	return runner.ArvClient.Update("containers", runner.Container.UUID,
		arvadosclient.Dict{"container": arvadosclient.Dict{"state": "Running"}}, nil)
}

// ContainerToken returns the api_token the container (and any
// arv-mount processes) are allowed to use.
func (runner *ContainerRunner) ContainerToken() (string, error) {
	if runner.token != "" {
		return runner.token, nil
	}

	var auth arvados.APIClientAuthorization
	err := runner.ArvClient.Call("GET", "containers", runner.Container.UUID, "auth", nil, &auth)
	if err != nil {
		return "", err
	}
	runner.token = auth.APIToken
	return runner.token, nil
}

// UpdateContainerComplete updates the container record state on API
// server to "Complete" or "Cancelled"
func (runner *ContainerRunner) UpdateContainerFinal() error {
	update := arvadosclient.Dict{}
	update["state"] = runner.finalState
	if runner.finalState == "Complete" {
		if runner.LogsPDH != nil {
			update["log"] = *runner.LogsPDH
		}
		if runner.ExitCode != nil {
			update["exit_code"] = *runner.ExitCode
		}
		if runner.OutputPDH != nil {
			update["output"] = *runner.OutputPDH
		}
	}
	return runner.ArvClient.Update("containers", runner.Container.UUID, arvadosclient.Dict{"container": update}, nil)
}

// IsCancelled returns the value of Cancelled, with goroutine safety.
func (runner *ContainerRunner) IsCancelled() bool {
	runner.CancelLock.Lock()
	defer runner.CancelLock.Unlock()
	return runner.Cancelled
}

// NewArvLogWriter creates an ArvLogWriter
func (runner *ContainerRunner) NewArvLogWriter(name string) io.WriteCloser {
	return &ArvLogWriter{runner.ArvClient, runner.Container.UUID, name, runner.LogCollection.Open(name + ".txt")}
}

// Run the full container lifecycle.
func (runner *ContainerRunner) Run() (err error) {
	runner.CrunchLog.Printf("Executing container '%s'", runner.Container.UUID)

	hostname, hosterr := os.Hostname()
	if hosterr != nil {
		runner.CrunchLog.Printf("Error getting hostname '%v'", hosterr)
	} else {
		runner.CrunchLog.Printf("Executing on host '%s'", hostname)
	}

	// Clean up temporary directories _after_ finalizing
	// everything (if we've made any by then)
	defer runner.CleanupDirs()

	runner.finalState = "Queued"

	defer func() {
		// checkErr prints e (unless it's nil) and sets err to
		// e (unless err is already non-nil). Thus, if err
		// hasn't already been assigned when Run() returns,
		// this cleanup func will cause Run() to return the
		// first non-nil error that is passed to checkErr().
		checkErr := func(e error) {
			if e == nil {
				return
			}
			runner.CrunchLog.Print(e)
			if err == nil {
				err = e
			}
		}

		// Log the error encountered in Run(), if any
		checkErr(err)

		if runner.finalState == "Queued" {
			runner.UpdateContainerFinal()
			return
		}

		if runner.IsCancelled() {
			runner.finalState = "Cancelled"
			// but don't return yet -- we still want to
			// capture partial output and write logs
		}

		checkErr(runner.CaptureOutput())
		checkErr(runner.CommitLogs())
		checkErr(runner.UpdateContainerFinal())

		// The real log is already closed, but then we opened
		// a new one in case we needed to log anything while
		// finalizing.
		runner.CrunchLog.Close()
	}()

	err = runner.ArvClient.Get("containers", runner.Container.UUID, nil, &runner.Container)
	if err != nil {
		err = fmt.Errorf("While getting container record: %v", err)
		return
	}

	// setup signal handling
	runner.SetupSignals()

	// check for and/or load image
	err = runner.LoadImage()
	if err != nil {
		err = fmt.Errorf("While loading container image: %v", err)
		return
	}

	// set up FUSE mount and binds
	err = runner.SetupMounts()
	if err != nil {
		err = fmt.Errorf("While setting up mounts: %v", err)
		return
	}

	err = runner.CreateContainer()
	if err != nil {
		return
	}

	runner.StartCrunchstat()

	if runner.IsCancelled() {
		return
	}

	err = runner.UpdateContainerRunning()
	if err != nil {
		return
	}
	runner.finalState = "Cancelled"

	err = runner.StartContainer()
	if err != nil {
		return
	}

	err = runner.WaitFinish()
	if err == nil {
		runner.finalState = "Complete"
	}
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
	cr.Container.UUID = containerUUID
	cr.CrunchLog = NewThrottledLogger(cr.NewLogWriter("crunch-run"))
	cr.CrunchLog.Immediate = log.New(os.Stderr, containerUUID+" ", 0)
	return cr
}

func main() {
	statInterval := flag.Duration("crunchstat-interval", 10*time.Second, "sampling period for periodic resource usage reporting")
	cgroupRoot := flag.String("cgroup-root", "/sys/fs/cgroup", "path to sysfs cgroup tree")
	cgroupParent := flag.String("cgroup-parent", "docker", "name of container's parent cgroup (ignored if -cgroup-parent-subsystem is used)")
	cgroupParentSubsystem := flag.String("cgroup-parent-subsystem", "", "use current cgroup for given subsystem as parent cgroup for container")
	flag.Parse()

	containerId := flag.Arg(0)

	api, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("%s: %v", containerId, err)
	}
	api.Retries = 8

	var kc *keepclient.KeepClient
	kc, err = keepclient.MakeKeepClient(&api)
	if err != nil {
		log.Fatalf("%s: %v", containerId, err)
	}
	kc.Retries = 4

	var docker *dockerclient.DockerClient
	docker, err = dockerclient.NewDockerClient("unix:///var/run/docker.sock", nil)
	if err != nil {
		log.Fatalf("%s: %v", containerId, err)
	}

	cr := NewContainerRunner(api, kc, docker, containerId)
	cr.statInterval = *statInterval
	cr.cgroupRoot = *cgroupRoot
	cr.expectCgroupParent = *cgroupParent
	if *cgroupParentSubsystem != "" {
		p := findCgroup(*cgroupParentSubsystem)
		cr.setCgroupParent = p
		cr.expectCgroupParent = p
	}

	err = cr.Run()
	if err != nil {
		log.Fatalf("%s: %v", containerId, err)
	}

}
