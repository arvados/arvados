package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
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
	args := []string{"agent"}
	{
		cf := path.Join(cfg.DataDir, "consul-encrypt.json")
		_, err := os.Stat(cf)
		if os.IsNotExist(err) {
			key, err := exec.Command(bin, "keygen").CombinedOutput()
			if err != nil {
				return err
			}
			err = atomicWriteJSON(cf, map[string]interface{}{
				"encrypt": strings.TrimSpace(string(key)),
			}, 0400)
		}
		if err != nil {
			return err
		}
		args = append(args, "-config-file="+cf)
	}
	{
		cf := path.Join(cfg.DataDir, "consul-ports.json")
		err = atomicWriteJSON(cf, map[string]interface{}{
			"client_addr":      "0.0.0.0",
			"bootstrap_expect": len(cfg.ControlHosts),
			"data_dir":         dataDir,
			"datacenter":       cfg.SiteID,
			"server":           true,
			"ui":               true,
			"ports": map[string]int{
				"dns":      cfg.Ports.ConsulDNS,
				"http":     cfg.Ports.ConsulHTTP,
				"https":    cfg.Ports.ConsulHTTPS,
				"rpc":      cfg.Ports.ConsulRPC,
				"serf_lan": cfg.Ports.ConsulSerfLAN,
				"serf_wan": cfg.Ports.ConsulSerfWAN,
				"server":   cfg.Ports.ConsulServer,
			},
		}, 0644)
		if err != nil {
			return err
		}
		args = append(args, "-config-file="+cf)
	}
	supervisor := newSupervisor(ctx, "arvados-consul", bin, args...)
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
	consulCfg.Address = fmt.Sprintf("127.0.0.1:%d", cfg.Ports.ConsulHTTP)
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

// OnlyNode returns true if this is the only consul node.
func (cb *consulBooter) OnlyNode() (bool, error) {
	c, err := api.NewClient(consulCfg)
	if err != nil {
		return false, err
	}
	nodes, _, err := c.Catalog().Nodes(nil)
	return len(nodes) == 1, err
}
