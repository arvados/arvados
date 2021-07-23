// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"syscall"

	"golang.org/x/net/context"
)

type singularityExecutor struct {
	logf          func(string, ...interface{})
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

func (e *singularityExecutor) ImageLoaded(string) bool {
	return false
}

// LoadImage will satisfy ContainerExecuter interface transforming
// containerImage into a sif file for later use.
func (e *singularityExecutor) LoadImage(imageTarballPath string) error {
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
	args = append(args, e.imageFilename)
	args = append(args, e.spec.Command...)

	// This is for singularity 3.5.2. There are some behaviors
	// that will change in singularity 3.6, please see:
	// https://sylabs.io/guides/3.7/user-guide/environment_and_metadata.html
	// https://sylabs.io/guides/3.5/user-guide/environment_and_metadata.html
	env := make([]string, 0, len(e.spec.Env))
	for k, v := range e.spec.Env {
		env = append(env, "SINGULARITYENV_"+k+"="+v)
	}

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
