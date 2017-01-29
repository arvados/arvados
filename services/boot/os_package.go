package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	installCerts = &osPackage{
		Debian: "ca-certificates",
	}
	installNginx = &osPackage{
		Debian: "nginx",
	}
	installRunit = &osPackage{
		Debian: "runit",
	}
)

type osPackage struct {
	Debian string
	RedHat string
}

var (
	osPackageMutex     sync.Mutex
	osPackageDidUpdate bool
)

func (pkg *osPackage) Boot(ctx context.Context) error {
	osPackageMutex.Lock()
	defer osPackageMutex.Unlock()

	if _, err := os.Stat("/var/lib/dpkg/info/" + pkg.Debian + ".list"); err == nil {
		return nil
	}
	if !osPackageDidUpdate {
		d, err := os.Open("/var/lib/apt/lists")
		if err != nil {
			return err
		}
		defer d.Close()
		if files, err := d.Readdir(4); len(files) < 4 || err != nil {
			err = pkg.aptGet("update")
			if err != nil {
				return err
			}
			osPackageDidUpdate = true
		}
	}
	return pkg.aptGet("install", "-y", "--no-install-recommends", pkg.Debian)
}

func (*osPackage) aptGet(args ...string) error {
	cmd := command("apt-get", args...)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "DEBIAN_FRONTEND=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, "DEBIAN_FRONTEND=noninteractive")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s: %s", cmd.Args, err)
	}
	return nil
}
