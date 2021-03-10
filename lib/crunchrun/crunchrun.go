// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/crunchstat"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"git.arvados.org/arvados.git/sdk/go/manifest"
	"golang.org/x/net/context"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockernetwork "github.com/docker/docker/api/types/network"
	dockerclient "github.com/docker/docker/client"
)

type command struct{}

var Command = command{}

// IArvadosClient is the minimal Arvados API methods used by crunch-run.
type IArvadosClient interface {
	Create(resourceType string, parameters arvadosclient.Dict, output interface{}) error
	Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error
	Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error
	Call(method, resourceType, uuid, action string, parameters arvadosclient.Dict, output interface{}) error
	CallRaw(method string, resourceType string, uuid string, action string, parameters arvadosclient.Dict) (reader io.ReadCloser, err error)
	Discovery(key string) (interface{}, error)
}

// ErrCancelled is the error returned when the container is cancelled.
var ErrCancelled = errors.New("Cancelled")

// IKeepClient is the minimal Keep API methods used by crunch-run.
type IKeepClient interface {
	PutB(buf []byte) (string, int, error)
	ReadAt(locator string, p []byte, off int) (int, error)
	ManifestFileReader(m manifest.Manifest, filename string) (arvados.File, error)
	LocalLocator(locator string) (string, error)
	ClearBlockCache()
}

// NewLogWriter is a factory function to create a new log writer.
type NewLogWriter func(name string) (io.WriteCloser, error)

type RunArvMount func(args []string, tok string) (*exec.Cmd, error)

type MkTempDir func(string, string) (string, error)

// ThinDockerClient is the minimal Docker client interface used by crunch-run.
type ThinDockerClient interface {
	ContainerAttach(ctx context.Context, container string, options dockertypes.ContainerAttachOptions) (dockertypes.HijackedResponse, error)
	ContainerCreate(ctx context.Context, config *dockercontainer.Config, hostConfig *dockercontainer.HostConfig,
		networkingConfig *dockernetwork.NetworkingConfig, containerName string) (dockercontainer.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, container string, options dockertypes.ContainerStartOptions) error
	ContainerRemove(ctx context.Context, container string, options dockertypes.ContainerRemoveOptions) error
	ContainerWait(ctx context.Context, container string, condition dockercontainer.WaitCondition) (<-chan dockercontainer.ContainerWaitOKBody, <-chan error)
	ContainerInspect(ctx context.Context, id string) (dockertypes.ContainerJSON, error)
	ImageInspectWithRaw(ctx context.Context, image string) (dockertypes.ImageInspect, []byte, error)
	ImageLoad(ctx context.Context, input io.Reader, quiet bool) (dockertypes.ImageLoadResponse, error)
	ImageRemove(ctx context.Context, image string, options dockertypes.ImageRemoveOptions) ([]dockertypes.ImageDeleteResponseItem, error)
}

type PsProcess interface {
	CmdlineSlice() ([]string, error)
}

// ContainerRunner is the main stateful struct used for a single execution of a
// container.
type ContainerRunner struct {
	Docker ThinDockerClient

	// Dispatcher client is initialized with the Dispatcher token.
	// This is a privileged token used to manage container status
	// and logs.
	//
	// We have both dispatcherClient and DispatcherArvClient
	// because there are two different incompatible Arvados Go
	// SDKs and we have to use both (hopefully this gets fixed in
	// #14467)
	dispatcherClient     *arvados.Client
	DispatcherArvClient  IArvadosClient
	DispatcherKeepClient IKeepClient

	// Container client is initialized with the Container token
	// This token controls the permissions of the container, and
	// must be used for operations such as reading collections.
	//
	// Same comment as above applies to
	// containerClient/ContainerArvClient.
	containerClient     *arvados.Client
	ContainerArvClient  IArvadosClient
	ContainerKeepClient IKeepClient

	Container       arvados.Container
	ContainerConfig dockercontainer.Config
	HostConfig      dockercontainer.HostConfig
	token           string
	ContainerID     string
	ExitCode        *int
	NewLogWriter    NewLogWriter
	loggingDone     chan bool
	CrunchLog       *ThrottledLogger
	Stdout          io.WriteCloser
	Stderr          io.WriteCloser
	logUUID         string
	logMtx          sync.Mutex
	LogCollection   arvados.CollectionFileSystem
	LogsPDH         *string
	RunArvMount     RunArvMount
	MkTempDir       MkTempDir
	ArvMount        *exec.Cmd
	ArvMountPoint   string
	HostOutputDir   string
	Binds           []string
	Volumes         map[string]struct{}
	OutputPDH       *string
	SigChan         chan os.Signal
	ArvMountExit    chan error
	SecretMounts    map[string]arvados.Mount
	MkArvClient     func(token string) (IArvadosClient, IKeepClient, *arvados.Client, error)
	finalState      string
	parentTemp      string

	statLogger       io.WriteCloser
	statReporter     *crunchstat.Reporter
	hoststatLogger   io.WriteCloser
	hoststatReporter *crunchstat.Reporter
	statInterval     time.Duration
	cgroupRoot       string
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

	cStateLock sync.Mutex
	cCancelled bool // StopContainer() invoked
	cRemoved   bool // docker confirmed the container no longer exists

	enableNetwork string // one of "default" or "always"
	networkMode   string // passed through to HostConfig.NetworkMode
	arvMountLog   *ThrottledLogger

	containerWatchdogInterval time.Duration

	gateway Gateway
}

// setupSignals sets up signal handling to gracefully terminate the underlying
// Docker container and update state when receiving a TERM, INT or QUIT signal.
func (runner *ContainerRunner) setupSignals() {
	runner.SigChan = make(chan os.Signal, 1)
	signal.Notify(runner.SigChan, syscall.SIGTERM)
	signal.Notify(runner.SigChan, syscall.SIGINT)
	signal.Notify(runner.SigChan, syscall.SIGQUIT)

	go func(sig chan os.Signal) {
		for s := range sig {
			runner.stop(s)
		}
	}(runner.SigChan)
}

// stop the underlying Docker container.
func (runner *ContainerRunner) stop(sig os.Signal) {
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	if sig != nil {
		runner.CrunchLog.Printf("caught signal: %v", sig)
	}
	if runner.ContainerID == "" {
		return
	}
	runner.cCancelled = true
	runner.CrunchLog.Printf("removing container")
	err := runner.Docker.ContainerRemove(context.TODO(), runner.ContainerID, dockertypes.ContainerRemoveOptions{Force: true})
	if err != nil {
		runner.CrunchLog.Printf("error removing container: %s", err)
	}
	if err == nil || strings.Contains(err.Error(), "No such container: "+runner.ContainerID) {
		runner.cRemoved = true
	}
}

var errorBlacklist = []string{
	"(?ms).*[Cc]annot connect to the Docker daemon.*",
	"(?ms).*oci runtime error.*starting container process.*container init.*mounting.*to rootfs.*no such file or directory.*",
	"(?ms).*grpc: the connection is unavailable.*",
}
var brokenNodeHook *string = flag.String("broken-node-hook", "", "Script to run if node is detected to be broken (for example, Docker daemon is not running)")

func (runner *ContainerRunner) runBrokenNodeHook() {
	if *brokenNodeHook == "" {
		path := filepath.Join(lockdir, brokenfile)
		runner.CrunchLog.Printf("Writing %s to mark node as broken", path)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0700)
		if err != nil {
			runner.CrunchLog.Printf("Error writing %s: %s", path, err)
			return
		}
		f.Close()
	} else {
		runner.CrunchLog.Printf("Running broken node hook %q", *brokenNodeHook)
		// run killme script
		c := exec.Command(*brokenNodeHook)
		c.Stdout = runner.CrunchLog
		c.Stderr = runner.CrunchLog
		err := c.Run()
		if err != nil {
			runner.CrunchLog.Printf("Error running broken node hook: %v", err)
		}
	}
}

func (runner *ContainerRunner) checkBrokenNode(goterr error) bool {
	for _, d := range errorBlacklist {
		if m, e := regexp.MatchString(d, goterr.Error()); m && e == nil {
			runner.CrunchLog.Printf("Error suggests node is unable to run containers: %v", goterr)
			runner.runBrokenNodeHook()
			return true
		}
	}
	return false
}

// LoadImage determines the docker image id from the container record and
// checks if it is available in the local Docker image store.  If not, it loads
// the image from Keep.
func (runner *ContainerRunner) LoadImage() (err error) {

	runner.CrunchLog.Printf("Fetching Docker image from collection '%s'", runner.Container.ContainerImage)

	var collection arvados.Collection
	err = runner.ContainerArvClient.Get("collections", runner.Container.ContainerImage, nil, &collection)
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

	_, _, err = runner.Docker.ImageInspectWithRaw(context.TODO(), imageID)
	if err != nil {
		runner.CrunchLog.Print("Loading Docker image from keep")

		var readCloser io.ReadCloser
		readCloser, err = runner.ContainerKeepClient.ManifestFileReader(manifest, img)
		if err != nil {
			return fmt.Errorf("While creating ManifestFileReader for container image: %v", err)
		}

		response, err := runner.Docker.ImageLoad(context.TODO(), readCloser, true)
		if err != nil {
			return fmt.Errorf("While loading container image into Docker: %v", err)
		}

		defer response.Body.Close()
		rbody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("Reading response to image load: %v", err)
		}
		runner.CrunchLog.Printf("Docker response: %s", rbody)
	} else {
		runner.CrunchLog.Print("Docker image is available")
	}

	runner.ContainerConfig.Image = imageID

	runner.ContainerKeepClient.ClearBlockCache()

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

	w, err := runner.NewLogWriter("arv-mount")
	if err != nil {
		return nil, err
	}
	runner.arvMountLog = NewThrottledLogger(w)
	c.Stdout = runner.arvMountLog
	c.Stderr = runner.arvMountLog

	runner.CrunchLog.Printf("Running %v", c.Args)

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
		mnterr := c.Wait()
		if mnterr != nil {
			runner.CrunchLog.Printf("Arv-mount exit error: %v", mnterr)
		}
		runner.ArvMountExit <- mnterr
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

func (runner *ContainerRunner) SetupArvMountPoint(prefix string) (err error) {
	if runner.ArvMountPoint == "" {
		runner.ArvMountPoint, err = runner.MkTempDir(runner.parentTemp, prefix)
	}
	return
}

func copyfile(src string, dst string) (err error) {
	srcfile, err := os.Open(src)
	if err != nil {
		return
	}

	os.MkdirAll(path.Dir(dst), 0777)

	dstfile, err := os.Create(dst)
	if err != nil {
		return
	}
	_, err = io.Copy(dstfile, srcfile)
	if err != nil {
		return
	}

	err = srcfile.Close()
	err2 := dstfile.Close()

	if err != nil {
		return
	}

	if err2 != nil {
		return err2
	}

	return nil
}

func (runner *ContainerRunner) SetupMounts() (err error) {
	err = runner.SetupArvMountPoint("keep")
	if err != nil {
		return fmt.Errorf("While creating keep mount temp dir: %v", err)
	}

	token, err := runner.ContainerToken()
	if err != nil {
		return fmt.Errorf("could not get container token: %s", err)
	}

	pdhOnly := true
	tmpcount := 0
	arvMountCmd := []string{
		"--foreground",
		"--allow-other",
		"--read-write",
		fmt.Sprintf("--crunchstat-interval=%v", runner.statInterval.Seconds())}

	if runner.Container.RuntimeConstraints.KeepCacheRAM > 0 {
		arvMountCmd = append(arvMountCmd, "--file-cache", fmt.Sprintf("%d", runner.Container.RuntimeConstraints.KeepCacheRAM))
	}

	collectionPaths := []string{}
	runner.Binds = nil
	runner.Volumes = make(map[string]struct{})
	needCertMount := true
	type copyFile struct {
		src  string
		bind string
	}
	var copyFiles []copyFile

	var binds []string
	for bind := range runner.Container.Mounts {
		binds = append(binds, bind)
	}
	for bind := range runner.SecretMounts {
		if _, ok := runner.Container.Mounts[bind]; ok {
			return fmt.Errorf("secret mount %q conflicts with regular mount", bind)
		}
		if runner.SecretMounts[bind].Kind != "json" &&
			runner.SecretMounts[bind].Kind != "text" {
			return fmt.Errorf("secret mount %q type is %q but only 'json' and 'text' are permitted",
				bind, runner.SecretMounts[bind].Kind)
		}
		binds = append(binds, bind)
	}
	sort.Strings(binds)

	for _, bind := range binds {
		mnt, ok := runner.Container.Mounts[bind]
		if !ok {
			mnt = runner.SecretMounts[bind]
		}
		if bind == "stdout" || bind == "stderr" {
			// Is it a "file" mount kind?
			if mnt.Kind != "file" {
				return fmt.Errorf("unsupported mount kind '%s' for %s: only 'file' is supported", mnt.Kind, bind)
			}

			// Does path start with OutputPath?
			prefix := runner.Container.OutputPath
			if !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
			if !strings.HasPrefix(mnt.Path, prefix) {
				return fmt.Errorf("%s path does not start with OutputPath: %s, %s", strings.Title(bind), mnt.Path, prefix)
			}
		}

		if bind == "stdin" {
			// Is it a "collection" mount kind?
			if mnt.Kind != "collection" && mnt.Kind != "json" {
				return fmt.Errorf("unsupported mount kind '%s' for stdin: only 'collection' and 'json' are supported", mnt.Kind)
			}
		}

		if bind == "/etc/arvados/ca-certificates.crt" {
			needCertMount = false
		}

		if strings.HasPrefix(bind, runner.Container.OutputPath+"/") && bind != runner.Container.OutputPath+"/" {
			if mnt.Kind != "collection" && mnt.Kind != "text" && mnt.Kind != "json" {
				return fmt.Errorf("only mount points of kind 'collection', 'text' or 'json' are supported underneath the output_path for %q, was %q", bind, mnt.Kind)
			}
		}

		switch {
		case mnt.Kind == "collection" && bind != "stdin":
			var src string
			if mnt.UUID != "" && mnt.PortableDataHash != "" {
				return fmt.Errorf("cannot specify both 'uuid' and 'portable_data_hash' for a collection mount")
			}
			if mnt.UUID != "" {
				if mnt.Writable {
					return fmt.Errorf("writing to existing collections currently not permitted")
				}
				pdhOnly = false
				src = fmt.Sprintf("%s/by_id/%s", runner.ArvMountPoint, mnt.UUID)
			} else if mnt.PortableDataHash != "" {
				if mnt.Writable && !strings.HasPrefix(bind, runner.Container.OutputPath+"/") {
					return fmt.Errorf("can never write to a collection specified by portable data hash")
				}
				idx := strings.Index(mnt.PortableDataHash, "/")
				if idx > 0 {
					mnt.Path = path.Clean(mnt.PortableDataHash[idx:])
					mnt.PortableDataHash = mnt.PortableDataHash[0:idx]
					runner.Container.Mounts[bind] = mnt
				}
				src = fmt.Sprintf("%s/by_id/%s", runner.ArvMountPoint, mnt.PortableDataHash)
				if mnt.Path != "" && mnt.Path != "." {
					if strings.HasPrefix(mnt.Path, "./") {
						mnt.Path = mnt.Path[2:]
					} else if strings.HasPrefix(mnt.Path, "/") {
						mnt.Path = mnt.Path[1:]
					}
					src += "/" + mnt.Path
				}
			} else {
				src = fmt.Sprintf("%s/tmp%d", runner.ArvMountPoint, tmpcount)
				arvMountCmd = append(arvMountCmd, "--mount-tmp")
				arvMountCmd = append(arvMountCmd, fmt.Sprintf("tmp%d", tmpcount))
				tmpcount++
			}
			if mnt.Writable {
				if bind == runner.Container.OutputPath {
					runner.HostOutputDir = src
					runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s", src, bind))
				} else if strings.HasPrefix(bind, runner.Container.OutputPath+"/") {
					copyFiles = append(copyFiles, copyFile{src, runner.HostOutputDir + bind[len(runner.Container.OutputPath):]})
				} else {
					runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s", src, bind))
				}
			} else {
				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s:ro", src, bind))
			}
			collectionPaths = append(collectionPaths, src)

		case mnt.Kind == "tmp":
			var tmpdir string
			tmpdir, err = runner.MkTempDir(runner.parentTemp, "tmp")
			if err != nil {
				return fmt.Errorf("while creating mount temp dir: %v", err)
			}
			st, staterr := os.Stat(tmpdir)
			if staterr != nil {
				return fmt.Errorf("while Stat on temp dir: %v", staterr)
			}
			err = os.Chmod(tmpdir, st.Mode()|os.ModeSetgid|0777)
			if staterr != nil {
				return fmt.Errorf("while Chmod temp dir: %v", err)
			}
			runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s", tmpdir, bind))
			if bind == runner.Container.OutputPath {
				runner.HostOutputDir = tmpdir
			}

		case mnt.Kind == "json" || mnt.Kind == "text":
			var filedata []byte
			if mnt.Kind == "json" {
				filedata, err = json.Marshal(mnt.Content)
				if err != nil {
					return fmt.Errorf("encoding json data: %v", err)
				}
			} else {
				text, ok := mnt.Content.(string)
				if !ok {
					return fmt.Errorf("content for mount %q must be a string", bind)
				}
				filedata = []byte(text)
			}

			tmpdir, err := runner.MkTempDir(runner.parentTemp, mnt.Kind)
			if err != nil {
				return fmt.Errorf("creating temp dir: %v", err)
			}
			tmpfn := filepath.Join(tmpdir, "mountdata."+mnt.Kind)
			err = ioutil.WriteFile(tmpfn, filedata, 0444)
			if err != nil {
				return fmt.Errorf("writing temp file: %v", err)
			}
			if strings.HasPrefix(bind, runner.Container.OutputPath+"/") {
				copyFiles = append(copyFiles, copyFile{tmpfn, runner.HostOutputDir + bind[len(runner.Container.OutputPath):]})
			} else {
				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:%s:ro", tmpfn, bind))
			}

		case mnt.Kind == "git_tree":
			tmpdir, err := runner.MkTempDir(runner.parentTemp, "git_tree")
			if err != nil {
				return fmt.Errorf("creating temp dir: %v", err)
			}
			err = gitMount(mnt).extractTree(runner.ContainerArvClient, tmpdir, token)
			if err != nil {
				return err
			}
			runner.Binds = append(runner.Binds, tmpdir+":"+bind+":ro")
		}
	}

	if runner.HostOutputDir == "" {
		return fmt.Errorf("output path does not correspond to a writable mount point")
	}

	if needCertMount && runner.Container.RuntimeConstraints.API {
		for _, certfile := range arvadosclient.CertFiles {
			_, err := os.Stat(certfile)
			if err == nil {
				runner.Binds = append(runner.Binds, fmt.Sprintf("%s:/etc/arvados/ca-certificates.crt:ro", certfile))
				break
			}
		}
	}

	if pdhOnly {
		arvMountCmd = append(arvMountCmd, "--mount-by-pdh", "by_id")
	} else {
		arvMountCmd = append(arvMountCmd, "--mount-by-id", "by_id")
	}
	arvMountCmd = append(arvMountCmd, runner.ArvMountPoint)

	runner.ArvMount, err = runner.RunArvMount(arvMountCmd, token)
	if err != nil {
		return fmt.Errorf("while trying to start arv-mount: %v", err)
	}

	for _, p := range collectionPaths {
		_, err = os.Stat(p)
		if err != nil {
			return fmt.Errorf("while checking that input files exist: %v", err)
		}
	}

	for _, cp := range copyFiles {
		st, err := os.Stat(cp.src)
		if err != nil {
			return fmt.Errorf("while staging writable file from %q to %q: %v", cp.src, cp.bind, err)
		}
		if st.IsDir() {
			err = filepath.Walk(cp.src, func(walkpath string, walkinfo os.FileInfo, walkerr error) error {
				if walkerr != nil {
					return walkerr
				}
				target := path.Join(cp.bind, walkpath[len(cp.src):])
				if walkinfo.Mode().IsRegular() {
					copyerr := copyfile(walkpath, target)
					if copyerr != nil {
						return copyerr
					}
					return os.Chmod(target, walkinfo.Mode()|0777)
				} else if walkinfo.Mode().IsDir() {
					mkerr := os.MkdirAll(target, 0777)
					if mkerr != nil {
						return mkerr
					}
					return os.Chmod(target, walkinfo.Mode()|os.ModeSetgid|0777)
				} else {
					return fmt.Errorf("source %q is not a regular file or directory", cp.src)
				}
			})
		} else if st.Mode().IsRegular() {
			err = copyfile(cp.src, cp.bind)
			if err == nil {
				err = os.Chmod(cp.bind, st.Mode()|0777)
			}
		}
		if err != nil {
			return fmt.Errorf("while staging writable file from %q to %q: %v", cp.src, cp.bind, err)
		}
	}

	return nil
}

func (runner *ContainerRunner) ProcessDockerAttach(containerReader io.Reader) {
	// Handle docker log protocol
	// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.15/#attach-to-a-container
	defer close(runner.loggingDone)

	header := make([]byte, 8)
	var err error
	for err == nil {
		_, err = io.ReadAtLeast(containerReader, header, 8)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		readsize := int64(header[7]) | (int64(header[6]) << 8) | (int64(header[5]) << 16) | (int64(header[4]) << 24)
		if header[0] == 1 {
			// stdout
			_, err = io.CopyN(runner.Stdout, containerReader, readsize)
		} else {
			// stderr
			_, err = io.CopyN(runner.Stderr, containerReader, readsize)
		}
	}

	if err != nil {
		runner.CrunchLog.Printf("error reading docker logs: %v", err)
	}

	err = runner.Stdout.Close()
	if err != nil {
		runner.CrunchLog.Printf("error closing stdout logs: %v", err)
	}

	err = runner.Stderr.Close()
	if err != nil {
		runner.CrunchLog.Printf("error closing stderr logs: %v", err)
	}

	if runner.statReporter != nil {
		runner.statReporter.Stop()
		err = runner.statLogger.Close()
		if err != nil {
			runner.CrunchLog.Printf("error closing crunchstat logs: %v", err)
		}
	}
}

func (runner *ContainerRunner) stopHoststat() error {
	if runner.hoststatReporter == nil {
		return nil
	}
	runner.hoststatReporter.Stop()
	err := runner.hoststatLogger.Close()
	if err != nil {
		return fmt.Errorf("error closing hoststat logs: %v", err)
	}
	return nil
}

func (runner *ContainerRunner) startHoststat() error {
	w, err := runner.NewLogWriter("hoststat")
	if err != nil {
		return err
	}
	runner.hoststatLogger = NewThrottledLogger(w)
	runner.hoststatReporter = &crunchstat.Reporter{
		Logger:     log.New(runner.hoststatLogger, "", 0),
		CgroupRoot: runner.cgroupRoot,
		PollPeriod: runner.statInterval,
	}
	runner.hoststatReporter.Start()
	return nil
}

func (runner *ContainerRunner) startCrunchstat() error {
	w, err := runner.NewLogWriter("crunchstat")
	if err != nil {
		return err
	}
	runner.statLogger = NewThrottledLogger(w)
	runner.statReporter = &crunchstat.Reporter{
		CID:          runner.ContainerID,
		Logger:       log.New(runner.statLogger, "", 0),
		CgroupParent: runner.expectCgroupParent,
		CgroupRoot:   runner.cgroupRoot,
		PollPeriod:   runner.statInterval,
		TempDir:      runner.parentTemp,
	}
	runner.statReporter.Start()
	return nil
}

type infoCommand struct {
	label string
	cmd   []string
}

// LogHostInfo logs info about the current host, for debugging and
// accounting purposes. Although it's logged as "node-info", this is
// about the environment where crunch-run is actually running, which
// might differ from what's described in the node record (see
// LogNodeRecord).
func (runner *ContainerRunner) LogHostInfo() (err error) {
	w, err := runner.NewLogWriter("node-info")
	if err != nil {
		return
	}

	commands := []infoCommand{
		{
			label: "Host Information",
			cmd:   []string{"uname", "-a"},
		},
		{
			label: "CPU Information",
			cmd:   []string{"cat", "/proc/cpuinfo"},
		},
		{
			label: "Memory Information",
			cmd:   []string{"cat", "/proc/meminfo"},
		},
		{
			label: "Disk Space",
			cmd:   []string{"df", "-m", "/", os.TempDir()},
		},
		{
			label: "Disk INodes",
			cmd:   []string{"df", "-i", "/", os.TempDir()},
		},
	}

	// Run commands with informational output to be logged.
	for _, command := range commands {
		fmt.Fprintln(w, command.label)
		cmd := exec.Command(command.cmd[0], command.cmd[1:]...)
		cmd.Stdout = w
		cmd.Stderr = w
		if err := cmd.Run(); err != nil {
			err = fmt.Errorf("While running command %q: %v", command.cmd, err)
			fmt.Fprintln(w, err)
			return err
		}
		fmt.Fprintln(w, "")
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("While closing node-info logs: %v", err)
	}
	return nil
}

// LogContainerRecord gets and saves the raw JSON container record from the API server
func (runner *ContainerRunner) LogContainerRecord() error {
	logged, err := runner.logAPIResponse("container", "containers", map[string]interface{}{"filters": [][]string{{"uuid", "=", runner.Container.UUID}}}, nil)
	if !logged && err == nil {
		err = fmt.Errorf("error: no container record found for %s", runner.Container.UUID)
	}
	return err
}

// LogNodeRecord logs the current host's InstanceType config entry (or
// the arvados#node record, if running via crunch-dispatch-slurm).
func (runner *ContainerRunner) LogNodeRecord() error {
	if it := os.Getenv("InstanceType"); it != "" {
		// Dispatched via arvados-dispatch-cloud. Save
		// InstanceType config fragment received from
		// dispatcher on stdin.
		w, err := runner.LogCollection.OpenFile("node.json", os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer w.Close()
		_, err = io.WriteString(w, it)
		if err != nil {
			return err
		}
		return w.Close()
	}
	// Dispatched via crunch-dispatch-slurm. Look up
	// apiserver's node record corresponding to
	// $SLURMD_NODENAME.
	hostname := os.Getenv("SLURMD_NODENAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	_, err := runner.logAPIResponse("node", "nodes", map[string]interface{}{"filters": [][]string{{"hostname", "=", hostname}}}, func(resp interface{}) {
		// The "info" field has admin-only info when
		// obtained with a privileged token, and
		// should not be logged.
		node, ok := resp.(map[string]interface{})
		if ok {
			delete(node, "info")
		}
	})
	return err
}

func (runner *ContainerRunner) logAPIResponse(label, path string, params map[string]interface{}, munge func(interface{})) (logged bool, err error) {
	writer, err := runner.LogCollection.OpenFile(label+".json", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return false, err
	}
	w := &ArvLogWriter{
		ArvClient:     runner.DispatcherArvClient,
		UUID:          runner.Container.UUID,
		loggingStream: label,
		writeCloser:   writer,
	}

	reader, err := runner.DispatcherArvClient.CallRaw("GET", path, "", "", arvadosclient.Dict(params))
	if err != nil {
		return false, fmt.Errorf("error getting %s record: %v", label, err)
	}
	defer reader.Close()

	dec := json.NewDecoder(reader)
	dec.UseNumber()
	var resp map[string]interface{}
	if err = dec.Decode(&resp); err != nil {
		return false, fmt.Errorf("error decoding %s list response: %v", label, err)
	}
	items, ok := resp["items"].([]interface{})
	if !ok {
		return false, fmt.Errorf("error decoding %s list response: no \"items\" key in API list response", label)
	} else if len(items) < 1 {
		return false, nil
	}
	if munge != nil {
		munge(items[0])
	}
	// Re-encode it using indentation to improve readability
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	if err = enc.Encode(items[0]); err != nil {
		return false, fmt.Errorf("error logging %s record: %v", label, err)
	}
	err = w.Close()
	if err != nil {
		return false, fmt.Errorf("error closing %s.json in log collection: %v", label, err)
	}
	return true, nil
}

// AttachStreams connects the docker container stdin, stdout and stderr logs
// to the Arvados logger which logs to Keep and the API server logs table.
func (runner *ContainerRunner) AttachStreams() (err error) {

	runner.CrunchLog.Print("Attaching container streams")

	// If stdin mount is provided, attach it to the docker container
	var stdinRdr arvados.File
	var stdinJSON []byte
	if stdinMnt, ok := runner.Container.Mounts["stdin"]; ok {
		if stdinMnt.Kind == "collection" {
			var stdinColl arvados.Collection
			collID := stdinMnt.UUID
			if collID == "" {
				collID = stdinMnt.PortableDataHash
			}
			err = runner.ContainerArvClient.Get("collections", collID, nil, &stdinColl)
			if err != nil {
				return fmt.Errorf("While getting stdin collection: %v", err)
			}

			stdinRdr, err = runner.ContainerKeepClient.ManifestFileReader(
				manifest.Manifest{Text: stdinColl.ManifestText},
				stdinMnt.Path)
			if os.IsNotExist(err) {
				return fmt.Errorf("stdin collection path not found: %v", stdinMnt.Path)
			} else if err != nil {
				return fmt.Errorf("While getting stdin collection path %v: %v", stdinMnt.Path, err)
			}
		} else if stdinMnt.Kind == "json" {
			stdinJSON, err = json.Marshal(stdinMnt.Content)
			if err != nil {
				return fmt.Errorf("While encoding stdin json data: %v", err)
			}
		}
	}

	stdinUsed := stdinRdr != nil || len(stdinJSON) != 0
	response, err := runner.Docker.ContainerAttach(context.TODO(), runner.ContainerID,
		dockertypes.ContainerAttachOptions{Stream: true, Stdin: stdinUsed, Stdout: true, Stderr: true})
	if err != nil {
		return fmt.Errorf("While attaching container stdout/stderr streams: %v", err)
	}

	runner.loggingDone = make(chan bool)

	if stdoutMnt, ok := runner.Container.Mounts["stdout"]; ok {
		stdoutFile, err := runner.getStdoutFile(stdoutMnt.Path)
		if err != nil {
			return err
		}
		runner.Stdout = stdoutFile
	} else if w, err := runner.NewLogWriter("stdout"); err != nil {
		return err
	} else {
		runner.Stdout = NewThrottledLogger(w)
	}

	if stderrMnt, ok := runner.Container.Mounts["stderr"]; ok {
		stderrFile, err := runner.getStdoutFile(stderrMnt.Path)
		if err != nil {
			return err
		}
		runner.Stderr = stderrFile
	} else if w, err := runner.NewLogWriter("stderr"); err != nil {
		return err
	} else {
		runner.Stderr = NewThrottledLogger(w)
	}

	if stdinRdr != nil {
		go func() {
			_, err := io.Copy(response.Conn, stdinRdr)
			if err != nil {
				runner.CrunchLog.Printf("While writing stdin collection to docker container: %v", err)
				runner.stop(nil)
			}
			stdinRdr.Close()
			response.CloseWrite()
		}()
	} else if len(stdinJSON) != 0 {
		go func() {
			_, err := io.Copy(response.Conn, bytes.NewReader(stdinJSON))
			if err != nil {
				runner.CrunchLog.Printf("While writing stdin json to docker container: %v", err)
				runner.stop(nil)
			}
			response.CloseWrite()
		}()
	}

	go runner.ProcessDockerAttach(response.Reader)

	return nil
}

func (runner *ContainerRunner) getStdoutFile(mntPath string) (*os.File, error) {
	stdoutPath := mntPath[len(runner.Container.OutputPath):]
	index := strings.LastIndex(stdoutPath, "/")
	if index > 0 {
		subdirs := stdoutPath[:index]
		if subdirs != "" {
			st, err := os.Stat(runner.HostOutputDir)
			if err != nil {
				return nil, fmt.Errorf("While Stat on temp dir: %v", err)
			}
			stdoutPath := filepath.Join(runner.HostOutputDir, subdirs)
			err = os.MkdirAll(stdoutPath, st.Mode()|os.ModeSetgid|0777)
			if err != nil {
				return nil, fmt.Errorf("While MkdirAll %q: %v", stdoutPath, err)
			}
		}
	}
	stdoutFile, err := os.Create(filepath.Join(runner.HostOutputDir, stdoutPath))
	if err != nil {
		return nil, fmt.Errorf("While creating file %q: %v", stdoutPath, err)
	}

	return stdoutFile, nil
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

	runner.ContainerConfig.Volumes = runner.Volumes

	maxRAM := int64(runner.Container.RuntimeConstraints.RAM)
	minDockerRAM := int64(16)
	if maxRAM < minDockerRAM*1024*1024 {
		// Docker daemon won't let you set a limit less than ~10 MiB
		maxRAM = minDockerRAM * 1024 * 1024
	}
	runner.HostConfig = dockercontainer.HostConfig{
		Binds: runner.Binds,
		LogConfig: dockercontainer.LogConfig{
			Type: "none",
		},
		Resources: dockercontainer.Resources{
			CgroupParent: runner.setCgroupParent,
			NanoCPUs:     int64(runner.Container.RuntimeConstraints.VCPUs) * 1000000000,
			Memory:       maxRAM, // RAM
			MemorySwap:   maxRAM, // RAM+swap
			KernelMemory: maxRAM, // kernel portion
		},
	}

	if runner.Container.RuntimeConstraints.API {
		tok, err := runner.ContainerToken()
		if err != nil {
			return err
		}
		runner.ContainerConfig.Env = append(runner.ContainerConfig.Env,
			"ARVADOS_API_TOKEN="+tok,
			"ARVADOS_API_HOST="+os.Getenv("ARVADOS_API_HOST"),
			"ARVADOS_API_HOST_INSECURE="+os.Getenv("ARVADOS_API_HOST_INSECURE"),
		)
		runner.HostConfig.NetworkMode = dockercontainer.NetworkMode(runner.networkMode)
	} else {
		if runner.enableNetwork == "always" {
			runner.HostConfig.NetworkMode = dockercontainer.NetworkMode(runner.networkMode)
		} else {
			runner.HostConfig.NetworkMode = dockercontainer.NetworkMode("none")
		}
	}

	_, stdinUsed := runner.Container.Mounts["stdin"]
	runner.ContainerConfig.OpenStdin = stdinUsed
	runner.ContainerConfig.StdinOnce = stdinUsed
	runner.ContainerConfig.AttachStdin = stdinUsed
	runner.ContainerConfig.AttachStdout = true
	runner.ContainerConfig.AttachStderr = true

	createdBody, err := runner.Docker.ContainerCreate(context.TODO(), &runner.ContainerConfig, &runner.HostConfig, nil, runner.Container.UUID)
	if err != nil {
		return fmt.Errorf("While creating container: %v", err)
	}

	runner.ContainerID = createdBody.ID

	return runner.AttachStreams()
}

// StartContainer starts the docker container created by CreateContainer.
func (runner *ContainerRunner) StartContainer() error {
	runner.CrunchLog.Printf("Starting Docker container id '%s'", runner.ContainerID)
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	if runner.cCancelled {
		return ErrCancelled
	}
	err := runner.Docker.ContainerStart(context.TODO(), runner.ContainerID,
		dockertypes.ContainerStartOptions{})
	if err != nil {
		var advice string
		if m, e := regexp.MatchString("(?ms).*(exec|System error).*(no such file or directory|file not found).*", err.Error()); m && e == nil {
			advice = fmt.Sprintf("\nPossible causes: command %q is missing, the interpreter given in #! is missing, or script has Windows line endings.", runner.Container.Command[0])
		}
		return fmt.Errorf("could not start container: %v%s", err, advice)
	}
	return nil
}

// WaitFinish waits for the container to terminate, capture the exit code, and
// close the stdout/stderr logging.
func (runner *ContainerRunner) WaitFinish() error {
	var runTimeExceeded <-chan time.Time
	runner.CrunchLog.Print("Waiting for container to finish")

	waitOk, waitErr := runner.Docker.ContainerWait(context.TODO(), runner.ContainerID, dockercontainer.WaitConditionNotRunning)
	arvMountExit := runner.ArvMountExit
	if timeout := runner.Container.SchedulingParameters.MaxRunTime; timeout > 0 {
		runTimeExceeded = time.After(time.Duration(timeout) * time.Second)
	}

	containerGone := make(chan struct{})
	go func() {
		defer close(containerGone)
		if runner.containerWatchdogInterval < 1 {
			runner.containerWatchdogInterval = time.Minute
		}
		for range time.NewTicker(runner.containerWatchdogInterval).C {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(runner.containerWatchdogInterval))
			ctr, err := runner.Docker.ContainerInspect(ctx, runner.ContainerID)
			cancel()
			runner.cStateLock.Lock()
			done := runner.cRemoved || runner.ExitCode != nil
			runner.cStateLock.Unlock()
			if done {
				return
			} else if err != nil {
				runner.CrunchLog.Printf("Error inspecting container: %s", err)
				runner.checkBrokenNode(err)
				return
			} else if ctr.State == nil || !(ctr.State.Running || ctr.State.Status == "created") {
				runner.CrunchLog.Printf("Container is not running: State=%v", ctr.State)
				return
			}
		}
	}()

	for {
		select {
		case waitBody := <-waitOk:
			runner.CrunchLog.Printf("Container exited with code: %v", waitBody.StatusCode)
			code := int(waitBody.StatusCode)
			runner.ExitCode = &code

			// wait for stdout/stderr to complete
			<-runner.loggingDone
			return nil

		case err := <-waitErr:
			return fmt.Errorf("container wait: %v", err)

		case <-arvMountExit:
			runner.CrunchLog.Printf("arv-mount exited while container is still running.  Stopping container.")
			runner.stop(nil)
			// arvMountExit will always be ready now that
			// it's closed, but that doesn't interest us.
			arvMountExit = nil

		case <-runTimeExceeded:
			runner.CrunchLog.Printf("maximum run time exceeded. Stopping container.")
			runner.stop(nil)
			runTimeExceeded = nil

		case <-containerGone:
			return errors.New("docker client never returned status")
		}
	}
}

func (runner *ContainerRunner) updateLogs() {
	ticker := time.NewTicker(crunchLogUpdatePeriod / 360)
	defer ticker.Stop()

	sigusr1 := make(chan os.Signal, 1)
	signal.Notify(sigusr1, syscall.SIGUSR1)
	defer signal.Stop(sigusr1)

	saveAtTime := time.Now().Add(crunchLogUpdatePeriod)
	saveAtSize := crunchLogUpdateSize
	var savedSize int64
	for {
		select {
		case <-ticker.C:
		case <-sigusr1:
			saveAtTime = time.Now()
		}
		runner.logMtx.Lock()
		done := runner.LogsPDH != nil
		runner.logMtx.Unlock()
		if done {
			return
		}
		size := runner.LogCollection.Size()
		if size == savedSize || (time.Now().Before(saveAtTime) && size < saveAtSize) {
			continue
		}
		saveAtTime = time.Now().Add(crunchLogUpdatePeriod)
		saveAtSize = runner.LogCollection.Size() + crunchLogUpdateSize
		saved, err := runner.saveLogCollection(false)
		if err != nil {
			runner.CrunchLog.Printf("error updating log collection: %s", err)
			continue
		}

		var updated arvados.Container
		err = runner.DispatcherArvClient.Update("containers", runner.Container.UUID, arvadosclient.Dict{
			"container": arvadosclient.Dict{"log": saved.PortableDataHash},
		}, &updated)
		if err != nil {
			runner.CrunchLog.Printf("error updating container log to %s: %s", saved.PortableDataHash, err)
			continue
		}

		savedSize = size
	}
}

// CaptureOutput saves data from the container's output directory if
// needed, and updates the container output accordingly.
func (runner *ContainerRunner) CaptureOutput() error {
	if runner.Container.RuntimeConstraints.API {
		// Output may have been set directly by the container, so
		// refresh the container record to check.
		err := runner.DispatcherArvClient.Get("containers", runner.Container.UUID,
			nil, &runner.Container)
		if err != nil {
			return err
		}
		if runner.Container.Output != "" {
			// Container output is already set.
			runner.OutputPDH = &runner.Container.Output
			return nil
		}
	}

	txt, err := (&copier{
		client:        runner.containerClient,
		arvClient:     runner.ContainerArvClient,
		keepClient:    runner.ContainerKeepClient,
		hostOutputDir: runner.HostOutputDir,
		ctrOutputDir:  runner.Container.OutputPath,
		binds:         runner.Binds,
		mounts:        runner.Container.Mounts,
		secretMounts:  runner.SecretMounts,
		logger:        runner.CrunchLog,
	}).Copy()
	if err != nil {
		return err
	}
	if n := len(regexp.MustCompile(` [0-9a-f]+\+\S*\+R`).FindAllStringIndex(txt, -1)); n > 0 {
		runner.CrunchLog.Printf("Copying %d data blocks from remote input collections...", n)
		fs, err := (&arvados.Collection{ManifestText: txt}).FileSystem(runner.containerClient, runner.ContainerKeepClient)
		if err != nil {
			return err
		}
		txt, err = fs.MarshalManifest(".")
		if err != nil {
			return err
		}
	}
	var resp arvados.Collection
	err = runner.ContainerArvClient.Create("collections", arvadosclient.Dict{
		"ensure_unique_name": true,
		"collection": arvadosclient.Dict{
			"is_trashed":    true,
			"name":          "output for " + runner.Container.UUID,
			"manifest_text": txt,
		},
	}, &resp)
	if err != nil {
		return fmt.Errorf("error creating output collection: %v", err)
	}
	runner.OutputPDH = &resp.PortableDataHash
	return nil
}

func (runner *ContainerRunner) CleanupDirs() {
	if runner.ArvMount != nil {
		var delay int64 = 8
		umount := exec.Command("arv-mount", fmt.Sprintf("--unmount-timeout=%d", delay), "--unmount", runner.ArvMountPoint)
		umount.Stdout = runner.CrunchLog
		umount.Stderr = runner.CrunchLog
		runner.CrunchLog.Printf("Running %v", umount.Args)
		umnterr := umount.Start()

		if umnterr != nil {
			runner.CrunchLog.Printf("Error unmounting: %v", umnterr)
		} else {
			// If arv-mount --unmount gets stuck for any reason, we
			// don't want to wait for it forever.  Do Wait() in a goroutine
			// so it doesn't block crunch-run.
			umountExit := make(chan error)
			go func() {
				mnterr := umount.Wait()
				if mnterr != nil {
					runner.CrunchLog.Printf("Error unmounting: %v", mnterr)
				}
				umountExit <- mnterr
			}()

			for again := true; again; {
				again = false
				select {
				case <-umountExit:
					umount = nil
					again = true
				case <-runner.ArvMountExit:
					break
				case <-time.After(time.Duration((delay + 1) * int64(time.Second))):
					runner.CrunchLog.Printf("Timed out waiting for unmount")
					if umount != nil {
						umount.Process.Kill()
					}
					runner.ArvMount.Process.Kill()
				}
			}
		}
	}

	if runner.ArvMountPoint != "" {
		if rmerr := os.Remove(runner.ArvMountPoint); rmerr != nil {
			runner.CrunchLog.Printf("While cleaning up arv-mount directory %s: %v", runner.ArvMountPoint, rmerr)
		}
	}

	if rmerr := os.RemoveAll(runner.parentTemp); rmerr != nil {
		runner.CrunchLog.Printf("While cleaning up temporary directory %s: %v", runner.parentTemp, rmerr)
	}
}

// CommitLogs posts the collection containing the final container logs.
func (runner *ContainerRunner) CommitLogs() error {
	func() {
		// Hold cStateLock to prevent races on CrunchLog (e.g., stop()).
		runner.cStateLock.Lock()
		defer runner.cStateLock.Unlock()

		runner.CrunchLog.Print(runner.finalState)

		if runner.arvMountLog != nil {
			runner.arvMountLog.Close()
		}
		runner.CrunchLog.Close()

		// Closing CrunchLog above allows them to be committed to Keep at this
		// point, but re-open crunch log with ArvClient in case there are any
		// other further errors (such as failing to write the log to Keep!)
		// while shutting down
		runner.CrunchLog = NewThrottledLogger(&ArvLogWriter{
			ArvClient:     runner.DispatcherArvClient,
			UUID:          runner.Container.UUID,
			loggingStream: "crunch-run",
			writeCloser:   nil,
		})
		runner.CrunchLog.Immediate = log.New(os.Stderr, runner.Container.UUID+" ", 0)
	}()

	if runner.LogsPDH != nil {
		// If we have already assigned something to LogsPDH,
		// we must be closing the re-opened log, which won't
		// end up getting attached to the container record and
		// therefore doesn't need to be saved as a collection
		// -- it exists only to send logs to other channels.
		return nil
	}
	saved, err := runner.saveLogCollection(true)
	if err != nil {
		return fmt.Errorf("error saving log collection: %s", err)
	}
	runner.logMtx.Lock()
	defer runner.logMtx.Unlock()
	runner.LogsPDH = &saved.PortableDataHash
	return nil
}

func (runner *ContainerRunner) saveLogCollection(final bool) (response arvados.Collection, err error) {
	runner.logMtx.Lock()
	defer runner.logMtx.Unlock()
	if runner.LogsPDH != nil {
		// Already finalized.
		return
	}
	updates := arvadosclient.Dict{
		"name": "logs for " + runner.Container.UUID,
	}
	mt, err1 := runner.LogCollection.MarshalManifest(".")
	if err1 == nil {
		// Only send updated manifest text if there was no
		// error.
		updates["manifest_text"] = mt
	}

	// Even if flushing the manifest had an error, we still want
	// to update the log record, if possible, to push the trash_at
	// and delete_at times into the future.  Details on bug
	// #17293.
	if final {
		updates["is_trashed"] = true
	} else {
		exp := time.Now().Add(crunchLogUpdatePeriod * 24)
		updates["trash_at"] = exp
		updates["delete_at"] = exp
	}
	reqBody := arvadosclient.Dict{"collection": updates}
	var err2 error
	if runner.logUUID == "" {
		reqBody["ensure_unique_name"] = true
		err2 = runner.DispatcherArvClient.Create("collections", reqBody, &response)
	} else {
		err2 = runner.DispatcherArvClient.Update("collections", runner.logUUID, reqBody, &response)
	}
	if err2 == nil {
		runner.logUUID = response.UUID
	}

	if err1 != nil || err2 != nil {
		err = fmt.Errorf("error recording logs: %q, %q", err1, err2)
	}
	return
}

// UpdateContainerRunning updates the container state to "Running"
func (runner *ContainerRunner) UpdateContainerRunning() error {
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	if runner.cCancelled {
		return ErrCancelled
	}
	return runner.DispatcherArvClient.Update("containers", runner.Container.UUID,
		arvadosclient.Dict{"container": arvadosclient.Dict{"state": "Running", "gateway_address": runner.gateway.Address}}, nil)
}

// ContainerToken returns the api_token the container (and any
// arv-mount processes) are allowed to use.
func (runner *ContainerRunner) ContainerToken() (string, error) {
	if runner.token != "" {
		return runner.token, nil
	}

	var auth arvados.APIClientAuthorization
	err := runner.DispatcherArvClient.Call("GET", "containers", runner.Container.UUID, "auth", nil, &auth)
	if err != nil {
		return "", err
	}
	runner.token = fmt.Sprintf("v2/%s/%s/%s", auth.UUID, auth.APIToken, runner.Container.UUID)
	return runner.token, nil
}

// UpdateContainerFinal updates the container record state on API
// server to "Complete" or "Cancelled"
func (runner *ContainerRunner) UpdateContainerFinal() error {
	update := arvadosclient.Dict{}
	update["state"] = runner.finalState
	if runner.LogsPDH != nil {
		update["log"] = *runner.LogsPDH
	}
	if runner.finalState == "Complete" {
		if runner.ExitCode != nil {
			update["exit_code"] = *runner.ExitCode
		}
		if runner.OutputPDH != nil {
			update["output"] = *runner.OutputPDH
		}
	}
	return runner.DispatcherArvClient.Update("containers", runner.Container.UUID, arvadosclient.Dict{"container": update}, nil)
}

// IsCancelled returns the value of Cancelled, with goroutine safety.
func (runner *ContainerRunner) IsCancelled() bool {
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	return runner.cCancelled
}

// NewArvLogWriter creates an ArvLogWriter
func (runner *ContainerRunner) NewArvLogWriter(name string) (io.WriteCloser, error) {
	writer, err := runner.LogCollection.OpenFile(name+".txt", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	return &ArvLogWriter{
		ArvClient:     runner.DispatcherArvClient,
		UUID:          runner.Container.UUID,
		loggingStream: name,
		writeCloser:   writer,
	}, nil
}

// Run the full container lifecycle.
func (runner *ContainerRunner) Run() (err error) {
	runner.CrunchLog.Printf("crunch-run %s started", cmd.Version.String())
	runner.CrunchLog.Printf("Executing container '%s'", runner.Container.UUID)

	hostname, hosterr := os.Hostname()
	if hosterr != nil {
		runner.CrunchLog.Printf("Error getting hostname '%v'", hosterr)
	} else {
		runner.CrunchLog.Printf("Executing on host '%s'", hostname)
	}

	runner.finalState = "Queued"

	defer func() {
		runner.CleanupDirs()

		runner.CrunchLog.Printf("crunch-run finished")
		runner.CrunchLog.Close()
	}()

	err = runner.fetchContainerRecord()
	if err != nil {
		return
	}
	if runner.Container.State != "Locked" {
		return fmt.Errorf("dispatch error detected: container %q has state %q", runner.Container.UUID, runner.Container.State)
	}

	defer func() {
		// checkErr prints e (unless it's nil) and sets err to
		// e (unless err is already non-nil). Thus, if err
		// hasn't already been assigned when Run() returns,
		// this cleanup func will cause Run() to return the
		// first non-nil error that is passed to checkErr().
		checkErr := func(errorIn string, e error) {
			if e == nil {
				return
			}
			runner.CrunchLog.Printf("error in %s: %v", errorIn, e)
			if err == nil {
				err = e
			}
			if runner.finalState == "Complete" {
				// There was an error in the finalization.
				runner.finalState = "Cancelled"
			}
		}

		// Log the error encountered in Run(), if any
		checkErr("Run", err)

		if runner.finalState == "Queued" {
			runner.UpdateContainerFinal()
			return
		}

		if runner.IsCancelled() {
			runner.finalState = "Cancelled"
			// but don't return yet -- we still want to
			// capture partial output and write logs
		}

		checkErr("CaptureOutput", runner.CaptureOutput())
		checkErr("stopHoststat", runner.stopHoststat())
		checkErr("CommitLogs", runner.CommitLogs())
		checkErr("UpdateContainerFinal", runner.UpdateContainerFinal())
	}()

	runner.setupSignals()
	err = runner.startHoststat()
	if err != nil {
		return
	}

	// check for and/or load image
	err = runner.LoadImage()
	if err != nil {
		if !runner.checkBrokenNode(err) {
			// Failed to load image but not due to a "broken node"
			// condition, probably user error.
			runner.finalState = "Cancelled"
		}
		err = fmt.Errorf("While loading container image: %v", err)
		return
	}

	// set up FUSE mount and binds
	err = runner.SetupMounts()
	if err != nil {
		runner.finalState = "Cancelled"
		err = fmt.Errorf("While setting up mounts: %v", err)
		return
	}

	err = runner.CreateContainer()
	if err != nil {
		return
	}
	err = runner.LogHostInfo()
	if err != nil {
		return
	}
	err = runner.LogNodeRecord()
	if err != nil {
		return
	}
	err = runner.LogContainerRecord()
	if err != nil {
		return
	}

	if runner.IsCancelled() {
		return
	}

	err = runner.UpdateContainerRunning()
	if err != nil {
		return
	}
	runner.finalState = "Cancelled"

	err = runner.startCrunchstat()
	if err != nil {
		return
	}

	err = runner.StartContainer()
	if err != nil {
		runner.checkBrokenNode(err)
		return
	}

	err = runner.WaitFinish()
	if err == nil && !runner.IsCancelled() {
		runner.finalState = "Complete"
	}
	return
}

// Fetch the current container record (uuid = runner.Container.UUID)
// into runner.Container.
func (runner *ContainerRunner) fetchContainerRecord() error {
	reader, err := runner.DispatcherArvClient.CallRaw("GET", "containers", runner.Container.UUID, "", nil)
	if err != nil {
		return fmt.Errorf("error fetching container record: %v", err)
	}
	defer reader.Close()

	dec := json.NewDecoder(reader)
	dec.UseNumber()
	err = dec.Decode(&runner.Container)
	if err != nil {
		return fmt.Errorf("error decoding container record: %v", err)
	}

	var sm struct {
		SecretMounts map[string]arvados.Mount `json:"secret_mounts"`
	}

	containerToken, err := runner.ContainerToken()
	if err != nil {
		return fmt.Errorf("error getting container token: %v", err)
	}

	runner.ContainerArvClient, runner.ContainerKeepClient,
		runner.containerClient, err = runner.MkArvClient(containerToken)
	if err != nil {
		return fmt.Errorf("error creating container API client: %v", err)
	}

	err = runner.ContainerArvClient.Call("GET", "containers", runner.Container.UUID, "secret_mounts", nil, &sm)
	if err != nil {
		if apierr, ok := err.(arvadosclient.APIServerError); !ok || apierr.HttpStatusCode != 404 {
			return fmt.Errorf("error fetching secret_mounts: %v", err)
		}
		// ok && apierr.HttpStatusCode == 404, which means
		// secret_mounts isn't supported by this API server.
	}
	runner.SecretMounts = sm.SecretMounts

	return nil
}

// NewContainerRunner creates a new container runner.
func NewContainerRunner(dispatcherClient *arvados.Client,
	dispatcherArvClient IArvadosClient,
	dispatcherKeepClient IKeepClient,
	docker ThinDockerClient,
	containerUUID string) (*ContainerRunner, error) {

	cr := &ContainerRunner{
		dispatcherClient:     dispatcherClient,
		DispatcherArvClient:  dispatcherArvClient,
		DispatcherKeepClient: dispatcherKeepClient,
		Docker:               docker,
	}
	cr.NewLogWriter = cr.NewArvLogWriter
	cr.RunArvMount = cr.ArvMountCmd
	cr.MkTempDir = ioutil.TempDir
	cr.MkArvClient = func(token string) (IArvadosClient, IKeepClient, *arvados.Client, error) {
		cl, err := arvadosclient.MakeArvadosClient()
		if err != nil {
			return nil, nil, nil, err
		}
		cl.ApiToken = token
		kc, err := keepclient.MakeKeepClient(cl)
		if err != nil {
			return nil, nil, nil, err
		}
		c2 := arvados.NewClientFromEnv()
		c2.AuthToken = token
		return cl, kc, c2, nil
	}
	var err error
	cr.LogCollection, err = (&arvados.Collection{}).FileSystem(cr.dispatcherClient, cr.DispatcherKeepClient)
	if err != nil {
		return nil, err
	}
	cr.Container.UUID = containerUUID
	w, err := cr.NewLogWriter("crunch-run")
	if err != nil {
		return nil, err
	}
	cr.CrunchLog = NewThrottledLogger(w)
	cr.CrunchLog.Immediate = log.New(os.Stderr, containerUUID+" ", 0)

	loadLogThrottleParams(dispatcherArvClient)
	go cr.updateLogs()

	return cr, nil
}

func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	statInterval := flags.Duration("crunchstat-interval", 10*time.Second, "sampling period for periodic resource usage reporting")
	cgroupRoot := flags.String("cgroup-root", "/sys/fs/cgroup", "path to sysfs cgroup tree")
	cgroupParent := flags.String("cgroup-parent", "docker", "name of container's parent cgroup (ignored if -cgroup-parent-subsystem is used)")
	cgroupParentSubsystem := flags.String("cgroup-parent-subsystem", "", "use current cgroup for given subsystem as parent cgroup for container")
	caCertsPath := flags.String("ca-certs", "", "Path to TLS root certificates")
	detach := flags.Bool("detach", false, "Detach from parent process and run in the background")
	stdinEnv := flags.Bool("stdin-env", false, "Load environment variables from JSON message on stdin")
	sleep := flags.Duration("sleep", 0, "Delay before starting (testing use only)")
	kill := flags.Int("kill", -1, "Send signal to an existing crunch-run process for given UUID")
	list := flags.Bool("list", false, "List UUIDs of existing crunch-run processes")
	enableNetwork := flags.String("container-enable-networking", "default",
		`Specify if networking should be enabled for container.  One of 'default', 'always':
    	default: only enable networking if container requests it.
    	always:  containers always have networking enabled
    	`)
	networkMode := flags.String("container-network-mode", "default",
		`Set networking mode for container.  Corresponds to Docker network mode (--net).
    	`)
	memprofile := flags.String("memprofile", "", "write memory profile to `file` after running container")
	flags.Duration("check-containerd", 0, "Ignored. Exists for compatibility with older versions.")

	ignoreDetachFlag := false
	if len(args) > 0 && args[0] == "-no-detach" {
		// This process was invoked by a parent process, which
		// has passed along its own arguments, including
		// -detach, after the leading -no-detach flag.  Strip
		// the leading -no-detach flag (it's not recognized by
		// flags.Parse()) and ignore the -detach flag that
		// comes later.
		args = args[1:]
		ignoreDetachFlag = true
	}

	if err := flags.Parse(args); err == flag.ErrHelp {
		return 0
	} else if err != nil {
		log.Print(err)
		return 1
	}

	if *stdinEnv && !ignoreDetachFlag {
		// Load env vars on stdin if asked (but not in a
		// detached child process, in which case stdin is
		// /dev/null).
		err := loadEnv(os.Stdin)
		if err != nil {
			log.Print(err)
			return 1
		}
	}

	containerID := flags.Arg(0)

	switch {
	case *detach && !ignoreDetachFlag:
		return Detach(containerID, prog, args, os.Stdout, os.Stderr)
	case *kill >= 0:
		return KillProcess(containerID, syscall.Signal(*kill), os.Stdout, os.Stderr)
	case *list:
		return ListProcesses(os.Stdout, os.Stderr)
	}

	if containerID == "" {
		log.Printf("usage: %s [options] UUID", prog)
		return 1
	}

	log.Printf("crunch-run %s started", cmd.Version.String())
	time.Sleep(*sleep)

	if *caCertsPath != "" {
		arvadosclient.CertFiles = []string{*caCertsPath}
	}

	api, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Printf("%s: %v", containerID, err)
		return 1
	}
	api.Retries = 8

	kc, kcerr := keepclient.MakeKeepClient(api)
	if kcerr != nil {
		log.Printf("%s: %v", containerID, kcerr)
		return 1
	}
	kc.BlockCache = &keepclient.BlockCache{MaxBlocks: 2}
	kc.Retries = 4

	// API version 1.21 corresponds to Docker 1.9, which is currently the
	// minimum version we want to support.
	docker, dockererr := dockerclient.NewClient(dockerclient.DefaultDockerHost, "1.21", nil, nil)

	cr, err := NewContainerRunner(arvados.NewClientFromEnv(), api, kc, docker, containerID)
	if err != nil {
		log.Print(err)
		return 1
	}
	if dockererr != nil {
		cr.CrunchLog.Printf("%s: %v", containerID, dockererr)
		cr.checkBrokenNode(dockererr)
		cr.CrunchLog.Close()
		return 1
	}

	cr.gateway = Gateway{
		Address:           os.Getenv("GatewayAddress"),
		AuthSecret:        os.Getenv("GatewayAuthSecret"),
		ContainerUUID:     containerID,
		DockerContainerID: &cr.ContainerID,
		Log:               cr.CrunchLog,
	}
	os.Unsetenv("GatewayAuthSecret")
	if cr.gateway.Address != "" {
		err = cr.gateway.Start()
		if err != nil {
			log.Printf("error starting gateway server: %s", err)
			return 1
		}
	}

	parentTemp, tmperr := cr.MkTempDir("", "crunch-run."+containerID+".")
	if tmperr != nil {
		log.Printf("%s: %v", containerID, tmperr)
		return 1
	}

	cr.parentTemp = parentTemp
	cr.statInterval = *statInterval
	cr.cgroupRoot = *cgroupRoot
	cr.expectCgroupParent = *cgroupParent
	cr.enableNetwork = *enableNetwork
	cr.networkMode = *networkMode
	if *cgroupParentSubsystem != "" {
		p := findCgroup(*cgroupParentSubsystem)
		cr.setCgroupParent = p
		cr.expectCgroupParent = p
	}

	runerr := cr.Run()

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Printf("could not create memory profile: %s", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Printf("could not write memory profile: %s", err)
		}
		closeerr := f.Close()
		if closeerr != nil {
			log.Printf("closing memprofile file: %s", err)
		}
	}

	if runerr != nil {
		log.Printf("%s: %v", containerID, runerr)
		return 1
	}
	return 0
}

func loadEnv(rdr io.Reader) error {
	buf, err := ioutil.ReadAll(rdr)
	if err != nil {
		return fmt.Errorf("read stdin: %s", err)
	}
	var env map[string]string
	err = json.Unmarshal(buf, &env)
	if err != nil {
		return fmt.Errorf("decode stdin: %s", err)
	}
	for k, v := range env {
		err = os.Setenv(k, v)
		if err != nil {
			return fmt.Errorf("setenv(%q): %s", k, err)
		}
	}
	return nil
}
