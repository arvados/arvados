// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
package crunchrun

import (
	"io"

	"golang.org/x/net/context"
)

type bindmount struct {
	HostPath string
	ReadOnly bool
}

type containerSpec struct {
	Image         string
	VCPUs         int
	RAM           int64
	WorkingDir    string
	Env           map[string]string
	BindMounts    map[string]bindmount
	Command       []string
	EnableNetwork bool
	NetworkMode   string // docker network mode, normally "default"
	CgroupParent  string
	Stdin         io.ReadCloser
	Stdout        io.WriteCloser
	Stderr        io.WriteCloser
}

// containerExecutor is an interface to a container runtime
// (docker/singularity).
type containerExecutor interface {
	// ImageLoaded determines whether the given image is already
	// available to use without calling ImageLoad.
	ImageLoaded(imageID string) bool

	// ImageLoad loads the image from the given tarball such that
	// it can be used to create/start a container.
	LoadImage(filename string) error

	// Wait for the container process to finish, and return its
	// exit code. If applicable, also remove the stopped container
	// before returning.
	Wait(context.Context) (int, error)

	// Create a container, but don't start it yet.
	Create(containerSpec) error

	// Start the container
	Start() error

	// CID the container will belong to
	CgroupID() string

	// Stop the container immediately
	Stop() error

	// Release resources (temp dirs, stopped containers)
	Close()
}
