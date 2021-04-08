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
	Env    []string       // List of environment variable to set in the container "VALUE=KEY" format
	Cmd    []string       // Command to run when starting the container
	Mounts []volumeMounts // List of Mounts used for the container.

}

// ContainerExecuter interface will implement all the methods needed for Docker,
// Singularity, and others to load images, run containers, get thier outputs,
// etc.
type ContainerExecuter interface {
	CheckImageIsLoaded(imageID ImageID) bool
	LoadImage(containerImage io.Reader) error
	ImageRemove(imageID ImageID) error

	ContainerState() (ContainerState, error)

	// CreateContainer will prepare anything in the creation on the container
	// env is an array of string "KEY=VALUE" that represents the environment variables
	// containerConfig has all the parameters needed to start the container
	// in the past we also had
	// - volumes: Now this is ExecOptions.mounts
	// - hostConfig: Now this is ExecOptions.enableNetwork and others
	// this are called when StartContainer() is executed.
	// this will also do any prep work needed to Std*Pipe() to work
	CreateContainer(env []string, execOptions ExecOptions, containerConfig ContainerConfig) error

	// StartContainer will start the container little questions asked
	StartContainer() error

	// Kill the container and optionally remove the underlying image returns an
	// error if it didn't work (including timeout)
	Kill() error

	// this is similar how https://golang.org/pkg/os/exec/#Cmd does it.
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)

	// Wait for the container to finish
	Wait()

	// Returns the exit coui
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

// ExecOptions are the options used in StartContainer()
type ExecOptions struct {
	enableNetworking bool
	mounts           []volumeMounts
}

type volumeMounts struct {
	hostPath      string
	containerPath string
	readonly      bool
}
