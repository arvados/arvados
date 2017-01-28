package main

import (
	"bytes"
	"os/exec"
	"strings"
)

func BashScript(script string) ([]byte, []byte, error) {
	cmd := exec.Command("bash", "-e", "-x")
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
