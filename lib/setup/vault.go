package setup

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	consulAPI "github.com/hashicorp/consul/api"
	vaultAPI "github.com/hashicorp/vault/api"
)

func (s *Setup) installVault() error {
	if err := s.consulInit(); err != nil {
		return err
	}
	if err := s.vaultInit(); err != nil {
		return err
	}
	if s.vaultCheck() == nil {
		return nil
	}

	log.Printf("download & install vault")
	bin := s.UsrDir + "/bin/vault"
	err := (&download{
		URL:        "https://releases.hashicorp.com/vault/0.6.4/vault_0.6.4_linux_amd64.zip",
		Dest:       bin,
		Size:       52518022,
		Mode:       0755,
		PreloadDir: s.PreloadDir,
	}).install()
	if err != nil {
		return err
	}

	haAddr := fmt.Sprintf("http://%s:%d", s.LANHost, s.Ports.VaultServer)

	cfgPath := path.Join(s.DataDir, "vault.hcl")
	err = atomicWriteFile(cfgPath, []byte(fmt.Sprintf(`
		cluster_name = %q
		backend "consul" {
			address = "127.0.0.1:%d"
			redirect_addr = %q
			cluster_addr = %q
			path = %q
			token = %q
		}
		listener "tcp" {
			address = %q
			tls_disable = 1
		}
		`,
		s.ClusterID,
		s.Ports.ConsulHTTP,
		haAddr,
		haAddr,
		"vault-"+s.ClusterID+"/",
		s.masterToken,
		fmt.Sprintf("%s:%d", s.LANHost, s.Ports.VaultServer),
	)), 0600)
	if err != nil {
		return err
	}

	args := []string{"server", "-config=" + cfgPath}
	err = s.installService(daemon{
		name:       "arvados-vault",
		prog:       bin,
		args:       args,
		noRegister: true,
	})
	if err != nil {
		return err
	}

	if !s.Unseal {
		return nil
	}

	if err := s.vaultBootstrap(); err != nil {
		return err
	}
	return waitCheck(30*time.Second, s.vaultCheck)
}

func (s *Setup) vaultBootstrap() error {
	var vault *vaultAPI.Client
	var initialized bool
	resp := &vaultAPI.InitResponse{}
	if err := waitCheck(time.Minute, func() error {
		var err error
		vault, err = s.vaultClient()
		if err != nil {
			return err
		}
		initialized, err = vault.Sys().InitStatus()
		if err != nil {
			return err
		} else if s.InitVault {
			return nil
		}
		_, err = os.Stat(path.Join(s.DataDir, "vault", "mgmt-token.txt"))
		if err != nil {
			log.Print("vault is not initialized, waiting")
			return fmt.Errorf("vault is not initialized")
		}
		return nil
	}); err != nil {
		return err
	} else if !initialized && s.InitVault {
		resp, err = vault.Sys().Init(&vaultAPI.InitRequest{
			SecretShares:    5,
			SecretThreshold: 3,
		})
		if err != nil {
			return fmt.Errorf("vault-init: %s", err)
		}
		atomicWriteJSON(path.Join(s.DataDir, "vault", "keys.json"), resp, 0400)
		atomicWriteFile(path.Join(s.DataDir, "vault", "root-token.txt"), []byte(resp.RootToken), 0400)
	} else {
		j, err := ioutil.ReadFile(path.Join(s.DataDir, "vault", "keys.json"))
		if err != nil {
			return err
		}
		err = json.Unmarshal(j, resp)
		if err != nil {
			return err
		}
	}
	vault.SetToken(resp.RootToken)

	ok := false
	for _, key := range resp.Keys {
		resp, err := vault.Sys().Unseal(key)
		if err != nil {
			log.Printf("error: unseal: %s", err)
			continue
		}
		if !resp.Sealed {
			log.Printf("unseal successful")
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("vault unseal failed!")
	}

	if s.InitVault {
		// Use master token to create a management token
		master, err := s.consulMaster()
		if err != nil {
			return err
		}
		mgmtToken, _, err := master.ACL().Create(&consulAPI.ACLEntry{Name: "vault", Type: "management"}, nil)
		if err != nil {
			return err
		}

		// Mount+configure consul backend
		alreadyMounted := false
		if err = waitCheck(30*time.Second, func() error {
			// Typically this first fails "500 node not active but
			// active node not found" but then succeeds.
			err := vault.Sys().Mount("consul", &vaultAPI.MountInput{Type: "consul"})
			if err != nil && strings.Contains(err.Error(), "existing mount at consul") {
				alreadyMounted = true
				err = nil
			}
			return err
		}); err != nil {
			return err
		}
		_, err = vault.Logical().Write("consul/config/access", map[string]interface{}{
			"address": fmt.Sprintf("127.0.0.1:%d", s.Ports.ConsulHTTP),
			"token":   string(mgmtToken),
		})
		if err != nil {
			return err
		}

		// Create a role
		_, err = vault.Logical().Write("consul/roles/write-all", map[string]interface{}{
			"policy": base64.StdEncoding.EncodeToString([]byte(`key "" { policy = "write" }`)),
		})
		if err != nil {
			return err
		}

		// Write mgmtToken after bootstrapping is done. If
		// other nodes share our vault data dir, this is their
		// signal to try unseal.
		if err = atomicWriteFile(path.Join(s.DataDir, "vault", "mgmt-token.txt"), []byte(mgmtToken), 0400); err != nil {
			return err
		}
	}

	// Test: generate a new token with the write-all role
	secret, err := vault.Logical().Read("consul/creds/write-all")
	if err != nil {
		return err
	}
	token, ok := secret.Data["token"].(string)
	if !ok {
		return fmt.Errorf("secret token broken?? %+v", secret)
	}
	log.Printf("Vault supplied token with lease duration %s (renewable=%v): %q", time.Duration(secret.LeaseDuration)*time.Second, secret.Renewable, token)

	return nil
}

func (s *Setup) vaultInit() error {
	s.vaultCfg = vaultAPI.DefaultConfig()
	s.vaultCfg.Address = fmt.Sprintf("http://%s:%d", s.LANHost, s.Ports.VaultServer)
	return nil
}

func (s *Setup) vaultClient() (*vaultAPI.Client, error) {
	return vaultAPI.NewClient(s.vaultCfg)
}

func (s *Setup) vaultCheck() error {
	vault, err := s.vaultClient()
	if err != nil {
		return err
	}
	token, err := ioutil.ReadFile(path.Join(s.DataDir, "vault", "root-token.txt"))
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
