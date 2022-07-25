// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"git.arvados.org/arvados.git/lib/install"
)

func fpm(ctx context.Context, opts opts, stdin io.Reader, stdout, stderr io.Writer) error {
	var chownUid, chownGid int
	if opts.PackageChown != "" {
		_, err := fmt.Sscanf(opts.PackageChown, "%d:%d", &chownUid, &chownGid)
		if err != nil {
			return fmt.Errorf("invalid value %q for PackageChown: %w", opts.PackageChown, err)
		}
	}

	exitcode := install.Command.RunCommand("arvados-server install", []string{
		"-type", "package",
		"-package-version", opts.PackageVersion,
		"-source", opts.SourceDir,
	}, stdin, stdout, stderr)
	if exitcode != 0 {
		return fmt.Errorf("arvados-server install failed: exit code %d", exitcode)
	}

	cmd := exec.Command("/var/lib/arvados/bin/gem", "env", "gempath")
	cmd.Stderr = stderr
	buf, err := cmd.Output() // /root/.gem/ruby/2.7.0:...
	if err != nil || len(buf) == 0 {
		return fmt.Errorf("gem env gempath: %w", err)
	}
	gempath := string(bytes.TrimRight(bytes.Split(buf, []byte{':'})[0], "\n"))

	cmd = exec.Command("/var/lib/arvados/bin/gem", "install", "--user", "--no-document", "fpm")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// Avoid "WARNING: You don't have [...] in your PATH, gem
	// executables will not run"
	cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH")+":"+gempath)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("gem install fpm: %w", err)
	}

	if _, err := os.Stat(gempath + "/gems/fpm-1.11.0/lib/fpm/package/deb.rb"); err == nil {
		// Workaround for fpm bug https://github.com/jordansissel/fpm/issues/1739
		cmd = exec.Command("sed", "-i", `/require "digest"/a require "zlib"`, gempath+"/gems/fpm-1.11.0/lib/fpm/package/deb.rb")
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("monkeypatch fpm: %w", err)
		}
	}

	// Remove unneeded files. This is much faster than "fpm
	// --exclude X" because fpm copies everything into a staging
	// area before looking at the --exclude args.
	cmd = exec.Command("bash", "-c", "cd /var/www/.gem/ruby && rm -rf */cache */bundler/gems/*/.git */bundler/gems/arvados-*/[^s]* */bundler/gems/arvados-*/s[^d]* */bundler/gems/arvados-*/sdk/[^cr]* */gems/passenger-*/src/cxx* ruby/*/gems/*/ext /var/lib/arvados/go /var/lib/arvados/arvados-workbench2 /var/lib/arvados/node-*")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %w", cmd.Args, err)
	}

	format := "deb" // TODO: rpm
	pkgfile := filepath.Join(opts.PackageDir, "arvados-server-easy_"+opts.PackageVersion+"_amd64."+format)

	cmd = exec.Command(gempath+"/bin/fpm",
		"--package", pkgfile,
		"--name", "arvados-server-easy",
		"--version", opts.PackageVersion,
		"--url", "https://arvados.org",
		"--maintainer", opts.Maintainer,
		"--vendor", opts.Vendor,
		"--license", "GNU Affero General Public License, version 3.0",
		"--description", "platform for managing, processing, and sharing genomic and other large scientific and biomedical data",
		"--input-type", "dir",
		"--output-type", format)
	deps, err := install.ProductionDependencies()
	if err != nil {
		return err
	}
	for _, pkg := range deps {
		cmd.Args = append(cmd.Args, "--depends", pkg)
	}
	cmd.Args = append(cmd.Args,
		"--verbose",
		"--deb-use-file-permissions",
		"--rpm-use-file-permissions",
		"--deb-systemd", "/lib/systemd/system/arvados.service",
		"--deb-systemd-enable",
		"--no-deb-systemd-auto-start",
		"--no-deb-systemd-restart-after-upgrade",
		"--deb-suggests", "postgresql",
		"--deb-suggests", "docker.io",
		"/usr/bin/arvados-client",
		"/usr/bin/arvados-server",
		"/usr/bin/arv",
		"/usr/bin/arv-ruby",
		"/usr/bin/arv-tag",
		"/var/lib/arvados",
		"/usr/bin/arv-copy",
		"/usr/bin/arv-federation-migrate",
		"/usr/bin/arv-get",
		"/usr/bin/arv-keepdocker",
		"/usr/bin/arv-ls",
		"/usr/bin/arv-migrate-docker19",
		"/usr/bin/arv-normalize",
		"/usr/bin/arv-put",
		"/usr/bin/arv-ws",
		"/usr/bin/arv-mount",
		"/var/www/.gem",
		"/var/www/.passenger",
		"/var/www/.bundle",
	)
	fmt.Fprintf(stderr, "... %s\n", cmd.Args)
	cmd.Dir = opts.PackageDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("fpm: %w", err)
	}

	if opts.PackageChown != "" {
		err = os.Chown(pkgfile, chownUid, chownGid)
		if err != nil {
			return fmt.Errorf("chown %s: %w", pkgfile, err)
		}
	}

	cmd = exec.Command("ls", "-l", pkgfile)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	_ = cmd.Run()

	return nil
}
