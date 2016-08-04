package main

import (
	"bytes"
	"io/ioutil"
	"log"
)

// Return the current process's cgroup for the given subsystem.
func findCgroup(subsystem string) string {
	subsys := []byte(subsystem)
	cgroups, err := ioutil.ReadFile("/proc/self/cgroup")
	if err != nil {
		log.Fatal(err)
	}
	for _, line := range bytes.Split(cgroups, []byte("\n")) {
		toks := bytes.SplitN(line, []byte(":"), 4)
		if len(toks) < 3 {
			continue
		}
		for _, s := range bytes.Split(toks[1], []byte(",")) {
			if bytes.Compare(s, subsys) == 0 {
				return string(toks[2])
			}
		}
	}
	log.Fatalf("subsystem %q not found in /proc/self/cgroup", subsystem)
	return ""
}
