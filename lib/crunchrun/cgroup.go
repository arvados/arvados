// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
)

// Return the current process's cgroup for the given subsystem.
//
// If the host has cgroups v2 and not v1 (i.e., unified mode), return
// the current process's cgroup.
func findCgroup(fsys fs.FS, subsystem string) (string, error) {
	subsys := []byte(subsystem)
	cgroups, err := fs.ReadFile(fsys, "proc/self/cgroup")
	if err != nil {
		return "", err
	}
	for _, line := range bytes.Split(cgroups, []byte("\n")) {
		toks := bytes.SplitN(line, []byte(":"), 4)
		if len(toks) < 3 {
			continue
		}
		if len(toks[1]) == 0 && string(toks[0]) == "0" {
			// cgroups v2: "0::$PATH"
			//
			// In "hybrid" mode, this entry is last, so we
			// use it when the specified subsystem doesn't
			// match a cgroups v1 entry.
			//
			// In "unified" mode, this is the only entry,
			// so we use it regardless of which subsystem
			// was specified.
			return string(toks[2]), nil
		}
		for _, s := range bytes.Split(toks[1], []byte(",")) {
			// cgroups v1: "7:cpu,cpuacct:/user.slice"
			if bytes.Compare(s, subsys) == 0 {
				return string(toks[2]), nil
			}
		}
	}
	return "", fmt.Errorf("subsystem %q not found in /proc/self/cgroup", subsystem)
}

var (
	// After calling checkCgroupSupport, cgroupSupport indicates
	// support for singularity resource limits.
	//
	// E.g., cgroupSupport["memory"]==true if systemd is installed
	// and configured such that singularity can use the "memory"
	// cgroup controller to set resource limits.
	cgroupSupport     map[string]bool
	cgroupSupportLock sync.Mutex
)

// checkCgroupSupport should be called before looking up strings like
// "memory" and "cpu" in cgroupSupport.
func checkCgroupSupport(logf func(string, ...interface{})) {
	cgroupSupportLock.Lock()
	defer cgroupSupportLock.Unlock()
	if cgroupSupport != nil {
		return
	}
	cgroupSupport = make(map[string]bool)
	if os.Getuid() != 0 {
		xrd := os.Getenv("XDG_RUNTIME_DIR")
		if xrd == "" || os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
			logf("not running as root, and empty XDG_RUNTIME_DIR or DBUS_SESSION_BUS_ADDRESS -- singularity resource limits are not supported")
			return
		}
		if fi, err := os.Stat(xrd + "/systemd"); err != nil || !fi.IsDir() {
			logf("not running as root, and %s/systemd is not a directory -- singularity resource limits are not supported", xrd)
			return
		}
		version, err := exec.Command("systemd-run", "--version").CombinedOutput()
		if match := regexp.MustCompile(`^systemd (\d+)`).FindSubmatch(version); err != nil || match == nil {
			logf("not running as root, and could not get systemd version -- singularity resource limits are not supported")
			return
		} else if v, _ := strconv.ParseInt(string(match[1]), 10, 64); v < 224 {
			logf("not running as root, and systemd version %s < minimum 224 -- singularity resource limits are not supported", match[1])
			return
		}
	}
	mount, err := cgroupMount()
	if err != nil {
		logf("no cgroup support: %s", err)
		return
	}
	cgroup, err := findCgroup(os.DirFS("/"), "")
	if err != nil {
		logf("cannot find cgroup: %s", err)
		return
	}
	controllers, err := os.ReadFile(mount + cgroup + "/cgroup.controllers")
	if err != nil {
		logf("cannot read cgroup.controllers file: %s", err)
		return
	}
	for _, controller := range bytes.Split(bytes.TrimRight(controllers, "\n"), []byte{' '}) {
		cgroupSupport[string(controller)] = true
	}
}

// Return the cgroup2 mount point, typically "/sys/fs/cgroup".
func cgroupMount() (string, error) {
	mounts, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", err
	}
	for _, mount := range bytes.Split(mounts, []byte{'\n'}) {
		toks := bytes.Split(mount, []byte{' '})
		if len(toks) > 2 && bytes.Equal(toks[0], []byte("cgroup2")) {
			return string(toks[1]), nil
		}
	}
	return "", errors.New("cgroup2 mount not found")
}
