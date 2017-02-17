package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"

	"github.com/hashicorp/nomad/api"
)

var nomad = &nomadBooter{}

type nomadBooter struct {
	sync.Mutex
}

func (nb *nomadBooter) Boot(ctx context.Context) error {
	nb.Lock()
	defer nb.Unlock()

	if nb.check(ctx) == nil {
		return nil
	}
	cfg := cfg(ctx)
	bin := cfg.UsrDir + "/bin/nomad"
	err := (&download{
		URL:  "https://releases.hashicorp.com/nomad/0.5.4/nomad_0.5.4_linux_amd64.zip",
		Dest: bin,
		Size: 34150464,
		Mode: 0755,
	}).Boot(ctx)
	if err != nil {
		return err
	}

	masterToken, err := ioutil.ReadFile(cfg.masterTokenFile())
	if err != nil {
		return err
	}

	dataDir := path.Join(cfg.DataDir, "nomad")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}

	cf := path.Join(cfg.DataDir, "nomad.json")
	err = atomicWriteJSON(cf, map[string]interface{}{
		"client": map[string]interface{}{
			"enabled": true,
			"options": map[string]interface{}{
				"driver.raw_exec.enable": true,
			},
		},
		"consul": map[string]interface{}{
			"address": fmt.Sprintf("127.0.0.1:%d", cfg.Ports.ConsulHTTP),
			"token":   string(masterToken),
		},
		"data_dir":   dataDir,
		"datacenter": cfg.SiteID,
		"ports": map[string]int{
			"http": cfg.Ports.NomadHTTP,
			"rpc":  cfg.Ports.NomadRPC,
			"serf": cfg.Ports.NomadSerf,
		},
		"server": map[string]interface{}{
			"enabled":          true,
			"bootstrap_expect": len(cfg.ControlHosts),
		},
	}, 0644)
	if err != nil {
		return err
	}

	supervisor := newSupervisor(ctx, "arvados-nomad", bin, "agent", "-config="+cf)
	running, err := supervisor.Running(ctx)
	if err != nil {
		return err
	}
	if !running {
		defer feedbackf(ctx, "starting nomad service")()
		err = supervisor.Start(ctx)
		if err != nil {
			return fmt.Errorf("starting nomad: %s", err)
		}
	}
	return waitCheck(ctx, 30*time.Second, nb.check)
}

var nomadCfg = api.DefaultConfig()

func (nb *nomadBooter) check(ctx context.Context) error {
	cfg := cfg(ctx)
	nomadCfg.Address = fmt.Sprintf("http://127.0.0.1:%d", cfg.Ports.NomadHTTP)
	nomad, err := api.NewClient(nomadCfg)
	if err != nil {
		return err
	}
	_, err = nomad.Agent().Datacenter()
	if err != nil {
		return err
	}
	return nil
}
