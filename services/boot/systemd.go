package main

import (
	"fmt"
	"os"
	"os/exec"
)

type supervisor interface {
	Check() (bool, error)
	Start() error
}

func newSupervisor(name, cmd string, args ...string) supervisor {
	return &systemdUnit{
		name: name,
		cmd: cmd,
		args: args,
	}
}

type systemdUnit struct {
	name string
	cmd string
	args []string
}

func (u *systemdUnit) Start() error {
	cmd := exec.Command("systemd-run", append([]string{"--unit=arvados-"+u.name, u.cmd}, u.args...)...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("systemd-run: %s", err)
	}
	return err
}

func (u *systemdUnit) Check() (bool, error) {
	cmd := exec.Command("systemctl", "status", "arvados-"+u.name)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	switch err.(type) {
	case *exec.ExitError:
		return false, nil
	case nil:
		return true, nil
	default:
		return false, err
	}
}
