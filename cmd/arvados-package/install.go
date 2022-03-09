// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
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

func testinstall(ctx context.Context, opts opts, stdin io.Reader, stdout, stderr io.Writer) error {
	depsImageName := "arvados-package-deps-" + opts.TargetOS
	depsCtrName := strings.Replace(depsImageName, ":", "-", -1)
	absPackageDir, err := filepath.Abs(opts.PackageDir)
	if err != nil {
		return fmt.Errorf("error resolving PackageDir %q: %w", opts.PackageDir, err)
	}

	_, prog := filepath.Split(os.Args[0])
	tmpdir, err := ioutil.TempDir("", prog+".")
	if err != nil {
		return fmt.Errorf("TempDir: %w", err)
	}
	defer os.RemoveAll(tmpdir)

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
			"-v", absPackageDir+":/pkg:ro",
			"--env", "DEBIAN_FRONTEND=noninteractive",
			opts.TargetOS,
			"bash", "-c", `
set -e -o pipefail
apt-get --allow-releaseinfo-change update
apt-get install -y --no-install-recommends dpkg-dev eatmydata

mkdir /tmp/pkg
ln -s /pkg/*.deb /tmp/pkg/
(cd /tmp/pkg; dpkg-scanpackages --multiversion . | gzip > Packages.gz)
echo >/etc/apt/sources.list.d/arvados-local.list "deb [trusted=yes] file:///tmp/pkg ./"
apt-get --allow-releaseinfo-change update

eatmydata apt-get install -y --no-install-recommends arvados-server-easy postgresql
eatmydata apt-get remove -y dpkg-dev
SUDO_FORCE_REMOVE=yes apt-get autoremove -y
eatmydata apt-get remove -y arvados-server-easy
rm /etc/apt/sources.list.d/arvados-local.list
`)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("%v: %w", cmd.Args, err)
		}

		cmd = exec.CommandContext(ctx, "docker", "commit", depsCtrName, depsImageName)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("%v: %w", cmd.Args, err)
		}
	}

	versionsuffix := ""
	if opts.PackageVersion != "" {
		versionsuffix = "=" + opts.PackageVersion
	}
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"--tmpfs", "/tmp:exec,mode=01777",
		"-v", absPackageDir+":/pkg:ro",
		"--env", "DEBIAN_FRONTEND=noninteractive",
		depsImageName,
		"bash", "-c", `
set -e -o pipefail
PATH="/var/lib/arvados/bin:$PATH"
apt-get --allow-releaseinfo-change update
apt-get install -y --no-install-recommends dpkg-dev
mkdir /tmp/pkg
ln -s /pkg/*.deb /tmp/pkg/
(cd /tmp/pkg; dpkg-scanpackages --multiversion . | gzip > Packages.gz)
apt-get remove -y dpkg-dev
echo

echo >/etc/apt/sources.list.d/arvados-local.list "deb [trusted=yes] file:///tmp/pkg ./"
apt-get --allow-releaseinfo-change update
eatmydata apt-get install --reinstall -y --no-install-recommends arvados-server-easy`+versionsuffix+`
SUDO_FORCE_REMOVE=yes apt-get autoremove -y

/etc/init.d/postgresql start
arvados-server init -cluster-id x1234
exec arvados-server boot -listen-host 0.0.0.0 -shutdown
`)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %w", cmd.Args, err)
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
