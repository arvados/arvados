// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type SingularityClient struct {
	exec.Cmd
	containerConfig     ContainerConfig
	imageSIFLocation    ImageID
	scratchSIFDirectory string
}

func (c *SingularityClient) CheckImageIsLoaded(imageID ImageID) bool {
	if c.imageSIFLocation == "" {
		return false
	}
	if _, err := os.Stat(string(c.imageSIFLocation)); err != nil {
		return false
	}
	return true

}

// LoadImage will satisfy ContainerExecuter interface transforming
// containerImage into a sif file for later use.
func (c *SingularityClient) LoadImage(containerImage io.Reader) (ImageID, error) {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	tempFilePrefix := filepath.Join(c.scratchSIFDirectory, hex.EncodeToString(randBytes)) //ioutil.TempFile maybe is better?

	sifFile := tempFilePrefix + ".sif"
	tarFile := tempFilePrefix + ".tar"

	f, err := os.OpenFile(tarFile, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return "", err
	}
	io.Copy(f, containerImage)
	err = f.Close()
	if err != nil {
		return "", err
	}
	archiveRef := fmt.Sprintf("docker-archive://%s", tarFile)

	buildCommand := exec.Cmd{
		Path: "/usr/bin/singularity",
		Args: []string{"/usr/bin/singularity", "build", sifFile, archiveRef},
	}

	buildCommand.Stdin = containerImage
	var out bytes.Buffer
	buildCommand.Stdout = &out
	var errBuf bytes.Buffer
	buildCommand.Stderr = &errBuf

	err = buildCommand.Run()
	if err != nil {
		fmt.Printf("%#v", buildCommand.Args)
		fmt.Printf("OUTPUT:'%s'\n", out.String())
		fmt.Printf("ERROR:'%s'\n", errBuf.String())
		return ImageID(""), err
	}
	//review: should we checkout out "out" for extra errors?
	// this is roughfly a successful output in singulariry
	// INFO:    Starting build...
	// Getting image source signatures
	// Copying blob ab15617702de done
	// Copying config 651e02b8a2 done
	// Writing manifest to image destination
	// Storing signatures
	// 2021/04/22 14:42:14  info unpack layer: sha256:21cbfd3a344c52b197b9fa36091e66d9cbe52232703ff78d44734f85abb7ccd3
	// INFO:    Creating SIF file...
	// INFO:    Build complete: arvados-jobs.latest.sif
	if err := os.Remove(string(tarFile)); err != nil {
		return ImageID(""), err
	}
	c.imageSIFLocation = ImageID(sifFile)

	return c.imageSIFLocation, nil
}

func (c *SingularityClient) ImageRemove(imageID ImageID) error {
	if c.imageSIFLocation == "" {
		return errors.New("No image loaded")
	}

	if _, err := os.Stat(string(c.imageSIFLocation)); err != nil {
		return fmt.Errorf("Image '%s' is invalid", string(imageID))
	}

	if err := os.Remove(string(c.imageSIFLocation)); err != nil {
		return err
	}
	c.imageSIFLocation = ""
	return nil
}

func (c *SingularityClient) ContainerState() (ContainerState, error) {

	return ContainerState{}, nil
}
func (c *SingularityClient) CreateContainer(containerConfig ContainerConfig) error {
	c.containerConfig = containerConfig
	// maybe we should ne doing extra checks if mounts don't exist for example
	return nil
}

func (c *SingularityClient) StartContainer() error {
	c.Cmd.Path = "/usr/bin/singularity"
	if c.containerConfig.EnableNetworking {
		c.Cmd.Args = []string{"/usr/bin/singularity", "run", "--contain", string(c.imageSIFLocation)}
	} else {
		c.Cmd.Args = []string{"/usr/bin/singularity", "run", "--nonet", "--contain"}
	}

	for _, mounts := range c.containerConfig.Mounts {
		var opt string
		if mounts.readonly {
			opt = "ro"
		} else {
			opt = "rw"
		}
		// this might need coma separated
		c.Cmd.Args = append(c.Cmd.Args, "--bind", fmt.Sprintf("%s:%s:%s", mounts.hostPath, mounts.containerPath, opt))
	}
	c.Cmd.Args = append(c.Cmd.Args, string(c.imageSIFLocation)) // image should be the last parameter before the external command
	c.Cmd.Args = append(c.Cmd.Args, c.containerConfig.Cmd...)

	// keyEqualsValuePair is assumed to be strings in the form "foo=bar" to
	// inject in the container environment
	for _, keyEqualsValuePair := range c.containerConfig.Env {
		// there are some behaviours that changed in singularity 3.6, please see:
		// https://sylabs.io/guides/3.7/user-guide/environment_and_metadata.html
		// https://sylabs.io/guides/3.5/user-guide/environment_and_metadata.html
		c.Cmd.Env = append(c.Cmd.Env, fmt.Sprintf("SINGULARITYENV_%s", keyEqualsValuePair))
	}
	return c.Cmd.Start()
}

func (c *SingularityClient) Kill() error {
	return nil
}

/*
func (c *SingularityClient) StdinPipe() (io.WriteCloser, error) {
	var x io.WriteCloser = (*os.File)(nil)
	return x, nil
}

func (c *SingularityClient) StdoutPipe() (io.ReadCloser, error) {
	var x io.ReadCloser = (*os.File)(nil)
	return x, nil
}

func (c *SingularityClient) StderrPipe() (io.ReadCloser, error) {
	var x io.ReadCloser = (*os.File)(nil)
	return x, nil
}
func (c *SingularityClient) Wait() {
}
*/

func (c *SingularityClient) ExitCode() (int, error) {
	return c.ProcessState.ExitCode(), nil
}

// NewSingularityClient creates  client that satisfy the ContainerExecuter interface
// to install singularity in ArvBox (debian buster) do the following:
// echo 'deb http://httpredir.debian.org/debian unstable main' > /etc/apt/sources.list.d/debian-unstable.list
// cat > /etc/apt/preferences.d/pin-stable << EOF
// Package: *
// Pin: release stable
// Pin-Priority: 600
// EOF
// apt update
// apt install -y singularity-container/unstable

func NewSingularityClient() (*SingularityClient, error) {
	var s = &SingularityClient{}
	// TODO: define if this should be a parameter.
	s.scratchSIFDirectory = os.TempDir()
	return s, nil
}
