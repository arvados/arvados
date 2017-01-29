package main

import (
	"context"
	"fmt"
	"os/exec"
)

type systemdUnit struct {
	name string
	cmd  string
	args []string
}

func (u *systemdUnit) Start(ctx context.Context) error {
	cmd := command("systemd-run", append([]string{"--unit=arvados-" + u.name, u.cmd}, u.args...)...)
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("systemd-run: %s", err)
	}
	return err
}

func (u *systemdUnit) Running(ctx context.Context) (bool, error) {
	return runStatusCmd("systemctl", "status", "arvados-"+u.name)
}

func runStatusCmd(prog string, args ...string) (bool, error) {
	cmd := command(prog, args...)
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
