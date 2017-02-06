package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"sync"
	"time"

	"github.com/hashicorp/vault/api"
)

var (
	vault    = &vaultBooter{}
	vaultCfg = api.DefaultConfig()
)

type vaultBooter struct {
	sync.Mutex
}

func (vb *vaultBooter) Boot(ctx context.Context) error {
	vb.Lock()
	defer vb.Unlock()

	if vb.check(ctx) == nil {
		return nil
	}
	cfg := cfg(ctx)
	bin := cfg.UsrDir + "/bin/vault"
	err := (&download{
		URL:  "https://releases.hashicorp.com/vault/0.6.4/vault_0.6.4_linux_amd64.zip",
		Dest: bin,
		Size: 52518022,
		Mode: 0755,
	}).Boot(ctx)
	if err != nil {
		return err
	}

	cfgPath := path.Join(cfg.DataDir, "vault.hcl")
	err = atomicWriteFile(cfgPath, []byte(fmt.Sprintf(`backend "consul" {
		address = "127.0.0.1:%d"
		path = "vault"
	}
	listener "tcp" {
		address = "127.0.0.1:%d"
		tls_disable = 1
	}`, cfg.Ports.ConsulHTTP, cfg.Ports.VaultServer)), 0644)
	if err != nil {
		return err
	}

	args := []string{"server", "-config=" + cfgPath}
	supervisor := newSupervisor(ctx, "arvados-vault", bin, args...)
	running, err := supervisor.Running(ctx)
	if err != nil {
		return err
	}
	if !running {
		defer feedbackf(ctx, "starting vault service")()
		err = supervisor.Start(ctx)
		if err != nil {
			return fmt.Errorf("starting vault: %s", err)
		}
	}

	if err := vb.tryInit(ctx); err != nil {
		return err
	}
	return waitCheck(ctx, 30*time.Second, vb.check)
}

func (vb *vaultBooter) tryInit(ctx context.Context) error {
	cfg := cfg(ctx)

	var vault *api.Client
	var init bool
	if err := waitCheck(ctx, time.Minute, func(context.Context) error {
		var err error
		vault, err = vb.client(ctx)
		if err != nil {
			return err
		}
		init, err = vault.Sys().InitStatus()
		return err
	}); err != nil {
		return err
	} else if init {
		return nil
	}

	resp, err := vault.Sys().Init(&api.InitRequest{
		SecretShares:    5,
		SecretThreshold: 3,
	})
	if err != nil {
		return fmt.Errorf("vault-init: %s", err)
	}
	atomicWriteJSON(path.Join(cfg.DataDir, "vault-keys.json"), resp, 0400)
	atomicWriteFile(path.Join(cfg.DataDir, "vault-root-token.txt"), []byte(resp.RootToken), 0400)

	for _, key := range resp.Keys {
		resp, err := vault.Sys().Unseal(key)
		if err != nil {
			log.Printf("error: unseal: %s", err)
			continue
		}
		if !resp.Sealed {
			log.Printf("unseal successful")
			return nil
		}
	}
	return fmt.Errorf("vault unseal failed!")
}

func (vb *vaultBooter) client(ctx context.Context) (*api.Client, error) {
	cfg := cfg(ctx)
	vaultCfg.Address = fmt.Sprintf("http://0.0.0.0:%d", cfg.Ports.VaultServer)
	return api.NewClient(vaultCfg)
}

func (vb *vaultBooter) check(ctx context.Context) error {
	cfg := cfg(ctx)
	vault, err := vb.client(ctx)
	if err != nil {
		return err
	}
	token, err := ioutil.ReadFile(path.Join(cfg.DataDir, "vault-root-token.txt"))
	if err != nil {
		return err
	}
	vault.SetToken(string(token))
	if init, err := vault.Sys().InitStatus(); err != nil {
		return err
	} else if !init {
		return fmt.Errorf("vault is not initialized")
	}
	return nil
}
