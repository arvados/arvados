package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"testing"
)

func TestSetupDebian8(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", ":")
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	err = ln.Close()
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("Publishing consul webgui at %v", ln.Addr())
	for _, cmdline := range [][]string{
		{"go", "build"},
		{"docker", "build", "--tag=arvados-admin-debian8-test", "test-debian8"},
		{"docker", "run", "--rm", "--publish=" + port + ":18500", "--cap-add=IPC_LOCK", "--cap-add=SYS_ADMIN", "--volume=/sys/fs/cgroup", "--volume=" + cwd + "/arvados-admin:/usr/bin/arvados-admin:ro", "--volume=/var/cache/arvados:/var/cache/arvados:ro", "arvados-admin-debian8-test"},
	} {
		cmd := exec.Command(cmdline[0], cmdline[1:]...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			t.Fatal(err)
		}
	}
}
