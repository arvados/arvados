package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestSetupDockerCompose(t *testing.T) {
	for _, cmdline := range [][]string{
		{"go", "build"},
		{"docker-compose", "--file", "test-docker-compose/docker-compose.yml", "down"},
		{"docker-compose", "--file", "test-docker-compose/docker-compose.yml", "up"},
	} {
		cmd := exec.Command(cmdline[0], cmdline[1:]...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
	}
}
