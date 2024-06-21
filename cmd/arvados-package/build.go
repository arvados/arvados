// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/lib/crunchrun"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func build(ctx context.Context, opts opts, stdin io.Reader, stdout, stderr io.Writer) error {
	if opts.PackageVersion == "" {
		var buf bytes.Buffer
		cmd := exec.CommandContext(ctx, "bash", "./build/version-at-commit.sh", "HEAD")
		cmd.Stdout = &buf
		cmd.Stderr = stderr
		cmd.Dir = opts.SourceDir
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("%v: %w", cmd.Args, err)
		}
		opts.PackageVersion = strings.TrimSpace(buf.String())
		ctxlog.FromContext(ctx).Infof("version not specified; using %s", opts.PackageVersion)
	}

	if opts.PackageChown == "" {
		whoami, err := user.Current()
		if err != nil {
			return fmt.Errorf("user.Current: %w", err)
		}
		opts.PackageChown = whoami.Uid + ":" + whoami.Gid
	}

	// Build in a tempdir, then move to the desired destination
	// dir. Otherwise, errors might cause us to leave a mess:
	// truncated files, files owned by root, etc.
	_, prog := filepath.Split(os.Args[0])
	tmpdir, err := ioutil.TempDir(opts.PackageDir, prog+".")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	if abs, err := filepath.Abs(tmpdir); err != nil {
		return fmt.Errorf("error getting absolute path of tmpdir %s: %w", tmpdir, err)
	} else {
		tmpdir = abs
	}

	selfbin, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return fmt.Errorf("readlink /proc/self/exe: %w", err)
	}
	buildImageName := "arvados-package-build-" + opts.TargetOS
	packageFilename := "arvados-server-easy_" + opts.PackageVersion + "_amd64.deb"

	if ok, err := dockerImageExists(ctx, buildImageName); err != nil {
		return err
	} else if !ok || opts.RebuildImage {
		buildCtrName := strings.Replace(buildImageName, ":", "-", -1)
		err = dockerRm(ctx, buildCtrName)
		if err != nil {
			return err
		}

		defer dockerRm(ctx, buildCtrName)
		cmd := exec.CommandContext(ctx, "docker", "run",
			"--name", buildCtrName,
			"--tmpfs", "/tmp:exec,mode=01777",
			"-v", selfbin+":/arvados-package:ro",
			"-v", opts.SourceDir+":/arvados:ro",
			opts.TargetOS,
			"/arvados-package", "_install",
			"-eatmydata",
			"-type", "package",
			"-source", "/arvados",
			"-package-version", opts.PackageVersion,
		)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("%v: %w", cmd.Args, err)
		}

		cmd = exec.CommandContext(ctx, "docker", "commit", buildCtrName, buildImageName)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("docker commit: %w", err)
		}

		ctxlog.FromContext(ctx).Infof("created docker image %s", buildImageName)
	}

	cmd := exec.CommandContext(ctx, "docker", "run",
		"--rm",
		"--tmpfs", "/tmp:exec,mode=01777",
		"-v", tmpdir+":/pkg",
		"-v", selfbin+":/arvados-package:ro",
		"-v", opts.SourceDir+":/arvados:ro",
		buildImageName,
		"eatmydata", "/arvados-package", "_fpm",
		"-source", "/arvados",
		"-package-version", opts.PackageVersion,
		"-package-dir", "/pkg",
		"-package-chown", opts.PackageChown,
		"-package-maintainer", opts.Maintainer,
		"-package-vendor", opts.Vendor,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %w", cmd.Args, err)
	}

	err = os.Rename(tmpdir+"/"+packageFilename, opts.PackageDir+"/"+packageFilename)
	if err != nil {
		return err
	}
	cmd = exec.CommandContext(ctx, "ls", "-l", opts.PackageDir+"/"+packageFilename)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func dockerRm(ctx context.Context, name string) error {
	cli, err := client.NewClient(client.DefaultDockerHost, crunchrun.DockerAPIVersion, nil, nil)
	if err != nil {
		return err
	}
	ctrs, err := cli.ContainerList(ctx, container.ListOptions{All: true, Limit: -1})
	if err != nil {
		return err
	}
	for _, ctr := range ctrs {
		for _, ctrname := range ctr.Names {
			if ctrname == "/"+name {
				err = cli.ContainerRemove(ctx, ctr.ID, container.RemoveOptions{})
				if err != nil {
					return fmt.Errorf("error removing container %s: %w", ctr.ID, err)
				}
				break
			}
		}
	}
	return nil
}
