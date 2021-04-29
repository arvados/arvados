// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
package crunchrun

import (
	"io"
)

// ContainerID is either Docker Container ID or Singularity something that can
// be tracked down with singularity instance list or anything that casn be
// referenced as "instance://instance1"
type ContainerID string

// ImageID is Docker Image ID or a "unique reference" in Singularity like a path
// to the image. This is used to remove the image from disk.
type ImageID string

// ContainerStateStatus represents the status of our container
type ContainerStateStatus int

const (
	Created ContainerStateStatus = iota
	Running
	Paused
	Restarting
	Removing
	Exited
	Dead
	OOMKilled
)

func (c ContainerStateStatus) String() string {
	return [...]string{"Created",
		"Running",
		"Paused",
		"Restarting",
		"Removing",
		"Exited",
		"Dead",
		"OOMKilled",
	}[c]
}

// ContainerState stores container's running state. In Docker is a subset of
// ContainerJSONBase returned by ContainerInspect. In Singularity TBD how we'll
// get the state of a running/dead container.
type ContainerState struct {
	Status     ContainerStateStatus
	Pid        int
	ExitCode   int
	Error      string
	StartedAt  string
	FinishedAt string
}

type ContainerConfig struct {
	Env              []string       // List of environment variable to set in the container "VALUE=KEY" format
	Cmd              []string       // Command to run when starting the container
	Mounts           []volumeMounts // List of Mounts used for the container.
	EnableNetworking bool           // This will allow the container to reach the outside world
}

// ContainerExecuter interface will implement all the methods needed for Docker,
// Singularity, and others to load images, run containers, get thier outputs,
// etc.
type ContainerExecuter interface {
	// CheckImageIsLoaded checks if imageID is already in the local environment,
	// either something we can reference later in Docker API or similar, or a
	// filepath containing the image to be used
	CheckImageIsLoaded(imageID ImageID) bool

	// LoadImage translates a io.Reader that has a tarball in docker format
	// (Usually created with 'docker save'), and load it to the local
	// ContainerExecuter.
	//
	// Returns an ImageID to be referenced later, could be an identifier or a
	// filepath or something else
	LoadImage(containerImage io.Reader) (ImageID, error)

	// ImageRemove removes the image loaded using LoadImage()
	ImageRemove(imageID ImageID) error

	ContainerState() (ContainerState, error)

	// CreateContainer prepares anything in the creation on the container
	// env is an array of string "KEY=VALUE" that represents the environment variables
	// containerConfig has all the parameters needed to start the container
	CreateContainer(containerConfig ContainerConfig) error

	// StartContainer starts the container
	StartContainer() error

	// Kill the container and optionally remove the underlying image returns an
	// error  including timeout errors
	Kill() error

	// This is how https://golang.org/pkg/os/exec/#Cmd does it.
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)

	// Wait for the container to finish
	Wait() error

	// Returns the exit code of the last executed container
	ExitCode() (int, error)
}

// ContainerExec is a helper that implements ContainerExecuter interface and is
// meant be be composed into the implementation as in:
//<
// type DockerExec struct {
//	ContainerExec
//
//  newDockerExec() *ContainerExecuter
//}
type ContainerExec struct {
	// TODO: add anything that could be useful for all ContainerExecs to
	// minimize the boilerplate for creating new ones.
}

type volumeMounts struct {
	hostPath      string
	containerPath string
	readonly      bool
}
