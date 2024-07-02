// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
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

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/crunchstat"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"golang.org/x/sys/unix"
)

type command struct{}

var arvadosCertPath = "/etc/arvados/ca-certificates.crt"

var Command = command{}

// ConfigData contains environment variables and (when needed) cluster
// configuration, passed from dispatchcloud to crunch-run on stdin.
type ConfigData struct {
	Env          map[string]string
	KeepBuffers  int
	EC2SpotCheck bool
	Cluster      *arvados.Cluster
}

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
	BlockWrite(context.Context, arvados.BlockWriteOptions) (arvados.BlockWriteResponse, error)
	ReadAt(locator string, p []byte, off int) (int, error)
	LocalLocator(locator string) (string, error)
	SetStorageClasses(sc []string)
}

type RunArvMount func(cmdline []string, tok string) (*exec.Cmd, error)

type MkTempDir func(string, string) (string, error)

type PsProcess interface {
	CmdlineSlice() ([]string, error)
}

// ContainerRunner is the main stateful struct used for a single execution of a
// container.
type ContainerRunner struct {
	executor       containerExecutor
	executorStdin  io.Closer
	executorStdout io.Closer
	executorStderr io.Closer

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

	Container     arvados.Container
	token         string
	ExitCode      *int
	CrunchLog     *logWriter
	logUUID       string
	logMtx        sync.Mutex
	LogCollection arvados.CollectionFileSystem
	LogsPDH       *string
	RunArvMount   RunArvMount
	MkTempDir     MkTempDir
	ArvMount      *exec.Cmd
	ArvMountPoint string
	HostOutputDir string
	Volumes       map[string]struct{}
	OutputPDH     *string
	SigChan       chan os.Signal
	ArvMountExit  chan error
	SecretMounts  map[string]arvados.Mount
	MkArvClient   func(token string) (IArvadosClient, IKeepClient, *arvados.Client, error)
	finalState    string
	parentTemp    string
	costStartTime time.Time

	keepstore        *exec.Cmd
	keepstoreLogger  io.WriteCloser
	keepstoreLogbuf  *bufThenWrite
	statLogger       io.WriteCloser
	statReporter     *crunchstat.Reporter
	hoststatLogger   io.WriteCloser
	hoststatReporter *crunchstat.Reporter
	statInterval     time.Duration
	// What we tell docker to use as the container's cgroup
	// parent.
	setCgroupParent string
	// Fake root dir where crunchstat.Reporter should read OS
	// files, for testing.
	crunchstatFakeFS fs.FS

	cStateLock sync.Mutex
	cCancelled bool // StopContainer() invoked

	enableMemoryLimit bool
	enableNetwork     string // one of "default" or "always"
	networkMode       string // "none", "host", or "" -- passed through to executor
	brokenNodeHook    string // script to run if node appears to be broken
	arvMountLog       io.WriteCloser

	containerWatchdogInterval time.Duration

	gateway Gateway

	prices     []cloud.InstancePrice
	pricesLock sync.Mutex
}

// setupSignals sets up signal handling to gracefully terminate the
// underlying container and update state when receiving a TERM, INT or
// QUIT signal.
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

// stop the underlying container.
func (runner *ContainerRunner) stop(sig os.Signal) {
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	if sig != nil {
		runner.CrunchLog.Printf("caught signal: %v", sig)
	}
	runner.cCancelled = true
	runner.CrunchLog.Printf("stopping container")
	err := runner.executor.Stop()
	if err != nil {
		runner.CrunchLog.Printf("error stopping container: %s", err)
	}
}

var errorBlacklist = []string{
	"(?ms).*[Cc]annot connect to the Docker daemon.*",
	"(?ms).*oci runtime error.*starting container process.*container init.*mounting.*to rootfs.*no such file or directory.*",
	"(?ms).*grpc: the connection is unavailable.*",
}

func (runner *ContainerRunner) runBrokenNodeHook() {
	if runner.brokenNodeHook == "" {
		path := filepath.Join(lockdir, brokenfile)
		runner.CrunchLog.Printf("Writing %s to mark node as broken", path)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0700)
		if err != nil {
			runner.CrunchLog.Printf("Error writing %s: %s", path, err)
			return
		}
		f.Close()
	} else {
		runner.CrunchLog.Printf("Running broken node hook %q", runner.brokenNodeHook)
		// run killme script
		c := exec.Command(runner.brokenNodeHook)
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
func (runner *ContainerRunner) LoadImage() (string, error) {
	runner.CrunchLog.Printf("Fetching Docker image from collection '%s'", runner.Container.ContainerImage)

	d, err := os.Open(runner.ArvMountPoint + "/by_id/" + runner.Container.ContainerImage)
	if err != nil {
		return "", err
	}
	defer d.Close()
	allfiles, err := d.Readdirnames(-1)
	if err != nil {
		return "", err
	}
	var tarfiles []string
	for _, fnm := range allfiles {
		if strings.HasSuffix(fnm, ".tar") {
			tarfiles = append(tarfiles, fnm)
		}
	}
	if len(tarfiles) == 0 {
		return "", fmt.Errorf("image collection does not include a .tar image file")
	}
	if len(tarfiles) > 1 {
		return "", fmt.Errorf("cannot choose from multiple tar files in image collection: %v", tarfiles)
	}
	imageID := tarfiles[0][:len(tarfiles[0])-4]
	imageTarballPath := runner.ArvMountPoint + "/by_id/" + runner.Container.ContainerImage + "/" + imageID + ".tar"
	runner.CrunchLog.Printf("Using Docker image id %q", imageID)

	runner.CrunchLog.Print("Loading Docker image from keep")
	err = runner.executor.LoadImage(imageID, imageTarballPath, runner.Container, runner.ArvMountPoint,
		runner.containerClient)
	if err != nil {
		return "", err
	}

	return imageID, nil
}

func (runner *ContainerRunner) ArvMountCmd(cmdline []string, token string) (c *exec.Cmd, err error) {
	c = exec.Command(cmdline[0], cmdline[1:]...)

	// Copy our environment, but override ARVADOS_API_TOKEN with
	// the container auth token.
	c.Env = nil
	for _, s := range os.Environ() {
		if !strings.HasPrefix(s, "ARVADOS_API_TOKEN=") {
			c.Env = append(c.Env, s)
		}
	}
	c.Env = append(c.Env, "ARVADOS_API_TOKEN="+token)

	runner.arvMountLog, err = runner.openLogFile("arv-mount")
	if err != nil {
		return nil, err
	}
	scanner := logScanner{
		Patterns: []string{
			"Keep write error",
			"Block not found error",
			"Unhandled exception during FUSE operation",
		},
		ReportFunc: func(pattern, text string) {
			runner.updateRuntimeStatus(arvadosclient.Dict{
				"warning":       "arv-mount: " + pattern,
				"warningDetail": text,
			})
		},
	}
	c.Stdout = runner.arvMountLog
	c.Stderr = io.MultiWriter(runner.arvMountLog, os.Stderr, &scanner)

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

func (runner *ContainerRunner) SetupMounts() (map[string]bindmount, error) {
	bindmounts := map[string]bindmount{}
	err := runner.SetupArvMountPoint("keep")
	if err != nil {
		return nil, fmt.Errorf("While creating keep mount temp dir: %v", err)
	}

	token, err := runner.ContainerToken()
	if err != nil {
		return nil, fmt.Errorf("could not get container token: %s", err)
	}
	runner.CrunchLog.Printf("container token %q", token)

	pdhOnly := true
	tmpcount := 0
	arvMountCmd := []string{
		"arv-mount",
		"--foreground",
		"--read-write",
		"--storage-classes", strings.Join(runner.Container.OutputStorageClasses, ","),
		fmt.Sprintf("--crunchstat-interval=%v", runner.statInterval.Seconds())}

	if _, isdocker := runner.executor.(*dockerExecutor); isdocker {
		arvMountCmd = append(arvMountCmd, "--allow-other")
	}

	if runner.Container.RuntimeConstraints.KeepCacheDisk > 0 {
		keepcachedir, err := runner.MkTempDir(runner.parentTemp, "keepcache")
		if err != nil {
			return nil, fmt.Errorf("while creating keep cache temp dir: %v", err)
		}
		arvMountCmd = append(arvMountCmd, "--disk-cache", "--disk-cache-dir", keepcachedir, "--file-cache", fmt.Sprintf("%d", runner.Container.RuntimeConstraints.KeepCacheDisk))
	} else if runner.Container.RuntimeConstraints.KeepCacheRAM > 0 {
		arvMountCmd = append(arvMountCmd, "--ram-cache", "--file-cache", fmt.Sprintf("%d", runner.Container.RuntimeConstraints.KeepCacheRAM))
	}

	collectionPaths := []string{}
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
			return nil, fmt.Errorf("secret mount %q conflicts with regular mount", bind)
		}
		if runner.SecretMounts[bind].Kind != "json" &&
			runner.SecretMounts[bind].Kind != "text" {
			return nil, fmt.Errorf("secret mount %q type is %q but only 'json' and 'text' are permitted",
				bind, runner.SecretMounts[bind].Kind)
		}
		binds = append(binds, bind)
	}
	sort.Strings(binds)

	for _, bind := range binds {
		mnt, notSecret := runner.Container.Mounts[bind]
		if !notSecret {
			mnt = runner.SecretMounts[bind]
		}
		if bind == "stdout" || bind == "stderr" {
			// Is it a "file" mount kind?
			if mnt.Kind != "file" {
				return nil, fmt.Errorf("unsupported mount kind '%s' for %s: only 'file' is supported", mnt.Kind, bind)
			}

			// Does path start with OutputPath?
			prefix := runner.Container.OutputPath
			if !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
			if !strings.HasPrefix(mnt.Path, prefix) {
				return nil, fmt.Errorf("%s path does not start with OutputPath: %s, %s", strings.Title(bind), mnt.Path, prefix)
			}
		}

		if bind == "stdin" {
			// Is it a "collection" mount kind?
			if mnt.Kind != "collection" && mnt.Kind != "json" {
				return nil, fmt.Errorf("unsupported mount kind '%s' for stdin: only 'collection' and 'json' are supported", mnt.Kind)
			}
		}

		if bind == arvadosCertPath {
			needCertMount = false
		}

		if strings.HasPrefix(bind, runner.Container.OutputPath+"/") && bind != runner.Container.OutputPath+"/" {
			if mnt.Kind != "collection" && mnt.Kind != "text" && mnt.Kind != "json" {
				return nil, fmt.Errorf("only mount points of kind 'collection', 'text' or 'json' are supported underneath the output_path for %q, was %q", bind, mnt.Kind)
			}
		}

		switch {
		case mnt.Kind == "collection" && bind != "stdin":
			var src string
			if mnt.UUID != "" && mnt.PortableDataHash != "" {
				return nil, fmt.Errorf("cannot specify both 'uuid' and 'portable_data_hash' for a collection mount")
			}
			if mnt.UUID != "" {
				if mnt.Writable {
					return nil, fmt.Errorf("writing to existing collections currently not permitted")
				}
				pdhOnly = false
				src = fmt.Sprintf("%s/by_id/%s", runner.ArvMountPoint, mnt.UUID)
			} else if mnt.PortableDataHash != "" {
				if mnt.Writable && !strings.HasPrefix(bind, runner.Container.OutputPath+"/") {
					return nil, fmt.Errorf("can never write to a collection specified by portable data hash")
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
				arvMountCmd = append(arvMountCmd, "--mount-tmp", fmt.Sprintf("tmp%d", tmpcount))
				tmpcount++
			}
			if mnt.Writable {
				if bind == runner.Container.OutputPath {
					runner.HostOutputDir = src
					bindmounts[bind] = bindmount{HostPath: src}
				} else if strings.HasPrefix(bind, runner.Container.OutputPath+"/") {
					copyFiles = append(copyFiles, copyFile{src, runner.HostOutputDir + bind[len(runner.Container.OutputPath):]})
				} else {
					bindmounts[bind] = bindmount{HostPath: src}
				}
			} else {
				bindmounts[bind] = bindmount{HostPath: src, ReadOnly: true}
			}
			collectionPaths = append(collectionPaths, src)

		case mnt.Kind == "tmp":
			var tmpdir string
			tmpdir, err = runner.MkTempDir(runner.parentTemp, "tmp")
			if err != nil {
				return nil, fmt.Errorf("while creating mount temp dir: %v", err)
			}
			st, staterr := os.Stat(tmpdir)
			if staterr != nil {
				return nil, fmt.Errorf("while Stat on temp dir: %v", staterr)
			}
			err = os.Chmod(tmpdir, st.Mode()|os.ModeSetgid|0777)
			if staterr != nil {
				return nil, fmt.Errorf("while Chmod temp dir: %v", err)
			}
			bindmounts[bind] = bindmount{HostPath: tmpdir}
			if bind == runner.Container.OutputPath {
				runner.HostOutputDir = tmpdir
			}

		case mnt.Kind == "json" || mnt.Kind == "text":
			var filedata []byte
			if mnt.Kind == "json" {
				filedata, err = json.Marshal(mnt.Content)
				if err != nil {
					return nil, fmt.Errorf("encoding json data: %v", err)
				}
			} else {
				text, ok := mnt.Content.(string)
				if !ok {
					return nil, fmt.Errorf("content for mount %q must be a string", bind)
				}
				filedata = []byte(text)
			}

			tmpdir, err := runner.MkTempDir(runner.parentTemp, mnt.Kind)
			if err != nil {
				return nil, fmt.Errorf("creating temp dir: %v", err)
			}
			tmpfn := filepath.Join(tmpdir, "mountdata."+mnt.Kind)
			err = ioutil.WriteFile(tmpfn, filedata, 0444)
			if err != nil {
				return nil, fmt.Errorf("writing temp file: %v", err)
			}
			if strings.HasPrefix(bind, runner.Container.OutputPath+"/") && (notSecret || runner.Container.Mounts[runner.Container.OutputPath].Kind != "collection") {
				// In most cases, if the container
				// specifies a literal file inside the
				// output path, we copy it into the
				// output directory (either a mounted
				// collection or a staging area on the
				// host fs). If it's a secret, it will
				// be skipped when copying output from
				// staging to Keep later.
				copyFiles = append(copyFiles, copyFile{tmpfn, runner.HostOutputDir + bind[len(runner.Container.OutputPath):]})
			} else {
				// If a secret is outside OutputPath,
				// we bind mount the secret file
				// directly just like other mounts. We
				// also use this strategy when a
				// secret is inside OutputPath but
				// OutputPath is a live collection, to
				// avoid writing the secret to
				// Keep. Attempting to remove a
				// bind-mounted secret file from
				// inside the container will return a
				// "Device or resource busy" error
				// that might not be handled well by
				// the container, which is why we
				// don't use this strategy when
				// OutputPath is a staging directory.
				bindmounts[bind] = bindmount{HostPath: tmpfn, ReadOnly: true}
			}
		}
	}

	if runner.HostOutputDir == "" {
		return nil, fmt.Errorf("output path does not correspond to a writable mount point")
	}

	if needCertMount && runner.Container.RuntimeConstraints.API {
		for _, certfile := range []string{
			// Populated by caller, or sdk/go/arvados init(), or test suite:
			os.Getenv("SSL_CERT_FILE"),
			// Copied from Go 1.21 stdlib (src/crypto/x509/root_linux.go):
			"/etc/ssl/certs/ca-certificates.crt",                // Debian/Ubuntu/Gentoo etc.
			"/etc/pki/tls/certs/ca-bundle.crt",                  // Fedora/RHEL 6
			"/etc/ssl/ca-bundle.pem",                            // OpenSUSE
			"/etc/pki/tls/cacert.pem",                           // OpenELEC
			"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", // CentOS/RHEL 7
			"/etc/ssl/cert.pem",                                 // Alpine Linux
		} {
			if _, err := os.Stat(certfile); err == nil {
				bindmounts[arvadosCertPath] = bindmount{HostPath: certfile, ReadOnly: true}
				break
			}
		}
	}

	if pdhOnly {
		// If we are only mounting collections by pdh, make
		// sure we don't subscribe to websocket events to
		// avoid putting undesired load on the API server
		arvMountCmd = append(arvMountCmd, "--mount-by-pdh", "by_id", "--disable-event-listening")
	} else {
		arvMountCmd = append(arvMountCmd, "--mount-by-id", "by_id")
	}
	// the by_uuid mount point is used by singularity when writing
	// out docker images converted to SIF
	arvMountCmd = append(arvMountCmd, "--mount-by-id", "by_uuid")
	arvMountCmd = append(arvMountCmd, runner.ArvMountPoint)

	runner.ArvMount, err = runner.RunArvMount(arvMountCmd, token)
	if err != nil {
		return nil, fmt.Errorf("while trying to start arv-mount: %v", err)
	}
	if runner.hoststatReporter != nil && runner.ArvMount != nil {
		runner.hoststatReporter.ReportPID("arv-mount", runner.ArvMount.Process.Pid)
	}

	for _, p := range collectionPaths {
		_, err = os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("while checking that input files exist: %v", err)
		}
	}

	for _, cp := range copyFiles {
		st, err := os.Stat(cp.src)
		if err != nil {
			return nil, fmt.Errorf("while staging writable file from %q to %q: %v", cp.src, cp.bind, err)
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
			return nil, fmt.Errorf("while staging writable file from %q to %q: %v", cp.src, cp.bind, err)
		}
	}

	return bindmounts, nil
}

func (runner *ContainerRunner) stopHoststat() error {
	if runner.hoststatReporter == nil {
		return nil
	}
	runner.hoststatReporter.Stop()
	runner.hoststatReporter.LogProcessMemMax(runner.CrunchLog)
	err := runner.hoststatLogger.Close()
	if err != nil {
		return fmt.Errorf("error closing hoststat logs: %v", err)
	}
	return nil
}

func (runner *ContainerRunner) startHoststat() error {
	var err error
	runner.hoststatLogger, err = runner.openLogFile("hoststat")
	if err != nil {
		return err
	}
	runner.hoststatReporter = &crunchstat.Reporter{
		Logger: newLogWriter(runner.hoststatLogger),
		// Our own cgroup is the "host" cgroup, in the sense
		// that it accounts for resource usage outside the
		// container. It doesn't count _all_ resource usage on
		// the system.
		//
		// TODO?: Use the furthest ancestor of our own cgroup
		// that has stats available. (Currently crunchstat
		// does not have that capability.)
		Pid:        os.Getpid,
		PollPeriod: runner.statInterval,
	}
	runner.hoststatReporter.Start()
	runner.hoststatReporter.ReportPID("crunch-run", os.Getpid())
	return nil
}

func (runner *ContainerRunner) startCrunchstat() error {
	var err error
	runner.statLogger, err = runner.openLogFile("crunchstat")
	if err != nil {
		return err
	}
	runner.statReporter = &crunchstat.Reporter{
		Pid:    runner.executor.Pid,
		FS:     runner.crunchstatFakeFS,
		Logger: newLogWriter(runner.statLogger),
		MemThresholds: map[string][]crunchstat.Threshold{
			"rss": crunchstat.NewThresholdsFromPercentages(runner.Container.RuntimeConstraints.RAM, []int64{90, 95, 99}),
		},
		PollPeriod:      runner.statInterval,
		TempDir:         runner.parentTemp,
		ThresholdLogger: runner.CrunchLog,
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
	w, err := runner.openLogFile("node-info")
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
	logged, err := runner.logAPIResponse("container", "containers", map[string]interface{}{"filters": [][]string{{"uuid", "=", runner.Container.UUID}}})
	if !logged && err == nil {
		err = fmt.Errorf("error: no container record found for %s", runner.Container.UUID)
	}
	return err
}

// LogNodeRecord logs the current host's InstanceType config entry, if
// running via arvados-dispatch-cloud.
func (runner *ContainerRunner) LogNodeRecord() error {
	it := os.Getenv("InstanceType")
	if it == "" {
		// Not dispatched by arvados-dispatch-cloud.
		return nil
	}
	// Save InstanceType config fragment received from dispatcher
	// on stdin.
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

func (runner *ContainerRunner) logAPIResponse(label, path string, params map[string]interface{}) (logged bool, err error) {
	writer, err := runner.LogCollection.OpenFile(label+".json", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return false, err
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
	// Re-encode it using indentation to improve readability
	enc := json.NewEncoder(writer)
	enc.SetIndent("", "    ")
	if err = enc.Encode(items[0]); err != nil {
		return false, fmt.Errorf("error logging %s record: %v", label, err)
	}
	err = writer.Close()
	if err != nil {
		return false, fmt.Errorf("error closing %s.json in log collection: %v", label, err)
	}
	return true, nil
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
func (runner *ContainerRunner) CreateContainer(imageID string, bindmounts map[string]bindmount) error {
	var stdin io.Reader
	if mnt, ok := runner.Container.Mounts["stdin"]; ok {
		switch mnt.Kind {
		case "collection":
			var collID string
			if mnt.UUID != "" {
				collID = mnt.UUID
			} else {
				collID = mnt.PortableDataHash
			}
			path := runner.ArvMountPoint + "/by_id/" + collID + "/" + mnt.Path
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			stdin = f
			runner.executorStdin = f
		case "json":
			j, err := json.Marshal(mnt.Content)
			if err != nil {
				return fmt.Errorf("error encoding stdin json data: %v", err)
			}
			stdin = bytes.NewReader(j)
			runner.executorStdin = io.NopCloser(nil)
		default:
			return fmt.Errorf("stdin mount has unsupported kind %q", mnt.Kind)
		}
	} else {
		stdin = bytes.NewReader(nil)
		runner.executorStdin = ioutil.NopCloser(nil)
	}

	var stdout, stderr io.Writer
	if mnt, ok := runner.Container.Mounts["stdout"]; ok {
		f, err := runner.getStdoutFile(mnt.Path)
		if err != nil {
			return err
		}
		stdout = f
		runner.executorStdout = f
	} else if w, err := runner.openLogFile("stdout"); err != nil {
		return err
	} else {
		stdout = newTimestamper(w)
		runner.executorStdout = w
	}

	if mnt, ok := runner.Container.Mounts["stderr"]; ok {
		f, err := runner.getStdoutFile(mnt.Path)
		if err != nil {
			return err
		}
		stderr = f
		runner.executorStderr = f
	} else if w, err := runner.openLogFile("stderr"); err != nil {
		return err
	} else {
		stderr = newTimestamper(w)
		runner.executorStderr = w
	}

	env := runner.Container.Environment
	enableNetwork := runner.enableNetwork == "always"
	if runner.Container.RuntimeConstraints.API {
		enableNetwork = true
		tok, err := runner.ContainerToken()
		if err != nil {
			return err
		}
		env = map[string]string{}
		for k, v := range runner.Container.Environment {
			env[k] = v
		}
		env["ARVADOS_API_TOKEN"] = tok
		env["ARVADOS_API_HOST"] = os.Getenv("ARVADOS_API_HOST")
		env["ARVADOS_API_HOST_INSECURE"] = os.Getenv("ARVADOS_API_HOST_INSECURE")
		env["ARVADOS_KEEP_SERVICES"] = os.Getenv("ARVADOS_KEEP_SERVICES")
	}
	workdir := runner.Container.Cwd
	if workdir == "." {
		// both "" and "." mean default
		workdir = ""
	}
	ram := runner.Container.RuntimeConstraints.RAM
	if !runner.enableMemoryLimit {
		ram = 0
	}

	if runner.Container.RuntimeConstraints.CUDA.DeviceCount > 0 {
		nvidiaModprobe(runner.CrunchLog)
	}

	return runner.executor.Create(containerSpec{
		Image:           imageID,
		VCPUs:           runner.Container.RuntimeConstraints.VCPUs,
		RAM:             ram,
		WorkingDir:      workdir,
		Env:             env,
		BindMounts:      bindmounts,
		Command:         runner.Container.Command,
		EnableNetwork:   enableNetwork,
		CUDADeviceCount: runner.Container.RuntimeConstraints.CUDA.DeviceCount,
		NetworkMode:     runner.networkMode,
		CgroupParent:    runner.setCgroupParent,
		Stdin:           stdin,
		Stdout:          stdout,
		Stderr:          stderr,
	})
}

// StartContainer starts the docker container created by CreateContainer.
func (runner *ContainerRunner) StartContainer() error {
	runner.CrunchLog.Printf("Starting container")
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	if runner.cCancelled {
		return ErrCancelled
	}
	err := runner.executor.Start()
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
	runner.CrunchLog.Print("Waiting for container to finish")
	var timeout <-chan time.Time
	if s := runner.Container.SchedulingParameters.MaxRunTime; s > 0 {
		timeout = time.After(time.Duration(s) * time.Second)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-timeout:
			runner.CrunchLog.Printf("maximum run time exceeded. Stopping container.")
			runner.stop(nil)
		case <-runner.ArvMountExit:
			runner.CrunchLog.Printf("arv-mount exited while container is still running. Stopping container.")
			runner.stop(nil)
		case <-ctx.Done():
		}
	}()
	exitcode, err := runner.executor.Wait(ctx)
	if err != nil {
		runner.checkBrokenNode(err)
		return err
	}
	runner.ExitCode = &exitcode

	extra := ""
	if exitcode&0x80 != 0 {
		// Convert raw exit status (0x80 + signal number) to a
		// string to log after the code, like " (signal 101)"
		// or " (signal 9, killed)"
		sig := syscall.WaitStatus(exitcode).Signal()
		if name := unix.SignalName(sig); name != "" {
			extra = fmt.Sprintf(" (signal %d, %s)", sig, name)
		} else {
			extra = fmt.Sprintf(" (signal %d)", sig)
		}
	}
	runner.CrunchLog.Printf("Container exited with status code %d%s", exitcode, extra)
	err = runner.DispatcherArvClient.Update("containers", runner.Container.UUID, arvadosclient.Dict{
		"select":    []string{"uuid"},
		"container": arvadosclient.Dict{"exit_code": exitcode},
	}, nil)
	if err != nil {
		runner.CrunchLog.Printf("ignoring error updating exit_code: %s", err)
	}

	var returnErr error
	if err = runner.executorStdin.Close(); err != nil {
		err = fmt.Errorf("error closing container stdin: %s", err)
		runner.CrunchLog.Printf("%s", err)
		returnErr = err
	}
	if err = runner.executorStdout.Close(); err != nil {
		err = fmt.Errorf("error closing container stdout: %s", err)
		runner.CrunchLog.Printf("%s", err)
		if returnErr == nil {
			returnErr = err
		}
	}
	if err = runner.executorStderr.Close(); err != nil {
		err = fmt.Errorf("error closing container stderr: %s", err)
		runner.CrunchLog.Printf("%s", err)
		if returnErr == nil {
			returnErr = err
		}
	}

	if runner.statReporter != nil {
		runner.statReporter.Stop()
		runner.statReporter.LogMaxima(runner.CrunchLog, map[string]int64{
			"rss": runner.Container.RuntimeConstraints.RAM,
		})
		err = runner.statLogger.Close()
		if err != nil {
			runner.CrunchLog.Printf("error closing crunchstat logs: %v", err)
		}
	}
	return returnErr
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

		err = runner.DispatcherArvClient.Update("containers", runner.Container.UUID, arvadosclient.Dict{
			"select": []string{"uuid"},
			"container": arvadosclient.Dict{
				"log": saved.PortableDataHash,
			},
		}, nil)
		if err != nil {
			runner.CrunchLog.Printf("error updating container log to %s: %s", saved.PortableDataHash, err)
			continue
		}

		savedSize = size
	}
}

var spotInterruptionCheckInterval = 5 * time.Second
var ec2MetadataBaseURL = "http://169.254.169.254"

const ec2TokenTTL = time.Second * 21600

func (runner *ContainerRunner) checkSpotInterruptionNotices() {
	type ec2metadata struct {
		Action string    `json:"action"`
		Time   time.Time `json:"time"`
	}
	runner.CrunchLog.Printf("Checking for spot interruptions every %v using instance metadata at %s", spotInterruptionCheckInterval, ec2MetadataBaseURL)
	var metadata ec2metadata
	var token string
	var tokenExp time.Time
	check := func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
		defer cancel()
		if token == "" || tokenExp.Sub(time.Now()) < time.Minute {
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, ec2MetadataBaseURL+"/latest/api/token", nil)
			if err != nil {
				return err
			}
			req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", fmt.Sprintf("%d", int(ec2TokenTTL/time.Second)))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("%s", resp.Status)
			}
			newtoken, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			token = strings.TrimSpace(string(newtoken))
			tokenExp = time.Now().Add(ec2TokenTTL)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ec2MetadataBaseURL+"/latest/meta-data/spot/instance-action", nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-aws-ec2-metadata-token", token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		metadata = ec2metadata{}
		switch resp.StatusCode {
		case http.StatusOK:
			break
		case http.StatusNotFound:
			// "If Amazon EC2 is not preparing to stop or
			// terminate the instance, or if you
			// terminated the instance yourself,
			// instance-action is not present in the
			// instance metadata and you receive an HTTP
			// 404 error when you try to retrieve it."
			return nil
		case http.StatusUnauthorized:
			token = ""
			return fmt.Errorf("%s", resp.Status)
		default:
			return fmt.Errorf("%s", resp.Status)
		}
		err = json.NewDecoder(resp.Body).Decode(&metadata)
		if err != nil {
			return err
		}
		return nil
	}
	failures := 0
	var lastmetadata ec2metadata
	for range time.NewTicker(spotInterruptionCheckInterval).C {
		err := check()
		if err != nil {
			runner.CrunchLog.Printf("Error checking spot interruptions: %s", err)
			failures++
			if failures > 5 {
				runner.CrunchLog.Printf("Giving up on checking spot interruptions after too many consecutive failures")
				return
			}
			continue
		}
		failures = 0
		if metadata != lastmetadata {
			lastmetadata = metadata
			text := fmt.Sprintf("Cloud provider scheduled instance %s at %s", metadata.Action, metadata.Time.UTC().Format(time.RFC3339))
			runner.CrunchLog.Printf("%s", text)
			runner.updateRuntimeStatus(arvadosclient.Dict{
				"warning":          "preemption notice",
				"warningDetail":    text,
				"preemptionNotice": text,
			})
			if proc, err := os.FindProcess(os.Getpid()); err == nil {
				// trigger updateLogs
				proc.Signal(syscall.SIGUSR1)
			}
		}
	}
}

func (runner *ContainerRunner) updateRuntimeStatus(status arvadosclient.Dict) {
	err := runner.DispatcherArvClient.Update("containers", runner.Container.UUID, arvadosclient.Dict{
		"select": []string{"uuid"},
		"container": arvadosclient.Dict{
			"runtime_status": status,
		},
	}, nil)
	if err != nil {
		runner.CrunchLog.Printf("error updating container runtime_status: %s", err)
	}
}

// CaptureOutput saves data from the container's output directory if
// needed, and updates the container output accordingly.
func (runner *ContainerRunner) CaptureOutput(bindmounts map[string]bindmount) error {
	if runner.Container.RuntimeConstraints.API {
		// Output may have been set directly by the container, so
		// refresh the container record to check.
		err := runner.DispatcherArvClient.Get("containers", runner.Container.UUID,
			arvadosclient.Dict{
				"select": []string{"output"},
			}, &runner.Container)
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
		keepClient:    runner.ContainerKeepClient,
		hostOutputDir: runner.HostOutputDir,
		ctrOutputDir:  runner.Container.OutputPath,
		globs:         runner.Container.OutputGlob,
		bindmounts:    bindmounts,
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
		"select":             []string{"portable_data_hash"},
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
			runner.ArvMount.Process.Kill()
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
		runner.ArvMount = nil
	}

	if runner.ArvMountPoint != "" {
		if rmerr := os.Remove(runner.ArvMountPoint); rmerr != nil {
			runner.CrunchLog.Printf("While cleaning up arv-mount directory %s: %v", runner.ArvMountPoint, rmerr)
		}
		runner.ArvMountPoint = ""
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

		// From now on just log to stderr, in case there are
		// any other further errors (such as failing to write
		// the log to Keep!)  while shutting down
		runner.CrunchLog = newLogWriter(newTimestamper(newStringPrefixer(os.Stderr, runner.Container.UUID+" ")))
	}()

	if runner.keepstoreLogger != nil {
		// Flush any buffered logs from our local keepstore
		// process.  Discard anything logged after this point
		// -- it won't end up in the log collection, so
		// there's no point writing it to the collectionfs.
		runner.keepstoreLogbuf.SetWriter(io.Discard)
		runner.keepstoreLogger.Close()
		runner.keepstoreLogger = nil
	}

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

// Create/update the log collection. Return value has UUID and
// PortableDataHash fields populated, but others may be blank.
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
		// We set trash_at so this collection gets
		// automatically cleaned up eventually.  It used to be
		// 12 hours but we had a situation where the API
		// server was down over a weekend but the containers
		// kept running such that the log collection got
		// trashed, so now we make it 2 weeks.  refs #20378
		exp := time.Now().Add(time.Duration(24*14) * time.Hour)
		updates["trash_at"] = exp
		updates["delete_at"] = exp
	}
	reqBody := arvadosclient.Dict{
		"select":     []string{"uuid", "portable_data_hash"},
		"collection": updates,
	}
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
func (runner *ContainerRunner) UpdateContainerRunning(logId string) error {
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	if runner.cCancelled {
		return ErrCancelled
	}
	updates := arvadosclient.Dict{
		"gateway_address": runner.gateway.Address,
		"state":           "Running",
	}
	if logId != "" {
		updates["log"] = logId
	}
	return runner.DispatcherArvClient.Update(
		"containers",
		runner.Container.UUID,
		arvadosclient.Dict{
			"select":    []string{"uuid"},
			"container": updates,
		},
		nil,
	)
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
	if runner.ExitCode != nil {
		update["exit_code"] = *runner.ExitCode
	} else {
		update["exit_code"] = nil
	}
	if runner.finalState == "Complete" && runner.OutputPDH != nil {
		update["output"] = *runner.OutputPDH
	}
	update["cost"] = runner.calculateCost(time.Now())
	return runner.DispatcherArvClient.Update("containers", runner.Container.UUID, arvadosclient.Dict{
		"select":    []string{"uuid"},
		"container": update,
	}, nil)
}

// IsCancelled returns the value of Cancelled, with goroutine safety.
func (runner *ContainerRunner) IsCancelled() bool {
	runner.cStateLock.Lock()
	defer runner.cStateLock.Unlock()
	return runner.cCancelled
}

func (runner *ContainerRunner) openLogFile(name string) (io.WriteCloser, error) {
	return runner.LogCollection.OpenFile(name+".txt", os.O_CREATE|os.O_WRONLY, 0666)
}

// Run the full container lifecycle.
func (runner *ContainerRunner) Run() (err error) {
	runner.CrunchLog.Printf("crunch-run %s started", cmd.Version.String())
	runner.CrunchLog.Printf("%s", currentUserAndGroups())
	v, _ := exec.Command("arv-mount", "--version").CombinedOutput()
	runner.CrunchLog.Printf("Using FUSE mount: %s", v)
	runner.CrunchLog.Printf("Using container runtime: %s", runner.executor.Runtime())
	runner.CrunchLog.Printf("Executing container: %s", runner.Container.UUID)
	runner.costStartTime = time.Now()

	hostname, hosterr := os.Hostname()
	if hosterr != nil {
		runner.CrunchLog.Printf("Error getting hostname '%v'", hosterr)
	} else {
		runner.CrunchLog.Printf("Executing on host '%s'", hostname)
	}

	sigusr2 := make(chan os.Signal, 1)
	signal.Notify(sigusr2, syscall.SIGUSR2)
	defer signal.Stop(sigusr2)
	runner.loadPrices()
	go runner.handleSIGUSR2(sigusr2)

	runner.finalState = "Queued"

	defer func() {
		runner.CleanupDirs()
		runner.CrunchLog.Printf("crunch-run finished")
	}()

	err = runner.fetchContainerRecord()
	if err != nil {
		return
	}
	if runner.Container.State != "Locked" {
		return fmt.Errorf("dispatch error detected: container %q has state %q", runner.Container.UUID, runner.Container.State)
	}

	var bindmounts map[string]bindmount
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

		if bindmounts != nil {
			checkErr("CaptureOutput", runner.CaptureOutput(bindmounts))
		}
		checkErr("stopHoststat", runner.stopHoststat())
		checkErr("CommitLogs", runner.CommitLogs())
		runner.CleanupDirs()
		checkErr("UpdateContainerFinal", runner.UpdateContainerFinal())
	}()

	runner.setupSignals()
	err = runner.startHoststat()
	if err != nil {
		return
	}
	if runner.keepstore != nil {
		runner.hoststatReporter.ReportPID("keepstore", runner.keepstore.Process.Pid)
	}

	// set up FUSE mount and binds
	bindmounts, err = runner.SetupMounts()
	if err != nil {
		runner.finalState = "Cancelled"
		err = fmt.Errorf("While setting up mounts: %v", err)
		return
	}

	// check for and/or load image
	imageID, err := runner.LoadImage()
	if err != nil {
		if !runner.checkBrokenNode(err) {
			// Failed to load image but not due to a "broken node"
			// condition, probably user error.
			runner.finalState = "Cancelled"
		}
		err = fmt.Errorf("While loading container image: %v", err)
		return
	}

	err = runner.CreateContainer(imageID, bindmounts)
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

	logCollection, err := runner.saveLogCollection(false)
	var logId string
	if err == nil {
		logId = logCollection.PortableDataHash
	} else {
		runner.CrunchLog.Printf("Error committing initial log collection: %v", err)
	}
	err = runner.UpdateContainerRunning(logId)
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

	runner.ContainerKeepClient.SetStorageClasses(runner.Container.OutputStorageClasses)
	runner.DispatcherKeepClient.SetStorageClasses(runner.Container.OutputStorageClasses)

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
	containerUUID string) (*ContainerRunner, error) {

	cr := &ContainerRunner{
		dispatcherClient:     dispatcherClient,
		DispatcherArvClient:  dispatcherArvClient,
		DispatcherKeepClient: dispatcherKeepClient,
	}
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
	f, err := cr.openLogFile("crunch-run")
	if err != nil {
		return nil, err
	}
	cr.CrunchLog = newLogWriter(newTimestamper(io.MultiWriter(f, newStringPrefixer(os.Stderr, cr.Container.UUID+" "))))

	go cr.updateLogs()

	return cr, nil
}

func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	log := log.New(stderr, "", 0)
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	statInterval := flags.Duration("crunchstat-interval", 10*time.Second, "sampling period for periodic resource usage reporting")
	flags.String("cgroup-root", "/sys/fs/cgroup", "path to sysfs cgroup tree (obsolete, ignored)")
	flags.String("cgroup-parent", "docker", "name of container's parent cgroup (obsolete, ignored)")
	cgroupParentSubsystem := flags.String("cgroup-parent-subsystem", "", "use current cgroup for given `subsystem` as parent cgroup for container (subsystem argument is only relevant for cgroups v1; in cgroups v2 / unified mode, any non-empty value means use current cgroup); if empty, use the docker daemon's default cgroup parent. See https://doc.arvados.org/install/crunch2-slurm/install-dispatch.html#CrunchRunCommand-cgroups")
	caCertsPath := flags.String("ca-certs", "", "Path to TLS root certificates")
	detach := flags.Bool("detach", false, "Detach from parent process and run in the background")
	stdinConfig := flags.Bool("stdin-config", false, "Load config and environment variables from JSON message on stdin")
	configFile := flags.String("config", arvados.DefaultConfigFile, "filename of cluster config file to try loading if -stdin-config=false (default is $ARVADOS_CONFIG)")
	sleep := flags.Duration("sleep", 0, "Delay before starting (testing use only)")
	kill := flags.Int("kill", -1, "Send signal to an existing crunch-run process for given UUID")
	list := flags.Bool("list", false, "List UUIDs of existing crunch-run processes (and notify them to use price data passed on stdin)")
	enableMemoryLimit := flags.Bool("enable-memory-limit", true, "tell container runtime to limit container's memory usage")
	enableNetwork := flags.String("container-enable-networking", "default", "enable networking \"always\" (for all containers) or \"default\" (for containers that request it)")
	networkMode := flags.String("container-network-mode", "default", `Docker network mode for container (use any argument valid for docker --net)`)
	memprofile := flags.String("memprofile", "", "write memory profile to `file` after running container")
	runtimeEngine := flags.String("runtime-engine", "docker", "container runtime: docker or singularity")
	brokenNodeHook := flags.String("broken-node-hook", "", "script to run if node is detected to be broken (for example, Docker daemon is not running)")
	flags.Duration("check-containerd", 0, "Ignored. Exists for compatibility with older versions.")
	version := flags.Bool("version", false, "Write version information to stdout and exit 0.")

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

	if ok, code := cmd.ParseFlags(flags, prog, args, "container-uuid", stderr); !ok {
		return code
	} else if *version {
		fmt.Fprintln(stdout, prog, cmd.Version.String())
		return 0
	} else if !*list && flags.NArg() != 1 {
		fmt.Fprintf(stderr, "missing required argument: container-uuid (try -help)\n")
		return 2
	}

	containerUUID := flags.Arg(0)

	switch {
	case *detach && !ignoreDetachFlag:
		return Detach(containerUUID, prog, args, stdin, stdout, stderr)
	case *kill >= 0:
		return KillProcess(containerUUID, syscall.Signal(*kill), stdout, stderr)
	case *list:
		return ListProcesses(stdin, stdout, stderr)
	}

	if len(containerUUID) != 27 {
		log.Printf("usage: %s [options] UUID", prog)
		return 1
	}

	var keepstoreLogbuf bufThenWrite
	var conf ConfigData
	if *stdinConfig {
		err := json.NewDecoder(stdin).Decode(&conf)
		if err != nil {
			log.Printf("decode stdin: %s", err)
			return 1
		}
		for k, v := range conf.Env {
			err = os.Setenv(k, v)
			if err != nil {
				log.Printf("setenv(%q): %s", k, err)
				return 1
			}
		}
		if conf.Cluster != nil {
			// ClusterID is missing from the JSON
			// representation, but we need it to generate
			// a valid config file for keepstore, so we
			// fill it using the container UUID prefix.
			conf.Cluster.ClusterID = containerUUID[:5]
		}
	} else {
		conf = hpcConfData(containerUUID, *configFile, io.MultiWriter(&keepstoreLogbuf, stderr))
	}

	log.Printf("crunch-run %s started", cmd.Version.String())
	time.Sleep(*sleep)

	if *caCertsPath != "" {
		os.Setenv("SSL_CERT_FILE", *caCertsPath)
	}

	keepstore, err := startLocalKeepstore(conf, io.MultiWriter(&keepstoreLogbuf, stderr))
	if err != nil {
		log.Print(err)
		return 1
	}
	if keepstore != nil {
		defer keepstore.Process.Kill()
	}

	api, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Printf("%s: %v", containerUUID, err)
		return 1
	}
	// arvadosclient now interprets Retries=10 to mean
	// Timeout=10m, retrying with exponential backoff + jitter.
	api.Retries = 10

	kc, err := keepclient.MakeKeepClient(api)
	if err != nil {
		log.Printf("%s: %v", containerUUID, err)
		return 1
	}
	kc.Retries = 4

	cr, err := NewContainerRunner(arvados.NewClientFromEnv(), api, kc, containerUUID)
	if err != nil {
		log.Print(err)
		return 1
	}

	cr.keepstore = keepstore
	if keepstore == nil {
		// Log explanation (if any) for why we're not running
		// a local keepstore.
		var buf bytes.Buffer
		keepstoreLogbuf.SetWriter(&buf)
		if buf.Len() > 0 {
			cr.CrunchLog.Printf("%s", strings.TrimSpace(buf.String()))
		}
	} else if logWhat := conf.Cluster.Containers.LocalKeepLogsToContainerLog; logWhat == "none" {
		cr.CrunchLog.Printf("using local keepstore process (pid %d) at %s", keepstore.Process.Pid, os.Getenv("ARVADOS_KEEP_SERVICES"))
		keepstoreLogbuf.SetWriter(io.Discard)
	} else {
		cr.CrunchLog.Printf("using local keepstore process (pid %d) at %s, writing logs to keepstore.txt in log collection", keepstore.Process.Pid, os.Getenv("ARVADOS_KEEP_SERVICES"))
		cr.keepstoreLogger, err = cr.openLogFile("keepstore")
		if err != nil {
			log.Print(err)
			return 1
		}

		var writer io.WriteCloser = cr.keepstoreLogger
		if logWhat == "errors" {
			writer = &filterKeepstoreErrorsOnly{WriteCloser: writer}
		} else if logWhat != "all" {
			// should have been caught earlier by
			// dispatcher's config loader
			log.Printf("invalid value for Containers.LocalKeepLogsToContainerLog: %q", logWhat)
			return 1
		}
		err = keepstoreLogbuf.SetWriter(writer)
		if err != nil {
			log.Print(err)
			return 1
		}
		cr.keepstoreLogbuf = &keepstoreLogbuf
	}

	switch *runtimeEngine {
	case "docker":
		cr.executor, err = newDockerExecutor(containerUUID, cr.CrunchLog.Printf, cr.containerWatchdogInterval)
	case "singularity":
		cr.executor, err = newSingularityExecutor(cr.CrunchLog.Printf)
	default:
		cr.CrunchLog.Printf("%s: unsupported RuntimeEngine %q", containerUUID, *runtimeEngine)
		return 1
	}
	if err != nil {
		cr.CrunchLog.Printf("%s: %v", containerUUID, err)
		cr.checkBrokenNode(err)
		return 1
	}
	defer cr.executor.Close()

	cr.brokenNodeHook = *brokenNodeHook

	gwAuthSecret := os.Getenv("GatewayAuthSecret")
	os.Unsetenv("GatewayAuthSecret")
	if gwAuthSecret == "" {
		// not safe to run a gateway service without an auth
		// secret
		cr.CrunchLog.Printf("Not starting a gateway server (GatewayAuthSecret was not provided by dispatcher)")
	} else {
		gwListen := os.Getenv("GatewayAddress")
		cr.gateway = Gateway{
			Address:       gwListen,
			AuthSecret:    gwAuthSecret,
			ContainerUUID: containerUUID,
			Target:        cr.executor,
			Log:           cr.CrunchLog,
			LogCollection: cr.LogCollection,
		}
		if gwListen == "" {
			// Direct connection won't work, so we use the
			// gateway_address field to indicate the
			// internalURL of the controller process that
			// has the current tunnel connection.
			cr.gateway.ArvadosClient = cr.dispatcherClient
			cr.gateway.UpdateTunnelURL = func(url string) {
				cr.gateway.Address = "tunnel " + url
				cr.DispatcherArvClient.Update("containers", containerUUID,
					arvadosclient.Dict{
						"select":    []string{"uuid"},
						"container": arvadosclient.Dict{"gateway_address": cr.gateway.Address},
					}, nil)
			}
		}
		err = cr.gateway.Start()
		if err != nil {
			log.Printf("error starting gateway server: %s", err)
			return 1
		}
	}

	parentTemp, tmperr := cr.MkTempDir("", "crunch-run."+containerUUID+".")
	if tmperr != nil {
		log.Printf("%s: %v", containerUUID, tmperr)
		return 1
	}

	cr.parentTemp = parentTemp
	cr.statInterval = *statInterval
	cr.enableMemoryLimit = *enableMemoryLimit
	cr.enableNetwork = *enableNetwork
	cr.networkMode = *networkMode
	if *cgroupParentSubsystem != "" {
		p, err := findCgroup(os.DirFS("/"), *cgroupParentSubsystem)
		if err != nil {
			log.Printf("fatal: cgroup parent subsystem: %s", err)
			return 1
		}
		cr.setCgroupParent = p
	}

	if conf.EC2SpotCheck {
		go cr.checkSpotInterruptionNotices()
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
		log.Printf("%s: %v", containerUUID, runerr)
		return 1
	}
	return 0
}

// Try to load ConfigData in hpc (slurm/lsf) environment. This means
// loading the cluster config from the specified file and (if that
// works) getting the runtime_constraints container field from
// controller to determine # VCPUs so we can calculate KeepBuffers.
func hpcConfData(uuid string, configFile string, stderr io.Writer) ConfigData {
	var conf ConfigData
	conf.Cluster = loadClusterConfigFile(configFile, stderr)
	if conf.Cluster == nil {
		// skip loading the container record -- we won't be
		// able to start local keepstore anyway.
		return conf
	}
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		fmt.Fprintf(stderr, "error setting up arvadosclient: %s\n", err)
		return conf
	}
	// arvadosclient now interprets Retries=10 to mean
	// Timeout=10m, retrying with exponential backoff + jitter.
	arv.Retries = 10
	var ctr arvados.Container
	err = arv.Call("GET", "containers", uuid, "", arvadosclient.Dict{"select": []string{"runtime_constraints"}}, &ctr)
	if err != nil {
		fmt.Fprintf(stderr, "error getting container record: %s\n", err)
		return conf
	}
	if ctr.RuntimeConstraints.VCPUs > 0 {
		conf.KeepBuffers = ctr.RuntimeConstraints.VCPUs * conf.Cluster.Containers.LocalKeepBlobBuffersPerVCPU
	}
	return conf
}

// Load cluster config file from given path. If an error occurs, log
// the error to stderr and return nil.
func loadClusterConfigFile(path string, stderr io.Writer) *arvados.Cluster {
	ldr := config.NewLoader(&bytes.Buffer{}, ctxlog.New(stderr, "plain", "info"))
	ldr.Path = path
	cfg, err := ldr.Load()
	if err != nil {
		fmt.Fprintf(stderr, "could not load config file %s: %s\n", path, err)
		return nil
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		fmt.Fprintf(stderr, "could not use config file %s: %s\n", path, err)
		return nil
	}
	fmt.Fprintf(stderr, "loaded config file %s\n", path)
	return cluster
}

func startLocalKeepstore(configData ConfigData, logbuf io.Writer) (*exec.Cmd, error) {
	if configData.KeepBuffers < 1 {
		fmt.Fprintf(logbuf, "not starting a local keepstore process because KeepBuffers=%v in config\n", configData.KeepBuffers)
		return nil, nil
	}
	if configData.Cluster == nil {
		fmt.Fprint(logbuf, "not starting a local keepstore process because cluster config file was not loaded\n")
		return nil, nil
	}
	for uuid, vol := range configData.Cluster.Volumes {
		if len(vol.AccessViaHosts) > 0 {
			fmt.Fprintf(logbuf, "not starting a local keepstore process because a volume (%s) uses AccessViaHosts\n", uuid)
			return nil, nil
		}
		if !vol.ReadOnly && vol.Replication < configData.Cluster.Collections.DefaultReplication {
			fmt.Fprintf(logbuf, "not starting a local keepstore process because a writable volume (%s) has replication less than Collections.DefaultReplication (%d < %d)\n", uuid, vol.Replication, configData.Cluster.Collections.DefaultReplication)
			return nil, nil
		}
	}

	// Rather than have an alternate way to tell keepstore how
	// many buffers to use, etc., when starting it this way, we
	// just modify the cluster configuration that we feed it on
	// stdin.
	ccfg := *configData.Cluster
	ccfg.API.MaxKeepBlobBuffers = configData.KeepBuffers
	ccfg.Collections.BlobTrash = false
	ccfg.Collections.BlobTrashConcurrency = 0
	ccfg.Collections.BlobDeleteConcurrency = 0

	localaddr := localKeepstoreAddr()
	ln, err := net.Listen("tcp", net.JoinHostPort(localaddr, "0"))
	if err != nil {
		return nil, err
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, err
	}
	ln.Close()
	url := "http://" + net.JoinHostPort(localaddr, port)

	fmt.Fprintf(logbuf, "starting keepstore on %s\n", url)

	var confJSON bytes.Buffer
	err = json.NewEncoder(&confJSON).Encode(arvados.Config{
		Clusters: map[string]arvados.Cluster{
			ccfg.ClusterID: ccfg,
		},
	})
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("/proc/self/exe", "keepstore", "-config=-")
	if target, err := os.Readlink(cmd.Path); err == nil && strings.HasSuffix(target, ".test") {
		// If we're a 'go test' process, running
		// /proc/self/exe would start the test suite in a
		// child process, which is not what we want.
		cmd.Path, _ = exec.LookPath("go")
		cmd.Args = append([]string{"go", "run", "../../cmd/arvados-server"}, cmd.Args[1:]...)
		cmd.Env = os.Environ()
	}
	cmd.Stdin = &confJSON
	cmd.Stdout = logbuf
	cmd.Stderr = logbuf
	cmd.Env = append(cmd.Env,
		"GOGC=10",
		"ARVADOS_SERVICE_INTERNAL_URL="+url)
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting keepstore process: %w", err)
	}
	cmdExited := false
	go func() {
		cmd.Wait()
		cmdExited = true
	}()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
	defer cancel()
	poll := time.NewTicker(time.Second / 10)
	defer poll.Stop()
	client := http.Client{}
	for range poll.C {
		testReq, err := http.NewRequestWithContext(ctx, "GET", url+"/_health/ping", nil)
		testReq.Header.Set("Authorization", "Bearer "+configData.Cluster.ManagementToken)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(testReq)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if cmdExited {
			return nil, fmt.Errorf("keepstore child process exited")
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("timed out waiting for new keepstore process to report healthy")
		}
	}
	os.Setenv("ARVADOS_KEEP_SERVICES", url)
	return cmd, nil
}

// return current uid, gid, groups in a format suitable for logging:
// "crunch-run process has uid=1234(arvados) gid=1234(arvados)
// groups=1234(arvados),114(fuse)"
func currentUserAndGroups() string {
	u, err := user.Current()
	if err != nil {
		return fmt.Sprintf("error getting current user ID: %s", err)
	}
	s := fmt.Sprintf("crunch-run process has uid=%s(%s) gid=%s", u.Uid, u.Username, u.Gid)
	if g, err := user.LookupGroupId(u.Gid); err == nil {
		s += fmt.Sprintf("(%s)", g.Name)
	}
	s += " groups="
	if gids, err := u.GroupIds(); err == nil {
		for i, gid := range gids {
			if i > 0 {
				s += ","
			}
			s += gid
			if g, err := user.LookupGroupId(gid); err == nil {
				s += fmt.Sprintf("(%s)", g.Name)
			}
		}
	}
	return s
}

// Return a suitable local interface address for a local keepstore
// service. Currently this is the numerically lowest non-loopback ipv4
// address assigned to a local interface that is not in any of the
// link-local/vpn/loopback ranges 169.254/16, 100.64/10, or 127/8.
func localKeepstoreAddr() string {
	var ips []net.IP
	// Ignore error (proceed with zero IPs)
	addrs, _ := processIPs(os.Getpid())
	for addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			// invalid
			continue
		}
		if ip.Mask(net.CIDRMask(8, 32)).Equal(net.IPv4(127, 0, 0, 0)) ||
			ip.Mask(net.CIDRMask(10, 32)).Equal(net.IPv4(100, 64, 0, 0)) ||
			ip.Mask(net.CIDRMask(16, 32)).Equal(net.IPv4(169, 254, 0, 0)) {
			// unsuitable
			continue
		}
		ips = append(ips, ip)
	}
	if len(ips) == 0 {
		return "0.0.0.0"
	}
	sort.Slice(ips, func(ii, jj int) bool {
		i, j := ips[ii], ips[jj]
		if len(i) != len(j) {
			return len(i) < len(j)
		}
		for x := range i {
			if i[x] != j[x] {
				return i[x] < j[x]
			}
		}
		return false
	})
	return ips[0].String()
}

func (cr *ContainerRunner) loadPrices() {
	buf, err := os.ReadFile(filepath.Join(lockdir, pricesfile))
	if err != nil {
		if !os.IsNotExist(err) {
			cr.CrunchLog.Printf("loadPrices: read: %s", err)
		}
		return
	}
	var prices []cloud.InstancePrice
	err = json.Unmarshal(buf, &prices)
	if err != nil {
		cr.CrunchLog.Printf("loadPrices: decode: %s", err)
		return
	}
	cr.pricesLock.Lock()
	defer cr.pricesLock.Unlock()
	var lastKnown time.Time
	if len(cr.prices) > 0 {
		lastKnown = cr.prices[0].StartTime
	}
	cr.prices = cloud.NormalizePriceHistory(append(prices, cr.prices...))
	for i := len(cr.prices) - 1; i >= 0; i-- {
		price := cr.prices[i]
		if price.StartTime.After(lastKnown) {
			cr.CrunchLog.Printf("Instance price changed to %#.3g at %s", price.Price, price.StartTime.UTC())
		}
	}
}

func (cr *ContainerRunner) calculateCost(now time.Time) float64 {
	cr.pricesLock.Lock()
	defer cr.pricesLock.Unlock()

	// First, make a "prices" slice with the real data as far back
	// as it goes, and (if needed) a "since the beginning of time"
	// placeholder containing a reasonable guess about what the
	// price was between cr.costStartTime and the earliest real
	// data point.
	prices := cr.prices
	if len(prices) == 0 {
		// use price info in InstanceType record initially
		// provided by cloud dispatcher
		var p float64
		var it arvados.InstanceType
		if j := os.Getenv("InstanceType"); j != "" && json.Unmarshal([]byte(j), &it) == nil && it.Price > 0 {
			p = it.Price
		}
		prices = []cloud.InstancePrice{{Price: p}}
	} else if prices[len(prices)-1].StartTime.After(cr.costStartTime) {
		// guess earlier pricing was the same as the earliest
		// price we know about
		filler := prices[len(prices)-1]
		filler.StartTime = time.Time{}
		prices = append(prices, filler)
	}

	// Now that our history of price changes goes back at least as
	// far as cr.costStartTime, add up the costs for each
	// interval.
	cost := 0.0
	spanEnd := now
	for _, ip := range prices {
		spanStart := ip.StartTime
		if spanStart.After(now) {
			// pricing information from the future -- not
			// expected from AWS, but possible in
			// principle, and exercised by tests.
			continue
		}
		last := false
		if spanStart.Before(cr.costStartTime) {
			spanStart = cr.costStartTime
			last = true
		}
		cost += ip.Price * spanEnd.Sub(spanStart).Seconds() / 3600
		if last {
			break
		}
		spanEnd = spanStart
	}

	return cost
}

func (runner *ContainerRunner) handleSIGUSR2(sigchan chan os.Signal) {
	for range sigchan {
		runner.loadPrices()
		update := arvadosclient.Dict{
			"select": []string{"uuid"},
			"container": arvadosclient.Dict{
				"cost": runner.calculateCost(time.Now()),
			},
		}
		runner.DispatcherArvClient.Update("containers", runner.Container.UUID, update, nil)
	}
}
