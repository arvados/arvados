package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/hashicorp/consul/api"
)

type consulService struct {
	supervisor
}

func (cs *consulService) Init(cfg *Config) {
	args := []string{
		"agent",
		"-server",
		"-advertise=127.0.0.1",
		"-data-dir", cfg.DataDir + "/consul",
		"-bootstrap-expect", fmt.Sprintf("%d", len(cfg.ControlHosts))}
	cs.supervisor = newSupervisor("consul", "/usr/local/bin/consul", args...)
}

func (cs *consulService) Children() []task {
	return nil
}

func (cs *consulService) ShortName() string {
	return "consul running"
}

func (cs *consulService) String() string {
	return "Ensure consul daemon is supervised & running"
}

func (cs *consulService) Check() error {
	consul, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}
	_, err = consul.Catalog().Datacenters()
	if err != nil {
		return err
	}
	return nil
}

func (cs *consulService) CanFix() bool {
	return true
}

func (cs *consulService) Fix() error {
	err := cs.supervisor.Start()
	if err != nil {
		return err
	}

	if len(cfg.ControlHosts) > 1 {
		cmd := exec.Command("/usr/local/bin/consul", append([]string{"join"}, cfg.ControlHosts...)...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return err
		}
	}

	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for cs.Check() != nil {
		select {
		case <-ticker.C:
		case <-timeout:
			return cs.Check()
		}
	}
	return nil
}
