// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
package crunchrun

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// Docker daemon won't let you set a limit less than ~10 MiB
const minDockerRAM = int64(16 * 1024 * 1024)

type dockerExecutor struct {
	containerUUID    string
	logf             func(string, ...interface{})
	watchdogInterval time.Duration
	dockerclient     *dockerclient.Client
	containerID      string
	doneIO           chan struct{}
	errIO            error
}

func newDockerExecutor(containerUUID string, logf func(string, ...interface{}), watchdogInterval time.Duration) (*dockerExecutor, error) {
	// API version 1.21 corresponds to Docker 1.9, which is
	// currently the minimum version we want to support.
	client, err := dockerclient.NewClient(dockerclient.DefaultDockerHost, "1.21", nil, nil)
	if watchdogInterval < 1 {
		watchdogInterval = time.Minute
	}
	return &dockerExecutor{
		containerUUID:    containerUUID,
		logf:             logf,
		watchdogInterval: watchdogInterval,
		dockerclient:     client,
	}, err
}

func (e *dockerExecutor) Runtime() string { return "docker" }

func (e *dockerExecutor) LoadImage(imageID string, imageTarballPath string, container arvados.Container, arvMountPoint string,
	containerClient *arvados.Client) error {
	_, _, err := e.dockerclient.ImageInspectWithRaw(context.TODO(), imageID)
	if err == nil {
		// already loaded
		return nil
	}

	f, err := os.Open(imageTarballPath)
	if err != nil {
		return err
	}
	defer f.Close()
	resp, err := e.dockerclient.ImageLoad(context.TODO(), f, true)
	if err != nil {
		return fmt.Errorf("While loading container image into Docker: %v", err)
	}
	defer resp.Body.Close()
	buf, _ := ioutil.ReadAll(resp.Body)
	e.logf("loaded image: response %s", buf)
	return nil
}

func (e *dockerExecutor) Create(spec containerSpec) error {
	e.logf("Creating Docker container")
	cfg := dockercontainer.Config{
		Image:        spec.Image,
		Cmd:          spec.Command,
		WorkingDir:   spec.WorkingDir,
		Volumes:      map[string]struct{}{},
		OpenStdin:    spec.Stdin != nil,
		StdinOnce:    spec.Stdin != nil,
		AttachStdin:  spec.Stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
	}
	if cfg.WorkingDir == "." {
		cfg.WorkingDir = ""
	}
	for k, v := range spec.Env {
		cfg.Env = append(cfg.Env, k+"="+v)
	}
	if spec.RAM > 0 && spec.RAM < minDockerRAM {
		spec.RAM = minDockerRAM
	}
	hostCfg := dockercontainer.HostConfig{
		LogConfig: dockercontainer.LogConfig{
			Type: "none",
		},
		NetworkMode: dockercontainer.NetworkMode("none"),
		Resources: dockercontainer.Resources{
			CgroupParent: spec.CgroupParent,
			NanoCPUs:     int64(spec.VCPUs) * 1000000000,
			Memory:       spec.RAM, // RAM
			MemorySwap:   spec.RAM, // RAM+swap
			KernelMemory: spec.RAM, // kernel portion
		},
	}
	for path, mount := range spec.BindMounts {
		bind := mount.HostPath + ":" + path
		if mount.ReadOnly {
			bind += ":ro"
		}
		hostCfg.Binds = append(hostCfg.Binds, bind)
	}
	if spec.EnableNetwork {
		hostCfg.NetworkMode = dockercontainer.NetworkMode(spec.NetworkMode)
	}

	created, err := e.dockerclient.ContainerCreate(context.TODO(), &cfg, &hostCfg, nil, e.containerUUID)
	if err != nil {
		return fmt.Errorf("While creating container: %v", err)
	}
	e.containerID = created.ID
	return e.startIO(spec.Stdin, spec.Stdout, spec.Stderr)
}

func (e *dockerExecutor) CgroupID() string {
	return e.containerID
}

func (e *dockerExecutor) Start() error {
	return e.dockerclient.ContainerStart(context.TODO(), e.containerID, dockertypes.ContainerStartOptions{})
}

func (e *dockerExecutor) Stop() error {
	err := e.dockerclient.ContainerRemove(context.TODO(), e.containerID, dockertypes.ContainerRemoveOptions{Force: true})
	if err != nil && strings.Contains(err.Error(), "No such container: "+e.containerID) {
		err = nil
	}
	return err
}

// Wait for the container to terminate, capture the exit code, and
// wait for stdout/stderr logging to finish.
func (e *dockerExecutor) Wait(ctx context.Context) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	watchdogErr := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(e.watchdogInterval)
		defer ticker.Stop()
		for range ticker.C {
			dctx, dcancel := context.WithDeadline(ctx, time.Now().Add(e.watchdogInterval))
			ctr, err := e.dockerclient.ContainerInspect(dctx, e.containerID)
			dcancel()
			if ctx.Err() != nil {
				// Either the container already
				// exited, or our caller is trying to
				// kill it.
				return
			} else if err != nil {
				e.logf("Error inspecting container: %s", err)
				watchdogErr <- err
				return
			} else if ctr.State == nil || !(ctr.State.Running || ctr.State.Status == "created") {
				watchdogErr <- fmt.Errorf("Container is not running: State=%v", ctr.State)
				return
			}
		}
	}()

	waitOk, waitErr := e.dockerclient.ContainerWait(ctx, e.containerID, dockercontainer.WaitConditionNotRunning)
	for {
		select {
		case waitBody := <-waitOk:
			e.logf("Container exited with code: %v", waitBody.StatusCode)
			// wait for stdout/stderr to complete
			<-e.doneIO
			return int(waitBody.StatusCode), nil

		case err := <-waitErr:
			return -1, fmt.Errorf("container wait: %v", err)

		case <-ctx.Done():
			return -1, ctx.Err()

		case err := <-watchdogErr:
			return -1, err
		}
	}
}

func (e *dockerExecutor) startIO(stdin io.Reader, stdout, stderr io.Writer) error {
	resp, err := e.dockerclient.ContainerAttach(context.TODO(), e.containerID, dockertypes.ContainerAttachOptions{
		Stream: true,
		Stdin:  stdin != nil,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("error attaching container stdin/stdout/stderr streams: %v", err)
	}
	var errStdin error
	if stdin != nil {
		go func() {
			errStdin = e.handleStdin(stdin, resp.Conn, resp.CloseWrite)
		}()
	}
	e.doneIO = make(chan struct{})
	go func() {
		e.errIO = e.handleStdoutStderr(stdout, stderr, resp.Reader)
		if e.errIO == nil && errStdin != nil {
			e.errIO = errStdin
		}
		close(e.doneIO)
	}()
	return nil
}

func (e *dockerExecutor) handleStdin(stdin io.Reader, conn io.Writer, closeConn func() error) error {
	defer closeConn()
	_, err := io.Copy(conn, stdin)
	if err != nil {
		return fmt.Errorf("While writing to docker container on stdin: %v", err)
	}
	return nil
}

// Handle docker log protocol; see
// https://docs.docker.com/engine/reference/api/docker_remote_api_v1.15/#attach-to-a-container
func (e *dockerExecutor) handleStdoutStderr(stdout, stderr io.Writer, reader io.Reader) error {
	header := make([]byte, 8)
	var err error
	for err == nil {
		_, err = io.ReadAtLeast(reader, header, 8)
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		readsize := int64(header[7]) | (int64(header[6]) << 8) | (int64(header[5]) << 16) | (int64(header[4]) << 24)
		if header[0] == 1 {
			_, err = io.CopyN(stdout, reader, readsize)
		} else {
			// stderr
			_, err = io.CopyN(stderr, reader, readsize)
		}
	}
	if err != nil {
		return fmt.Errorf("error copying stdout/stderr from docker: %v", err)
	}
	return nil
}

func (e *dockerExecutor) Close() {
	e.dockerclient.ContainerRemove(context.TODO(), e.containerID, dockertypes.ContainerRemoveOptions{Force: true})
}
