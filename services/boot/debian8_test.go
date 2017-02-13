package main

import (
	"log"
	"net"
	"os"
	"testing"
)

func TestDebian8Install(t *testing.T) {
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
	ln.Close()
	log.Printf("Publishing consul webgui at %v", ln.Addr())
	for _, cmdline := range [][]string{
		{"go", "build"},
		{"docker", "build", "--tag=arvados-boot-test-runit", "testimage_runit"},
		{"docker", "run", "--rm", "--publish=" + port + ":18500", "--cap-add=IPC_LOCK", "--cap-add=SYS_ADMIN", "--volume=/sys/fs/cgroup", "--volume=" + cwd + "/boot:/usr/bin/arvados-boot:ro", "arvados-boot-test-runit"},
	} {
		err = command(cmdline[0], cmdline[1:]...).Run()
		if err != nil {
			t.Fatal(err)
		}
	}
}
