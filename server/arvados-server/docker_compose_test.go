package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestDockerCompose(t *testing.T) {
	for _, cmdline := range [][]string{
		{"go", "build"},
		{"docker-compose", "--file", "test-docker-compose/docker-compose.yml", "-p", "arvados_setup_test", "down", "-v"},
		{"docker-compose", "--file", "test-docker-compose/docker-compose.yml", "-p", "arvados_setup_test", "up"},
		{"docker", "wait", "arvadossetuptest_sys0_1"},
		{"docker", "wait", "arvadossetuptest_sys1_1"},
		{"docker", "wait", "arvadossetuptest_sys2_1"},
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
