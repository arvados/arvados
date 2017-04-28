package setup

import "fmt"

type systemdSupervisor struct {
	daemon
}

func (ss *systemdSupervisor) Start() error {
	cmd := command("systemd-run", append([]string{"--unit=arvados-" + ss.name, ss.prog}, ss.args...)...)
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("systemd-run: %s", err)
	}
	return err
}

func (ss *systemdSupervisor) Running() (bool, error) {
	return runStatusCmd("systemctl", "status", "arvados-"+ss.name)
}
