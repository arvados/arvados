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

func (s *Setup) consulTemplateTrigger(name, dst, tmpl string, mode os.FileMode, reload string) error {
	atomicWriteFile(dst+".tmpl", []byte(tmpl), mode)

	svdir := "/etc/sv/" + name
	cfgPath := svdir + "/consul.json"
	atomicWriteJSON(cfgPath, map[string]interface{}{
		"consul": map[string]interface{}{
			"address": fmt.Sprintf("%s:%d", s.LANHost, s.Agent.Ports.ConsulHTTP),
			"token":   s.masterToken,
		},
		"vault": map[string]interface{}{
			"address": fmt.Sprintf("http://%s:%d", s.LANHost, s.Agent.Ports.VaultServer),
		},
	}, 0600)

	ct := path.Join(s.UsrDir, "bin", "consul-template")
	args := []string{
		"-config", cfgPath, "-template", dst + ".tmpl:" + dst + ":" + reload,
	}
	script := fmt.Sprintf("#!/bin/sh\nexec %q ", ct)
	for _, a := range args {
		script = script + fmt.Sprintf(" %q", a)
	}
	script = script + "\n"

	atomicWriteFile(svdir+"/run", []byte(script), 0755)
	err := command("sv", "term", svdir).Run()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		// "sv could not send term" is ok, but "sv not found" is not ok
		return err
	}
	return nil
}

func (s *Setup) installConsulTemplate() error {
	prog := path.Join(s.UsrDir, "bin", "consul-template")
	return (&download{
		URL:        "https://releases.hashicorp.com/consul-template/0.18.1/consul-template_0.18.1_linux_amd64.zip",
		Dest:       prog,
		Size:       6932736,
		Mode:       0755,
		PreloadDir: s.PreloadDir,
	}).install()
}

func (s *Setup) installConsul() error {
	prog := path.Join(s.UsrDir, "bin", "consul")
	err := (&download{
		URL:        "https://releases.hashicorp.com/consul/0.7.5/consul_0.7.5_linux_amd64.zip",
		Dest:       prog,
		Size:       36003713,
		Mode:       0755,
		PreloadDir: s.PreloadDir,
	}).install()
	if err != nil {
		return err
	}

	if err := s.consulInit(); err != nil {
		return err
	}
	if s.consulCheck() == nil {
		return nil
	}

	dataDir := path.Join(s.DataDir, "consul")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}

	cf := path.Join(s.DataDir, "consul-config.json")
	{
		c := map[string]interface{}{
			"acl_agent_token":       s.masterToken,
			"acl_datacenter":        s.ClusterID,
			"acl_default_policy":    "deny",
			"acl_enforce_version_8": true,
			"acl_master_token":      s.masterToken,
			"bootstrap_expect":      len(s.ControlHosts),
			"client_addr":           "0.0.0.0",
			"data_dir":              dataDir,
			"datacenter":            s.ClusterID,
			"encrypt":               s.encryptKey,
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
		}
		err = atomicWriteJSON(cf, c, 0600)
		if err != nil {
			return err
		}
	}

	err = s.installService(daemon{
		name:       "arvados-consul",
		prog:       prog,
		args:       []string{"agent", "-config-file=" + cf},
		noRegister: true,
	})
	if err != nil {
		return err
	}
	if err = waitCheck(20*time.Second, s.consulCheck); err != nil {
		return err
	}
	if len(s.ControlHosts) > 1 {
		args := []string{"join"}
		args = append(args, fmt.Sprintf("-rpc-addr=127.0.0.1:%d", s.Ports.ConsulRPC))
		args = append(args, s.ControlHosts...)
		cmd := exec.Command(prog, args...)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("consul join: %s", err)
		}
	}
	return nil
}

var consulCfg = api.DefaultConfig()

func (s *Setup) ConsulMaster() (*api.Client, error) {
	if s.masterToken == "" {
		t, err := ioutil.ReadFile(path.Join(s.DataDir, "master-token.txt"))
		if err != nil {
			return nil, err
		}
		s.masterToken = string(t)
	}
	ccfg := api.DefaultConfig()
	ccfg.Address = fmt.Sprintf("127.0.0.1:%d", s.Ports.ConsulHTTP)
	ccfg.Datacenter = s.ClusterID
	ccfg.Token = s.masterToken
	return api.NewClient(ccfg)
}

func (s *Setup) consulInit() error {
	prog := path.Join(s.UsrDir, "bin", "consul")
	keyPath := path.Join(s.DataDir, "encrypt-key.txt")
	key, err := ioutil.ReadFile(keyPath)
	if os.IsNotExist(err) {
		key, err = exec.Command(prog, "keygen").CombinedOutput()
		if err != nil {
			return err
		}
		err = atomicWriteFile(keyPath, key, 0400)
	}
	if err != nil {
		return err
	}
	s.encryptKey = strings.TrimSpace(string(key))

	tokPath := path.Join(s.DataDir, "master-token.txt")
	if tok, err := ioutil.ReadFile(tokPath); err != nil {
		s.masterToken = generateToken()
		err = atomicWriteFile(tokPath, []byte(s.masterToken), 0600)
		if err != nil {
			return err
		}
	} else {
		s.masterToken = string(tok)
	}
	return nil
}

func (s *Setup) consulCheck() error {
	consul, err := s.ConsulMaster()
	if err != nil {
		return err
	}
	_, err = consul.Catalog().Datacenters()
	return err
}

// OnlyNode returns true if this is the only consul node.
func (s *Setup) OnlyNode() (bool, error) {
	c, err := s.ConsulMaster()
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
