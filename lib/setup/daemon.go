package setup

import (
	"log"
	"math/rand"
	"os"

	"github.com/hashicorp/consul/api"
)

type daemon struct {
	name       string
	prog       string // program to run (absolute path) -- if blank, use name
	args       []string
	noRegister bool
}

func (s *Setup) installService(d daemon) error {
	if d.prog == "" {
		d.prog = d.name
	}
	if _, err := os.Stat(d.prog); err != nil {
		return err
	}
	sup := s.superviseDaemon(d)
	if ok, err := sup.Running(); err != nil {
		return err
	} else if !ok {
		if err := sup.Start(); err != nil {
			return err
		}
	}
	if d.noRegister {
		return nil
	}
	consul, err := s.ConsulMaster()
	if err != nil {
		return err
	}
	agent := consul.Agent()
	svcs, err := agent.Services()
	if err != nil {
		return err
	}
	if svc, ok := svcs[d.name]; ok {
		log.Printf("%q is registered: %#v", d.name, svc)
		return nil
	}
	return agent.ServiceRegister(&api.AgentServiceRegistration{
		ID:   d.name,
		Name: d.name,
		Port: availablePort(),
	})
}

type supervisor interface {
	Running() (bool, error)
	Start() error
}

func (s *Setup) superviseDaemon(d daemon) supervisor {
	switch s.DaemonSupervisor {
	case "runit":
		return &runitService{daemon: d, etcsv: s.RunitSvDir}
	case "systemd":
		return &systemdSupervisor{daemon: d}
	default:
		log.Fatalf("unknown DaemonSupervisor %q", s.DaemonSupervisor)
		return nil
	}
}

func availablePort() int {
	return rand.Intn(10000) + 20000
}
