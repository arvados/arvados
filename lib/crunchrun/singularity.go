// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"fmt"
	"io"

	"golang.org/x/net/context"
)

type SingularityClient struct {
	containerConfig ContainerConfig
	hostConfig      HostConfig
}

func (c SingularityClient) GetContainerConfig() (ContainerConfig, error) {
	return c.containerConfig, nil
}

func (c SingularityClient) GetHostConfig() (HostConfig, error) {
	return c.hostConfig, nil
}

func (c SingularityClient) GetImage() (imageID string) {
	return c.containerConfig.Image
}

func (c SingularityClient) SetImage(imageID string) {
	c.containerConfig.Image = imageID
}

func (c SingularityClient) GetNetworkMode() (networkMode NetworkMode) {
	return c.hostConfig.NetworkMode
}

func (c SingularityClient) SetNetworkMode(networkMode NetworkMode) {
	c.hostConfig.NetworkMode = networkMode
}

func (c SingularityClient) SetHostConfig(hostConfig HostConfig) error {
	c.hostConfig = hostConfig
	return nil
}

func (c SingularityClient) ContainerAttach(ctx context.Context, container string, options ContainerAttachOptions) (HijackedResponse, error) {
	fmt.Printf("placeholder for container ContainerAttach %s", container)

	return HijackedResponse{}, nil
}

func (c SingularityClient) ContainerCreate(ctx context.Context, config ContainerConfig, hostConfig HostConfig, networkingConfig *NetworkingConfig, containerName string) (ContainerCreateResponse, error) {
	fmt.Printf("placeholder for container ContainerCreate %s", containerName)

	return ContainerCreateResponse{}, nil
}

func (c SingularityClient) ContainerStart(ctx context.Context, container string, options ContainerStartOptions) error {
	fmt.Printf("placeholder for container ContainerStart %s", container)

	return nil
}

func (c SingularityClient) ContainerRemove(ctx context.Context, container string, options ContainerRemoveOptions) error {
	fmt.Printf("placeholder for container ContainerRemove %s", container)

	return nil
}

func (c SingularityClient) ContainerWait(ctx context.Context, container string, condition WaitCondition) (<-chan ContainerWaitOKBody, <-chan error) {
	fmt.Printf("placeholder for ContainerWait")
	chanC := make(chan ContainerWaitOKBody)
	chanE := make(chan error)
	return chanC, chanE
}

func (c SingularityClient) ContainerInspect(ctx context.Context, id string) (ContainerInspectResponse, error) {
	fmt.Printf("placeholder for container ContainerInspect %s", id)

	return ContainerInspectResponse{}, nil
}

func (c SingularityClient) ImageInspectWithRaw(ctx context.Context, image string) (ImageInspectResponse, []byte, error) {
	fmt.Printf("placeholder for ImageInspectWithRaw() %s", image)

	return ImageInspectResponse{}, []byte(""), nil
}

func (c SingularityClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (ImageLoadResponse, error) {
	fmt.Printf("placeholder for ImageLoad")
	return ImageLoadResponse{}, nil
}

func (c SingularityClient) ImageRemove(ctx context.Context, image string, options ImageRemoveOptions) ([]ImageDeleteResponseItem, error) {
	fmt.Printf("placeholder for ImageRemove")
	var responses []ImageDeleteResponseItem
	tmp := ImageDeleteResponseItem{}
	responses = append(responses, tmp)
	return responses, nil
}

func NewSingularityClient() (SingularityClient, error) {
	var s = &SingularityClient{}
	return *s, nil
}
