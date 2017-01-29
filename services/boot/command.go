package main

import (
	"os"
	"os/exec"
)

func command(prog string, args ...string) *exec.Cmd {
	cmd := exec.Command(prog, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	return cmd
}
