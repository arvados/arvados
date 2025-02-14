// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type singularityExecutor struct {
	logf          func(string, ...interface{})
	sudo          bool // use sudo to run singularity (only used by tests)
	spec          containerSpec
	tmpdir        string
	child         *exec.Cmd
	imageFilename string // "sif" image
}

func newSingularityExecutor(logf func(string, ...interface{})) (*singularityExecutor, error) {
	tmpdir, err := ioutil.TempDir("", "crunch-run-singularity-")
	if err != nil {
		return nil, err
	}
	return &singularityExecutor{
		logf:   logf,
		tmpdir: tmpdir,
	}, nil
}

func (e *singularityExecutor) Runtime() string {
	buf, err := exec.Command("singularity", "--version").CombinedOutput()
	if err != nil {
		return "singularity (unknown version)"
	}
	return strings.TrimSuffix(string(buf), "\n")
}

func (e *singularityExecutor) getOrCreateProject(ownerUuid string, name string, containerClient *arvados.Client) (*arvados.Group, error) {
	var gp arvados.GroupList
	err := containerClient.RequestAndDecode(&gp,
		arvados.EndpointGroupList.Method,
		arvados.EndpointGroupList.Path,
		nil, arvados.ListOptions{Filters: []arvados.Filter{
			arvados.Filter{"owner_uuid", "=", ownerUuid},
			arvados.Filter{"name", "=", name},
			arvados.Filter{"group_class", "=", "project"},
		},
			Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(gp.Items) == 1 {
		return &gp.Items[0], nil
	}

	var rgroup arvados.Group
	err = containerClient.RequestAndDecode(&rgroup,
		arvados.EndpointGroupCreate.Method,
		arvados.EndpointGroupCreate.Path,
		nil, map[string]interface{}{
			"group": map[string]string{
				"owner_uuid":  ownerUuid,
				"name":        name,
				"group_class": "project",
			},
		})
	if err != nil {
		return nil, err
	}
	return &rgroup, nil
}

func (e *singularityExecutor) getImageCacheProject(userUUID string, containerClient *arvados.Client) (*arvados.Group, error) {
	cacheProject, err := e.getOrCreateProject(userUUID, ".cache", containerClient)
	if err != nil {
		return nil, fmt.Errorf("error getting '.cache' project: %v", err)
	}
	imageProject, err := e.getOrCreateProject(cacheProject.UUID, "auto-generated singularity images", containerClient)
	if err != nil {
		return nil, fmt.Errorf("error getting 'auto-generated singularity images' project: %s", err)
	}
	return imageProject, nil
}

func (e *singularityExecutor) imageCacheExp() time.Time {
	return time.Now().Add(e.imageCacheTTL()).UTC()
}

func (e *singularityExecutor) imageCacheTTL() time.Duration {
	return 24 * 7 * 2 * time.Hour
}

// getCacheCollection returns an existing collection with a cached
// singularity image with the given name, or nil if none exists.
//
// Note that if there is no existing collection, this is not
// considered an error -- all return values will be nil/empty.
func (e *singularityExecutor) getCacheCollection(collectionName string, containerClient *arvados.Client, cacheProject *arvados.Group, arvMountPoint string) (collection *arvados.Collection, imageFile string, err error) {
	var cl arvados.CollectionList
	err = containerClient.RequestAndDecode(&cl,
		arvados.EndpointCollectionList.Method,
		arvados.EndpointCollectionList.Path,
		nil, arvados.ListOptions{Filters: []arvados.Filter{
			arvados.Filter{"owner_uuid", "=", cacheProject.UUID},
			arvados.Filter{"name", "=", collectionName},
		},
			Limit: 1})
	if err != nil {
		return nil, "", fmt.Errorf("error querying for collection %q in project %s: %w", collectionName, cacheProject.UUID, err)
	}
	if len(cl.Items) == 0 {
		// Successfully discovered that there's no cached
		// image collection.
		return nil, "", nil
	}
	// Check that the collection actually contains an "image.sif"
	// file.  If not, we can't use it, and trying to create a new
	// cache collection will probably fail too, so the caller
	// should not bother trying.
	coll := cl.Items[0]
	sifFile := path.Join(arvMountPoint, "by_id", coll.PortableDataHash, "image.sif")
	_, err = os.Stat(sifFile)
	if err != nil {
		return nil, "", fmt.Errorf("found collection %s (%s), but it did not contain an image file: %s", coll.UUID, coll.PortableDataHash, err)
	}
	if coll.TrashAt != nil && coll.TrashAt.Sub(time.Now()) < e.imageCacheTTL()*9/10 {
		// If the remaining TTL is less than 90% of our target
		// TTL, extend trash_at.  This avoids prematurely
		// trashing and re-converting images that are being
		// used regularly.
		err = containerClient.RequestAndDecode(nil,
			arvados.EndpointCollectionUpdate.Method,
			"arvados/v1/collections/"+coll.UUID,
			nil, map[string]interface{}{
				"collection": map[string]string{
					"trash_at": e.imageCacheExp().Format(time.RFC3339),
				},
			})
		if err != nil {
			e.logf("could not update expiry time of cached image collection (proceeding anyway): %s", err)
		}
	}
	return &coll, sifFile, nil
}

func (e *singularityExecutor) createCacheCollection(collectionName string, containerClient *arvados.Client, cacheProject *arvados.Group) (*arvados.Collection, error) {
	var coll arvados.Collection
	err := containerClient.RequestAndDecode(&coll,
		arvados.EndpointCollectionCreate.Method,
		arvados.EndpointCollectionCreate.Path,
		nil, map[string]interface{}{
			"collection": map[string]string{
				"owner_uuid": cacheProject.UUID,
				"name":       collectionName,
				"trash_at":   e.imageCacheExp().Format(time.RFC3339),
			},
			"ensure_unique_name": true,
		})
	if err != nil {
		return nil, fmt.Errorf("error creating '%v' collection: %s", collectionName, err)
	}
	return &coll, nil
}

func (e *singularityExecutor) convertDockerImage(srcPath, dstPath string) error {
	// Make sure the docker image is readable.
	if _, err := os.Stat(srcPath); err != nil {
		return err
	}

	e.logf("building singularity image")
	// "singularity build" does not accept a
	// docker-archive://... filename containing a ":" character,
	// as in "/path/to/sha256:abcd...1234.tar". Workaround: make a
	// symlink that doesn't have ":" chars.
	err := os.Symlink(srcPath, e.tmpdir+"/image.tar")
	if err != nil {
		return err
	}

	// Set up a cache and tmp dir for singularity build
	err = os.Mkdir(e.tmpdir+"/cache", 0700)
	if err != nil {
		return err
	}
	defer os.RemoveAll(e.tmpdir + "/cache")
	err = os.Mkdir(e.tmpdir+"/tmp", 0700)
	if err != nil {
		return err
	}
	defer os.RemoveAll(e.tmpdir + "/tmp")

	build := exec.Command("singularity", "build", dstPath, "docker-archive://"+e.tmpdir+"/image.tar")
	build.Env = os.Environ()
	build.Env = append(build.Env, "SINGULARITY_CACHEDIR="+e.tmpdir+"/cache")
	build.Env = append(build.Env, "SINGULARITY_TMPDIR="+e.tmpdir+"/tmp")
	e.logf("%v", build.Args)
	out, err := build.CombinedOutput()
	// INFO:    Starting build...
	// Getting image source signatures
	// Copying blob ab15617702de done
	// Copying config 651e02b8a2 done
	// Writing manifest to image destination
	// Storing signatures
	// 2021/04/22 14:42:14  info unpack layer: sha256:21cbfd3a344c52b197b9fa36091e66d9cbe52232703ff78d44734f85abb7ccd3
	// INFO:    Creating SIF file...
	// INFO:    Build complete: arvados-jobs.latest.sif
	e.logf("%s", out)
	return err
}

// LoadImage converts the given docker image to a singularity
// image.
//
// If containerClient is not nil, LoadImage first tries to use an
// existing image (in Home -> .cache -> auto-generated singularity
// images) and, if none was found there and the image was converted on
// the fly, tries to save the converted image to the cache so it can
// be reused next time.
//
// If containerClient is nil or a cache project/collection cannot be
// found or created, LoadImage converts the image on the fly and
// writes it to the local filesystem instead.
func (e *singularityExecutor) LoadImage(dockerImageID string, imageTarballPath string, container arvados.Container, arvMountPoint string, containerClient *arvados.Client) error {
	convertWithoutCache := func(err error) error {
		if err != nil {
			e.logf("cannot use singularity image cache: %s", err)
		}
		e.imageFilename = path.Join(e.tmpdir, "image.sif")
		return e.convertDockerImage(imageTarballPath, e.imageFilename)
	}

	if containerClient == nil {
		return convertWithoutCache(nil)
	}
	cacheProject, err := e.getImageCacheProject(container.RuntimeUserUUID, containerClient)
	if err != nil {
		return convertWithoutCache(err)
	}
	cacheCollectionName := fmt.Sprintf("singularity image for %s", dockerImageID)
	existingCollection, sifFile, err := e.getCacheCollection(cacheCollectionName, containerClient, cacheProject, arvMountPoint)
	if err != nil {
		return convertWithoutCache(err)
	}
	if existingCollection != nil {
		e.imageFilename = sifFile
		return nil
	}

	newCollection, err := e.createCacheCollection("converting "+cacheCollectionName, containerClient, cacheProject)
	if err != nil {
		return convertWithoutCache(err)
	}
	dstDir := path.Join(arvMountPoint, "by_uuid", newCollection.UUID)
	dstFile := path.Join(dstDir, "image.sif")
	err = e.convertDockerImage(imageTarballPath, dstFile)
	if err != nil {
		return err
	}
	buf, err := os.ReadFile(path.Join(dstDir, ".arvados#collection"))
	if err != nil {
		return fmt.Errorf("could not sync image collection: %w", err)
	}
	var synced arvados.Collection
	err = json.Unmarshal(buf, &synced)
	if err != nil {
		return fmt.Errorf("could not parse .arvados#collection: %w", err)
	}
	e.logf("saved converted image in %s with PDH %s", newCollection.UUID, synced.PortableDataHash)
	e.imageFilename = path.Join(arvMountPoint, "by_id", synced.PortableDataHash, "image.sif")

	if errRename := containerClient.RequestAndDecode(nil,
		arvados.EndpointCollectionUpdate.Method,
		"arvados/v1/collections/"+newCollection.UUID,
		nil, map[string]interface{}{
			"collection": map[string]string{
				"name": cacheCollectionName,
			},
		}); errRename != nil {
		// Error is probably a name collision caused by
		// another crunch-run process is converting the same
		// image concurrently.  In that case, we prefer to use
		// the one that won the race -- the resulting images
		// should be equivalent, but if they do differ at all,
		// it's better if all containers use the same
		// conversion.
		if existingCollection, sifFile, err := e.getCacheCollection(cacheCollectionName, containerClient, cacheProject, arvMountPoint); err == nil {
			e.logf("lost race -- abandoning our conversion in %s (%s) and using image from %s (%s) instead", newCollection.UUID, synced.PortableDataHash, existingCollection.UUID, existingCollection.PortableDataHash)
			e.imageFilename = sifFile
		} else {
			e.logf("using newly converted image anyway, despite error renaming collection: %v", errRename)
		}
	}
	return nil
}

func (e *singularityExecutor) Create(spec containerSpec) error {
	e.spec = spec
	return nil
}

func (e *singularityExecutor) execCmd(path string) *exec.Cmd {
	args := []string{path, "exec", "--containall", "--cleanenv", "--pwd=" + e.spec.WorkingDir}
	if !e.spec.EnableNetwork {
		args = append(args, "--net", "--network=none")
	} else if u, err := user.Current(); err == nil && u.Uid == "0" || e.sudo {
		// Specifying --network=bridge fails unless
		// singularity is running as root.
		//
		// Note this used to be possible with --fakeroot, or
		// configuring singularity like so:
		//
		// singularity config global --set 'allow net networks' bridge
		// singularity config global --set 'allow net groups' mygroup
		//
		// However, these options no longer work (as of debian
		// bookworm) because iptables now refuses to run in a
		// setuid environment.
		args = append(args, "--net", "--network=bridge")
	} else {
		// If we don't pass a --net argument at all, the
		// container will be in the same network namespace as
		// the host.
		//
		// Note this allows the container to listen on the
		// host's external ports.
	}
	if e.spec.GPUStack == "cuda" && e.spec.GPUDeviceCount > 0 {
		args = append(args, "--nv")
	}
	if e.spec.GPUStack == "rocm" && e.spec.GPUDeviceCount > 0 {
		args = append(args, "--rocm")
	}

	// If we ask for resource limits that aren't supported,
	// singularity will not run the container at all. So we probe
	// for support first, and only apply the limits that appear to
	// be supported.
	//
	// Default debian configuration lets non-root users set memory
	// limits but not CPU limits, so we enable/disable those
	// limits independently.
	//
	// https://rootlesscontaine.rs/getting-started/common/cgroup2/
	checkCgroupSupport(e.logf)
	if e.spec.VCPUs > 0 {
		if cgroupSupport["cpu"] {
			args = append(args, "--cpus", fmt.Sprintf("%d", e.spec.VCPUs))
		} else {
			e.logf("cpu limits are not supported by current systemd/cgroup configuration, not setting --cpu %d", e.spec.VCPUs)
		}
	}
	if e.spec.RAM > 0 {
		if cgroupSupport["memory"] {
			args = append(args, "--memory", fmt.Sprintf("%d", e.spec.RAM))
		} else {
			e.logf("memory limits are not supported by current systemd/cgroup configuration, not setting --memory %d", e.spec.RAM)
		}
	}

	readonlyflag := map[bool]string{
		false: "rw",
		true:  "ro",
	}
	var binds []string
	for path, _ := range e.spec.BindMounts {
		binds = append(binds, path)
	}
	sort.Strings(binds)
	for _, path := range binds {
		mount := e.spec.BindMounts[path]
		if path == e.spec.Env["HOME"] {
			// Singularity treats $HOME as special case
			args = append(args, "--home", mount.HostPath+":"+path)
		} else {
			args = append(args, "--bind", mount.HostPath+":"+path+":"+readonlyflag[mount.ReadOnly])
		}
	}

	// This is for singularity 3.5.2. There are some behaviors
	// that will change in singularity 3.6, please see:
	// https://sylabs.io/guides/3.7/user-guide/environment_and_metadata.html
	// https://sylabs.io/guides/3.5/user-guide/environment_and_metadata.html
	env := make([]string, 0, len(e.spec.Env))
	for k, v := range e.spec.Env {
		if k == "HOME" {
			// Singularity treats $HOME as special case,
			// this is handled with --home above
			continue
		}
		env = append(env, "SINGULARITYENV_"+k+"="+v)
	}

	// Singularity always makes all nvidia devices visible to the
	// container.  If a resource manager such as slurm or LSF told
	// us to select specific devices we need to propagate that.
	if cudaVisibleDevices := os.Getenv("CUDA_VISIBLE_DEVICES"); cudaVisibleDevices != "" {
		// If a resource manager such as slurm or LSF told
		// us to select specific devices we need to propagate that.
		env = append(env, "SINGULARITYENV_CUDA_VISIBLE_DEVICES="+cudaVisibleDevices)
	}
	// Singularity's default behavior is to evaluate each
	// SINGULARITYENV_* env var with a shell as a double-quoted
	// string and pass the result to the contained
	// process. Singularity 3.10+ has an option to pass env vars
	// through literally without evaluating, which is what we
	// want. See https://github.com/sylabs/singularity/pull/704
	// and https://dev.arvados.org/issues/19081
	env = append(env, "SINGULARITY_NO_EVAL=1")

	// If we don't propagate XDG_RUNTIME_DIR and
	// DBUS_SESSION_BUS_ADDRESS, singularity resource limits fail
	// with "FATAL: container creation failed: while applying
	// cgroups config: system configuration does not support
	// cgroup management" or "FATAL: container creation failed:
	// while applying cgroups config: rootless cgroups require a
	// D-Bus session - check that XDG_RUNTIME_DIR and
	// DBUS_SESSION_BUS_ADDRESS are set".
	env = append(env, "XDG_RUNTIME_DIR="+os.Getenv("XDG_RUNTIME_DIR"))
	env = append(env, "DBUS_SESSION_BUS_ADDRESS="+os.Getenv("DBUS_SESSION_BUS_ADDRESS"))

	args = append(args, e.imageFilename)
	args = append(args, e.spec.Command...)

	return &exec.Cmd{
		Path:   path,
		Args:   args,
		Env:    env,
		Stdin:  e.spec.Stdin,
		Stdout: e.spec.Stdout,
		Stderr: e.spec.Stderr,
	}
}

func (e *singularityExecutor) Start() error {
	path, err := exec.LookPath("singularity")
	if err != nil {
		return err
	}
	child := e.execCmd(path)
	if e.sudo {
		child.Args = append([]string{child.Path}, child.Args...)
		child.Path, err = exec.LookPath("sudo")
		if err != nil {
			return err
		}
	}
	err = child.Start()
	if err != nil {
		return err
	}
	e.child = child
	return nil
}

func (e *singularityExecutor) Pid() int {
	childproc, err := e.containedProcess()
	if err != nil {
		return 0
	}
	return childproc
}

func (e *singularityExecutor) Stop() error {
	if e.child == nil || e.child.Process == nil {
		// no process started, or Wait already called
		return nil
	}
	if err := e.child.Process.Signal(syscall.Signal(0)); err != nil {
		// process already exited
		return nil
	}
	return e.child.Process.Signal(syscall.SIGKILL)
}

func (e *singularityExecutor) Wait(context.Context) (int, error) {
	err := e.child.Wait()
	if err, ok := err.(*exec.ExitError); ok {
		return err.ProcessState.ExitCode(), nil
	}
	if err != nil {
		return 0, err
	}
	return e.child.ProcessState.ExitCode(), nil
}

func (e *singularityExecutor) Close() {
	err := os.RemoveAll(e.tmpdir)
	if err != nil {
		e.logf("error removing temp dir: %s", err)
	}
}

func (e *singularityExecutor) InjectCommand(ctx context.Context, detachKeys, username string, usingTTY bool, injectcmd []string) (*exec.Cmd, error) {
	target, err := e.containedProcess()
	if err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, "nsenter", append([]string{fmt.Sprintf("--target=%d", target), "--all"}, injectcmd...)...), nil
}

var (
	errContainerHasNoIPAddress = errors.New("container has no IP address distinct from host")
)

func (e *singularityExecutor) IPAddress() (string, error) {
	target, err := e.containedProcess()
	if err != nil {
		return "", err
	}
	targetIPs, err := processIPs(target)
	if err != nil {
		return "", err
	}
	selfIPs, err := processIPs(os.Getpid())
	if err != nil {
		return "", err
	}
	for ip := range targetIPs {
		if !selfIPs[ip] {
			return ip, nil
		}
	}
	return "", errContainerHasNoIPAddress
}

func processIPs(pid int) (map[string]bool, error) {
	fibtrie, err := os.ReadFile(fmt.Sprintf("/proc/%d/net/fib_trie", pid))
	if err != nil {
		return nil, err
	}

	addrs := map[string]bool{}
	// When we see a pair of lines like this:
	//
	//              |-- 10.1.2.3
	//                 /32 host LOCAL
	//
	// ...we set addrs["10.1.2.3"] = true
	lines := bytes.Split(fibtrie, []byte{'\n'})
	for linenumber, line := range lines {
		if !bytes.HasSuffix(line, []byte("/32 host LOCAL")) {
			continue
		}
		if linenumber < 1 {
			continue
		}
		i := bytes.LastIndexByte(lines[linenumber-1], ' ')
		if i < 0 || i >= len(line)-7 {
			continue
		}
		addr := string(lines[linenumber-1][i+1:])
		if net.ParseIP(addr).To4() != nil {
			addrs[addr] = true
		}
	}
	return addrs, nil
}

var (
	errContainerNotStarted = errors.New("container has not started yet")
	errCannotFindChild     = errors.New("failed to find any process inside the container")
	reProcStatusPPid       = regexp.MustCompile(`\nPPid:\t(\d+)\n`)
)

// Return the PID of a process that is inside the container (not
// necessarily the topmost/pid=1 process in the container).
func (e *singularityExecutor) containedProcess() (int, error) {
	if e.child == nil || e.child.Process == nil {
		return 0, errContainerNotStarted
	}
	cmd := exec.Command("lsns")
	if e.sudo {
		cmd = exec.Command("sudo", "lsns")
	}
	lsns, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("lsns: %w", err)
	}
	for _, line := range bytes.Split(lsns, []byte{'\n'}) {
		fields := bytes.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if !bytes.Equal(fields[1], []byte("pid")) {
			continue
		}
		pid, err := strconv.ParseInt(string(fields[3]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("error parsing PID field in lsns output: %q", fields[3])
		}
		for parent := pid; ; {
			procstatus, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", parent))
			if err != nil {
				break
			}
			m := reProcStatusPPid.FindSubmatch(procstatus)
			if m == nil {
				break
			}
			parent, err = strconv.ParseInt(string(m[1]), 10, 64)
			if err != nil {
				break
			}
			if int(parent) == e.child.Process.Pid {
				return int(pid), nil
			}
		}
	}
	return 0, errCannotFindChild
}
