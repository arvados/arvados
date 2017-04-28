package main

import (
	"log"
	"os"
	"os/exec"
	"testing"
)

func TestDockerCompose(t *testing.T) {
	run := func(cmdline []string) error {
		log.Printf("TestDockerCompose: %q", cmdline)
		cmd := exec.Command(cmdline[0], cmdline[1:]...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	for _, cmdline := range [][]string{
		{"go", "build"},
		{"docker-compose", "--file", "test-docker-compose/docker-compose.yml", "-p", "arvados_setup_test", "down", "-v"},
		{"docker-compose", "--file", "test-docker-compose/docker-compose.yml", "-p", "arvados_setup_test", "up", "-d"},
		{"docker-compose", "--file", "test-docker-compose/docker-compose.yml", "-p", "arvados_setup_test", "logs", "--timestamps", "--follow"},
		{"docker", "wait", "arvadossetuptest_sys0_1"},
	} {
		if cmdline[len(cmdline)-1] == "--follow" {
			go run(cmdline)
			continue
		}
		err := run(cmdline)
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for _, cmdline := range [][]string{
			{"docker", "stop", "arvadossetuptest_sys0_1"},
			{"docker", "stop", "arvadossetuptest_sys1_1"},
			{"docker", "stop", "arvadossetuptest_sys2_1"},
		} {
			run(cmdline)
		}
	}()
}
