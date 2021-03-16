// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"io"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockernetwork "github.com/docker/docker/api/types/network"
	"golang.org/x/net/context"
)

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
