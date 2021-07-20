// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"golang.org/x/net/context"
)

type singularityExecutor struct {
	logf            func(string, ...interface{})
	spec            containerSpec
	tmpdir          string
	child           *exec.Cmd
	imageFilename   string // "sif" image
	containerClient *arvados.Client
	container       arvados.Container
	keepClient      IKeepClient
	keepMount       string
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

func (e *singularityExecutor) getOrCreateProject(ownerUuid string, name string, create bool) (*arvados.Group, error) {
	var gp arvados.GroupList
	err := e.containerClient.RequestAndDecode(&gp,
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
	if !create {
		return nil, nil
	}
	var rgroup arvados.Group
	err = e.containerClient.RequestAndDecode(&rgroup,
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

func (e *singularityExecutor) ImageLoaded(imageId string) bool {
	// Check if docker image is cached in keep & if so set imageFilename

	// Cache the image to keep
	cacheGroup, err := e.getOrCreateProject(e.container.RuntimeUserUUID, ".cache", false)
	if err != nil {
		e.logf("error getting '.cache' project: %v", err)
		return false
	}
	imageGroup, err := e.getOrCreateProject(cacheGroup.UUID, "auto-generated singularity images", false)
	if err != nil {
		e.logf("error getting 'auto-generated singularity images' project: %s", err)
		return false
	}

	collectionName := fmt.Sprintf("singularity image for %v", imageId)
	var cl arvados.CollectionList
	err = e.containerClient.RequestAndDecode(&cl,
		arvados.EndpointCollectionList.Method,
		arvados.EndpointCollectionList.Path,
		nil, arvados.ListOptions{Filters: []arvados.Filter{
			arvados.Filter{"owner_uuid", "=", imageGroup.UUID},
			arvados.Filter{"name", "=", collectionName},
		},
			Limit: 1})
	if err != nil {
		e.logf("error getting collection '%v' project: %v", err)
		return false
	}
	if len(cl.Items) == 0 {
		e.logf("no cached image '%v' found", collectionName)
		return false
	}

	path := fmt.Sprintf("%s/by_id/%s/image.sif", e.keepMount, cl.Items[0].PortableDataHash)
	e.logf("Looking for %v", path)
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return false
	}
	e.imageFilename = path

	return true
}

// LoadImage will satisfy ContainerExecuter interface transforming
// containerImage into a sif file for later use.
func (e *singularityExecutor) LoadImage(imageTarballPath string) error {
	if e.imageFilename != "" {
		e.logf("using singularity image %v", e.imageFilename)

		// was set by ImageLoaded
		return nil
	}

	e.logf("building singularity image")
	// "singularity build" does not accept a
	// docker-archive://... filename containing a ":" character,
	// as in "/path/to/sha256:abcd...1234.tar". Workaround: make a
	// symlink that doesn't have ":" chars.
	err := os.Symlink(imageTarballPath, e.tmpdir+"/image.tar")
	if err != nil {
		return err
	}
	e.imageFilename = e.tmpdir + "/image.sif"
	build := exec.Command("singularity", "build", e.imageFilename, "docker-archive://"+e.tmpdir+"/image.tar")
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
	if err != nil {
		return err
	}

	// Cache the image to keep
	cacheGroup, err := e.getOrCreateProject(e.container.RuntimeUserUUID, ".cache", true)
	if err != nil {
		e.logf("error getting '.cache' project: %v", err)
		return nil
	}
	imageGroup, err := e.getOrCreateProject(cacheGroup.UUID, "auto-generated singularity images", true)
	if err != nil {
		e.logf("error getting 'auto-generated singularity images' project: %v", err)
		return nil
	}

	parts := strings.Split(imageTarballPath, "/")
	imageId := parts[len(parts)-1]
	if strings.HasSuffix(imageId, ".tar") {
		imageId = imageId[0 : len(imageId)-4]
	}

	fs, err := (&arvados.Collection{ManifestText: ""}).FileSystem(e.containerClient, e.keepClient)
	if err != nil {
		e.logf("error creating FileSystem: %s", err)
	}

	dst, err := fs.OpenFile("image.sif", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		e.logf("error creating opening collection file for writing: %s", err)
	}

	src, err := os.Open(e.imageFilename)
	if err != nil {
		dst.Close()
		return nil
	}
	defer src.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		dst.Close()
		return nil
	}

	manifestText, err := fs.MarshalManifest(".")
	if err != nil {
		e.logf("error creating manifest text: %s", err)
	}

	var imageCollection arvados.Collection
	collectionName := fmt.Sprintf("singularity image for %s", imageId)
	err = e.containerClient.RequestAndDecode(&imageCollection,
		arvados.EndpointCollectionCreate.Method,
		arvados.EndpointCollectionCreate.Path,
		nil, map[string]interface{}{
			"collection": map[string]string{
				"owner_uuid":    imageGroup.UUID,
				"name":          collectionName,
				"manifest_text": manifestText,
			},
		})
	if err != nil {
		e.logf("error creating '%v' collection: %s", collectionName, err)
	}

	return nil
}

func (e *singularityExecutor) Create(spec containerSpec) error {
	e.spec = spec
	return nil
}

func (e *singularityExecutor) Start() error {
	args := []string{"singularity", "exec", "--containall", "--no-home", "--cleanenv", "--pwd", e.spec.WorkingDir}
	if !e.spec.EnableNetwork {
		args = append(args, "--net", "--network=none")
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
		args = append(args, "--bind", mount.HostPath+":"+path+":"+readonlyflag[mount.ReadOnly])
	}

	// This is for singularity 3.5.2. There are some behaviors
	// that will change in singularity 3.6, please see:
	// https://sylabs.io/guides/3.7/user-guide/environment_and_metadata.html
	// https://sylabs.io/guides/3.5/user-guide/environment_and_metadata.html
	env := make([]string, 0, len(e.spec.Env))
	for k, v := range e.spec.Env {
		if k == "HOME" {
			// $HOME is a special case
			args = append(args, "--home="+v)
		} else {
			env = append(env, "SINGULARITYENV_"+k+"="+v)
		}
	}

	args = append(args, e.imageFilename)
	args = append(args, e.spec.Command...)

	path, err := exec.LookPath(args[0])
	if err != nil {
		return err
	}
	child := &exec.Cmd{
		Path:   path,
		Args:   args,
		Env:    env,
		Stdin:  e.spec.Stdin,
		Stdout: e.spec.Stdout,
		Stderr: e.spec.Stderr,
	}
	err = child.Start()
	if err != nil {
		return err
	}
	e.child = child
	return nil
}

func (e *singularityExecutor) CgroupID() string {
	return ""
}

func (e *singularityExecutor) Stop() error {
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

func (e *singularityExecutor) SetArvadoClient(containerClient *arvados.Client, keepClient IKeepClient, container arvados.Container, keepMount string) {
	e.containerClient = containerClient
	e.container = container
	e.keepClient = keepClient
	e.keepMount = keepMount
}
