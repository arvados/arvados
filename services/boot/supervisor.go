package main

import (
	"context"
	"log"
	"os"

	"github.com/hashicorp/consul/api"
)

type supervisor interface {
	Running(ctx context.Context) (bool, error)
	Start(ctx context.Context) error
}

func newSupervisor(ctx context.Context, name, cmd string, args ...string) supervisor {
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return &systemdUnit{
			name: name,
			cmd:  cmd,
			args: args,
		}
	}
	return &runitService{
		name: name,
		cmd:  cmd,
		args: args,
	}
}

type supervisedService struct {
	name string
	cmd  string
	args []string
}

func (s *supervisedService) Boot(ctx context.Context) error {
	bin := s.cmd
	if bin == "" {
		bin = s.name
	}
	if _, err := os.Stat(bin); err != nil {
		return err
	}
	sup := newSupervisor(ctx, s.name, bin)
	if ok, err := sup.Running(ctx); err != nil {
		return err
	} else if !ok {
		if err := sup.Start(ctx); err != nil {
			return err
		}
	}
	if err := consul.Boot(ctx); err != nil {
		return err
	}
	consul, err := api.NewClient(consulCfg)
	if err != nil {
		return err
	}
	agent := consul.Agent()
	svcs, err := agent.Services()
	if err != nil {
		return err
	}
	if svc, ok := svcs[s.name]; ok {
		log.Printf("%s is registered: %#v", s.name, svc)
		return nil
	}
	return agent.ServiceRegister(&api.AgentServiceRegistration{
		ID:   s.name,
		Name: s.name,
		Port: availablePort(),
	})
}
