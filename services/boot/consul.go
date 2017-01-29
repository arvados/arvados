package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/hashicorp/consul/api"
)

var consul = &consulBooter{}

type consulBooter struct {
	sync.Mutex
}

func (cb *consulBooter) Boot(ctx context.Context) error {
	cb.Lock()
	defer cb.Unlock()

	if cb.check(ctx) == nil {
		return nil
	}
	cfg := cfg(ctx)
	bin := cfg.UsrDir + "/bin/consul"
	err := (&download{
		URL:  "https://releases.hashicorp.com/consul/0.7.2/consul_0.7.2_linux_amd64.zip",
		Dest: bin,
		Size: 29079005,
		Mode: 0755,
	}).Boot(ctx)
	if err != nil {
		return err
	}
	dataDir := cfg.DataDir + "/consul"
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	args := []string{
		"agent",
		"-server",
		"-advertise=127.0.0.1",
		"-data-dir", dataDir,
		"-bootstrap-expect", fmt.Sprintf("%d", len(cfg.ControlHosts))}
	supervisor := newSupervisor(ctx, "consul", bin, args...)
	running, err := supervisor.Running(ctx)
	if err != nil {
		return err
	}
	if !running {
		defer feedbackf(ctx, "starting consul service")()
		err = supervisor.Start(ctx)
		if err != nil {
			return fmt.Errorf("starting consul: %s", err)
		}
		if len(cfg.ControlHosts) > 1 {
			cmd := exec.Command(bin, append([]string{"join"}, cfg.ControlHosts...)...)
			cmd.Stdout = os.Stderr
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				return fmt.Errorf("consul join: %s", err)
			}
		}
	}
	return cb.check(ctx)
}

var consulCfg = api.DefaultConfig()

func (cb *consulBooter) check(ctx context.Context) error {
	cfg := cfg(ctx)
	consulCfg.Datacenter = cfg.SiteID
	consul, err := api.NewClient(consulCfg)
	if err != nil {
		return err
	}
	_, err = consul.Catalog().Datacenters()
	if err != nil {
		return err
	}
	return nil
}
