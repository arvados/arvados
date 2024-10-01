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
	err := exec.Command("systemd-run", "--wait", "--user", "true").Run()
	if err != nil {
		logf("`systemd-run --wait --user true` failed (%s) -- singularity resource limits are not supported", err)
		return
	}
	version, err := exec.Command("systemd-run", "--version").CombinedOutput()
	if match := regexp.MustCompile(`^systemd (\d+)`).FindSubmatch(version); err != nil || match == nil {
		logf("could not get systemd version -- singularity resource limits are not supported")
		return
	} else if v, _ := strconv.ParseInt(string(match[1]), 10, 64); v < 224 {
		logf("systemd version %s < minimum 224 -- singularity resource limits are not supported", match[1])
		return
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
