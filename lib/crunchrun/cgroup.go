// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"fmt"
	"io/fs"
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
