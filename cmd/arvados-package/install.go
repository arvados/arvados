// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// sourcesfile=/tmp/sources.conf.d-arvados
// echo >$sourcesfile "deb [trusted=yes] file:///pkg ./"

// installimage="arvados-installpackage-${osbase}"
// if [[ "${opts[force-installimage]}" || -z "$(docker images --format {{.Repository}} "${installimage}")" ]]; then
//     echo >&2 building ${installimage}...
//     installctr=${installimage/:/-}
//     docker rm "${installctr}" || true
//     docker run -it \
//            --name "${installctr}" \
//            --tmpfs /tmp \
//            -v /tmp/pkg:/pkg:ro \
//            -v ${sourcesfile}:/etc/apt/sources.list.d/arvados-local.list:ro \
//            --env DEBIAN_FRONTEND=noninteractive \
//            "${osbase}" \
//            bash -c 'apt update && apt install -y eatmydata && eatmydata apt install -y arvados-server-easy postgresql && eatmydata apt remove -y arvados-server-easy'
//     docker commit "${installctr}" "${installimage}"
//     docker rm "${installctr}"
//     installctr=
// fi

func testinstall(ctx context.Context, opts opts, stdin io.Reader, stdout, stderr io.Writer) error {
	if opts.PackageVersion != "" {
		return errors.New("not implemented: package version was specified, but I only know how to test the latest version in pkgdir")
	}
	depsImageName := "arvados-package-deps-" + opts.TargetOS
	depsCtrName := strings.Replace(depsImageName, ":", "-", -1)

	_, prog := filepath.Split(os.Args[0])
	tmpdir, err := ioutil.TempDir("", prog+".")
	if err != nil {
		return fmt.Errorf("TempDir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

	sourcesFile := tmpdir + "/arvados-local.list"
	err = ioutil.WriteFile(sourcesFile, []byte("deb [trusted=yes] file:///pkg ./\n"), 0644)
	if err != nil {
		return fmt.Errorf("Write %s: %w", sourcesFile, err)
	}

	if exists, err := dockerImageExists(ctx, depsImageName); err != nil {
		return err
	} else if !exists || opts.RebuildImage {
		err = dockerRm(ctx, depsCtrName)
		if err != nil {
			return err
		}
		defer dockerRm(ctx, depsCtrName)
		cmd := exec.CommandContext(ctx, "docker", "run",
			"--name", depsCtrName,
			"--tmpfs", "/tmp:exec,mode=01777",
			"-v", opts.PackageDir+":/pkg:ro",
			"-v", sourcesFile+":/etc/apt/sources.list.d/arvados-local.list:ro",
			"--env", "DEBIAN_FRONTEND=noninteractive",
			opts.TargetOS,
			"bash", "-c", `
set -e
apt-get update
apt-get install -y eatmydata
eatmydata apt-get install -y --no-install-recommends arvados-server-easy postgresql
eatmydata apt-get remove -y arvados-server-easy
`)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("docker run: %w", err)
		}

		cmd = exec.CommandContext(ctx, "docker", "commit", depsCtrName, depsImageName)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("docker commit: %w", err)
		}
	}

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"--tmpfs", "/tmp:exec,mode=01777",
		"-v", opts.PackageDir+":/pkg:ro",
		"-v", sourcesFile+":/etc/apt/sources.list.d/arvados-local.list:ro",
		"--env", "DEBIAN_FRONTEND=noninteractive",
		depsImageName,
		"bash", "-c", `
set -e
PATH="/var/lib/arvados/bin:$PATH"
apt-get update
eatmydata apt-get install --reinstall -y --no-install-recommends arvados-server-easy
apt-get -y autoremove
/etc/init.d/postgresql start
arvados-server init -cluster-id x1234
exec arvados-server boot -listen-host 0.0.0.0 -shutdown
`)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("docker run: %w", err)
	}
	return nil
}

func dockerImageExists(ctx context.Context, name string) (bool, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return false, err
	}
	imgs, err := cli.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		return false, err
	}
	for _, img := range imgs {
		for _, tag := range img.RepoTags {
			if tag == name {
				return true, nil
			}
		}
	}
	return false, nil
}
