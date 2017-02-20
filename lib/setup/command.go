package setup

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
