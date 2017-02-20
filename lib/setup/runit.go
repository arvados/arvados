package setup

import (
	"bytes"
	"fmt"
	"os"
	"path"
)

func (s *Setup) installRunit() error {
	if s.DaemonSupervisor != "runit" {
		return nil
	}
	return (&osPackage{Debian: "runit"}).install()
}

type runitService struct {
	daemon
	etcsv string
}

func (r *runitService) Start() error {
	script := &bytes.Buffer{}
	fmt.Fprintf(script, "#!/bin/sh\n\nexec %q", r.prog)
	for _, arg := range r.args {
		fmt.Fprintf(script, " %q", arg)
	}
	fmt.Fprintf(script, " 2>&1\n")
	return atomicWriteFile(path.Join(r.svdir(), "run"), script.Bytes(), 0755)
}

func (r *runitService) Running() (bool, error) {
	if _, err := os.Stat(r.svdir()); err != nil && os.IsNotExist(err) {
		return false, nil
	}
	return runStatusCmd("sv", "stat", r.svdir())
}

func (r *runitService) svdir() string {
	return path.Join(r.etcsv, r.name)
}
