package setup

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
)

func (s *Setup) installConsul() error {
	prog := s.UsrDir + "/bin/consul"
	err := (&download{
		URL:        "https://releases.hashicorp.com/consul/0.7.4/consul_0.7.4_linux_amd64.zip",
		Dest:       prog,
		Size:       36003597,
		Mode:       0755,
		PreloadDir: s.PreloadDir,
	}).install()
	if err != nil {
		return err
	}
	dataDir := path.Join(s.DataDir, "consul")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	args := []string{"agent"}
	{
		cf := path.Join(s.DataDir, "consul-encrypt.json")
		if _, err := os.Stat(cf); err != nil && !os.IsNotExist(err) {
			return err
		} else if err != nil {
			key, err := exec.Command(prog, "keygen").CombinedOutput()
			if err != nil {
				return err
			}
			if err = atomicWriteJSON(cf, map[string]interface{}{
				"encrypt": strings.TrimSpace(string(key)),
			}, 0400); err != nil {
				return err
			}
		}
		args = append(args, "-config-file="+cf)
	}
	{
		s.masterToken = generateToken()
		// os.Setenv("CONSUL_TOKEN", s.masterToken)
		err = atomicWriteFile(path.Join(s.DataDir, "master-token.txt"), []byte(s.masterToken), 0600)
		if err != nil {
			return err
		}
		cf := path.Join(s.DataDir, "consul-config.json")
		err = atomicWriteJSON(cf, map[string]interface{}{
			"acl_datacenter":        s.ClusterID,
			"acl_default_policy":    "deny",
			"acl_enforce_version_8": true,
			"acl_master_token":      s.masterToken,
			"client_addr":           "0.0.0.0",
			"bootstrap_expect":      len(s.ControlHosts),
			"data_dir":              dataDir,
			"datacenter":            s.ClusterID,
			"server":                true,
			"ui":                    true,
			"ports": map[string]int{
				"dns":      s.Ports.ConsulDNS,
				"http":     s.Ports.ConsulHTTP,
				"https":    s.Ports.ConsulHTTPS,
				"rpc":      s.Ports.ConsulRPC,
				"serf_lan": s.Ports.ConsulSerfLAN,
				"serf_wan": s.Ports.ConsulSerfWAN,
				"server":   s.Ports.ConsulServer,
			},
		}, 0644)
		if err != nil {
			return err
		}
		args = append(args, "-config-file="+cf)
	}
	err = s.installService(daemon{
		name:       "arvados-consul",
		prog:       prog,
		args:       args,
		noRegister: true,
	})
	if err != nil {
		return err
	}
	if len(s.ControlHosts) > 1 {
		cmd := exec.Command(prog, append([]string{"join"}, s.ControlHosts...)...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("consul join: %s", err)
		}
	}
	return waitCheck(20*time.Second, s.consulCheck)
}

var consulCfg = api.DefaultConfig()

func (s *Setup) consulMaster() (*api.Client, error) {
	masterToken, err := ioutil.ReadFile(path.Join(s.DataDir, "master-token.txt"))
	if err != nil {
		return nil, err
	}
	ccfg := api.DefaultConfig()
	ccfg.Address = fmt.Sprintf("127.0.0.1:%d", s.Ports.ConsulHTTP)
	ccfg.Datacenter = s.ClusterID
	ccfg.Token = string(masterToken)
	return api.NewClient(ccfg)
}

func (s *Setup) consulCheck() error {
	consul, err := s.consulMaster()
	if err != nil {
		return err
	}
	_, err = consul.Catalog().Datacenters()
	return err
}

// OnlyNode returns true if this is the only consul node.
func (s *Setup) OnlyNode() (bool, error) {
	c, err := s.consulMaster()
	if err != nil {
		return false, err
	}
	nodes, _, err := c.Catalog().Nodes(nil)
	return len(nodes) == 1, err
}

func generateToken() string {
	var r [16]byte
	if _, err := rand.Read(r[:]); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", r)
}
