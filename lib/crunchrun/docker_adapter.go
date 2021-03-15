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

type DockerAdapter struct {
	docker          ThinDockerClient
	containerConfig ContainerConfig
	hostConfig      HostConfig
}

func (a *DockerAdapter) ContainerAttach(ctx context.Context, container string, options ContainerAttachOptions) (HijackedResponse, error) {
	dockerOptions := dockertypes.ContainerAttachOptions{
		Stream: options.Stream,
		Stdin:  options.Stdin,
		Stdout: options.Stdout,
		Stderr: options.Stderr}
	dockerResponse, docker_err := a.docker.ContainerAttach(ctx, container, dockerOptions)

	adapterResponse := HijackedResponse{
		Conn:   dockerResponse.Conn,
		Reader: dockerResponse.Reader,
	}

	return adapterResponse, docker_err
}

func (a *DockerAdapter) ContainerCreate(ctx context.Context, config ContainerConfig, hostConfig HostConfig, networkingConfig *NetworkingConfig, containerName string) (ContainerCreateResponse, error) {
	var dockerEndpointsConfig map[string]*dockernetwork.EndpointSettings

	var dockerNetworkConfig *dockernetwork.NetworkingConfig
	if networkingConfig != nil {
		for k, v := range networkingConfig.EndpointsConfig {

			dockerIpamConfig := &dockernetwork.EndpointIPAMConfig{
				IPv4Address:  v.IPAMConfig.IPv4Address,
				IPv6Address:  v.IPAMConfig.IPv6Address,
				LinkLocalIPs: v.IPAMConfig.LinkLocalIPs,
			}
			dockerEndpointsConfig[k] = &dockernetwork.EndpointSettings{
				IPAMConfig:          dockerIpamConfig,
				Links:               v.Links,
				Aliases:             v.Aliases,
				NetworkID:           v.NetworkID,
				EndpointID:          v.EndpointID,
				Gateway:             v.Gateway,
				IPAddress:           v.IPAddress,
				IPPrefixLen:         v.IPPrefixLen,
				IPv6Gateway:         v.IPv6Gateway,
				GlobalIPv6Address:   v.GlobalIPv6Address,
				GlobalIPv6PrefixLen: v.GlobalIPv6PrefixLen,
				MacAddress:          v.MacAddress,
				DriverOpts:          v.DriverOpts,
			}

		}

		dockerNetworkConfig = &dockernetwork.NetworkingConfig{
			EndpointsConfig: dockerEndpointsConfig,
		}
	}
	dockerConfig := dockercontainer.Config{
		OpenStdin:    config.OpenStdin,
		StdinOnce:    config.StdinOnce,
		AttachStdin:  config.AttachStdin,
		AttachStdout: config.AttachStdout,
		AttachStderr: config.AttachStderr,
		Cmd:          config.Cmd,
		WorkingDir:   config.WorkingDir,
		Env:          config.Env,
		Volumes:      config.Volumes,
	}
	dockerHostConfig := dockercontainer.HostConfig{}

	dockerResponse, dockerErr := a.docker.ContainerCreate(ctx,
		&dockerConfig,
		&dockerHostConfig, dockerNetworkConfig, containerName)
	adapterResponse := ContainerCreateResponse{
		ID:       dockerResponse.ID,
		Warnings: dockerResponse.Warnings,
	}
	return adapterResponse, dockerErr
}

func (a *DockerAdapter) ContainerStart(ctx context.Context, container string, options ContainerStartOptions) error {
	dockerContainerStartOptions := dockertypes.ContainerStartOptions{
		CheckpointID:  options.CheckpointID,
		CheckpointDir: options.CheckpointDir,
	}

	dockerErr := a.docker.ContainerStart(ctx, container, dockerContainerStartOptions)

	return dockerErr
}

func (a *DockerAdapter) ContainerRemove(ctx context.Context, container string, options ContainerRemoveOptions) error {
	dockerContainerRemoveOptions := dockertypes.ContainerRemoveOptions{Force: options.Force}

	dockerErr := a.docker.ContainerRemove(ctx, container, dockerContainerRemoveOptions)

	return dockerErr
}

func (a *DockerAdapter) ContainerInspect(ctx context.Context, id string) (ContainerInspectResponse, error) {

	dockerContainerInspectResponse, dockerErr := a.docker.ContainerInspect(ctx, id)

	containerState := &ContainerState{
		Running:    dockerContainerInspectResponse.State.Running,
		Paused:     dockerContainerInspectResponse.State.Paused,
		Restarting: dockerContainerInspectResponse.State.Restarting,
		OOMKilled:  dockerContainerInspectResponse.State.OOMKilled,
		Dead:       dockerContainerInspectResponse.State.Dead,
		Pid:        dockerContainerInspectResponse.State.Pid,
		ExitCode:   dockerContainerInspectResponse.State.ExitCode,
		Error:      dockerContainerInspectResponse.State.Error,
		StartedAt:  dockerContainerInspectResponse.State.StartedAt,
		FinishedAt: dockerContainerInspectResponse.State.FinishedAt,
	}

	adapterResponse := &ContainerInspectResponse{State: containerState}

	return *adapterResponse, dockerErr
}

func (a *DockerAdapter) ContainerWait(ctx context.Context, container string, condition WaitCondition) (<-chan ContainerWaitOKBody, <-chan error) {

	//var dockercontainerCondition dockercontainer.WaitCondition =
	dockercontainerCondition := dockercontainer.WaitCondition(condition)

	dockerContainerWaitOKBody, dockerErr := a.docker.ContainerWait(ctx, container, dockercontainerCondition)

	// translate from <-chan dockercontainer.ContainerWaitOKBody to <-chan ContainerWaitOKBody,
	adapterContainerWaitOKBody := make(chan ContainerWaitOKBody)
	go func() {
		for dockerMsg := range dockerContainerWaitOKBody {
			var adapterBodyMsg *ContainerWaitOKBody
			var adapterError *ContainerWaitOKBodyError

			if dockerMsg.Error != nil {
				adapterError = &ContainerWaitOKBodyError{
					Message: dockerMsg.Error.Message,
				}
			}

			adapterBodyMsg = &ContainerWaitOKBody{
				Error:      adapterError,
				StatusCode: dockerMsg.StatusCode,
			}

			adapterContainerWaitOKBody <- *adapterBodyMsg
		}
	}()

	return adapterContainerWaitOKBody, dockerErr
}

func (a *DockerAdapter) ImageInspectWithRaw(ctx context.Context, image string) (ImageInspectResponse, []byte, error) {
	dockerImageInspectResponse, rawBytes, dockerErr := a.docker.ImageInspectWithRaw(ctx, image)

	adapterImageInspectResponse := &ImageInspectResponse{
		ID: dockerImageInspectResponse.ID,
	}
	return *adapterImageInspectResponse, rawBytes, dockerErr
}

func (a *DockerAdapter) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (ImageLoadResponse, error) {
	dockerImageLoadResponse, dockerErr := a.docker.ImageLoad(ctx, input, quiet)

	adapterImageLoadResponse := &ImageLoadResponse{
		Body: dockerImageLoadResponse.Body,
		JSON: dockerImageLoadResponse.JSON,
	}

	return *adapterImageLoadResponse, dockerErr
}

func (a *DockerAdapter) ImageRemove(ctx context.Context, image string, options ImageRemoveOptions) ([]ImageDeleteResponseItem, error) {

	dockerOptions := &dockertypes.ImageRemoveOptions{
		Force:         options.Force,
		PruneChildren: options.PruneChildren,
	}
	dockerImageDeleteResponseItems, dockerErr := a.docker.ImageRemove(ctx, image, *dockerOptions)

	var adapterResponses []ImageDeleteResponseItem
	for _, dockerResponse := range dockerImageDeleteResponseItems {
		adapterResponse := &ImageDeleteResponseItem{
			Deleted:  dockerResponse.Deleted,
			Untagged: dockerResponse.Untagged,
		}
		adapterResponses = append(adapterResponses, *adapterResponse)
	}
	return adapterResponses, dockerErr
}

func (a *DockerAdapter) GetContainerConfig() (ContainerConfig, error) {
	return a.containerConfig, nil
}

func (a *DockerAdapter) GetHostConfig() (HostConfig, error) {
	return a.hostConfig, nil
}

func adapter(docker ThinDockerClient) ThinContainerExecRunner {
	return_object := &DockerAdapter{docker: docker}

	return return_object
}
