package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Config struct {
	// 5 alphanumeric chars. Must be either xx*, yy*, zz*, or
	// globally unique.
	SiteID string

	// Hostnames or IP addresses of control hosts. Use at least 3
	// in production. System functions only when a majority are
	// alive.
	ControlHosts []string
	Ports        portsConfig
	WebGUI       webguiConfig
	DataDir      string
	UsrDir       string
	RunitSvDir   string

	ArvadosAptRepo aptRepoConfig
}

type portsConfig struct {
	ConsulDNS     int
	ConsulHTTP    int
	ConsulHTTPS   int
	ConsulRPC     int
	ConsulSerfLAN int
	ConsulSerfWAN int
	ConsulServer  int
	NomadHTTP     int
	NomadRPC      int
	NomadSerf     int
	VaultServer   int
}

type webguiConfig struct {
	// addr:port to serve web-based setup/monitoring
	// application
	Listen string
}

type aptRepoConfig struct {
	Enabled bool
	URL     string
	Release string
}

func (c *Config) Boot(ctx context.Context) error {
	for _, path := range []string{c.DataDir, c.UsrDir, c.UsrDir + "/bin"} {
		if fi, err := os.Stat(path); err != nil {
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return err
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s: is not a directory", path)
		}
	}
	return nil
}

func DefaultConfig() *Config {
	var repoConf aptRepoConfig
	if rel, err := ioutil.ReadFile("/etc/os-release"); err == nil {
		rel := string(rel)
		for _, try := range []string{"jessie", "precise", "xenial"} {
			if !strings.Contains(rel, try) {
				continue
			}
			repoConf = aptRepoConfig{
				Enabled: true,
				URL:     "http://apt.arvados.org/",
				Release: try,
			}
			break
		}
	}
	return &Config{
		SiteID:         "zzzzz",
		ArvadosAptRepo: repoConf,
		ControlHosts:   []string{"127.0.0.1"},
		Ports: portsConfig{
			ConsulDNS:     18600,
			ConsulHTTP:    18500,
			ConsulHTTPS:   -1,
			ConsulRPC:     18400,
			ConsulSerfLAN: 18301,
			ConsulSerfWAN: 18302,
			ConsulServer:  18300,
			NomadHTTP:     14646,
			NomadRPC:      14647,
			NomadSerf:     14648,
			VaultServer:   18200,
		},
		DataDir:    "/var/lib/arvados",
		UsrDir:     "/usr/local/arvados",
		RunitSvDir: "/etc/sv",
		WebGUI: webguiConfig{
			Listen: "127.0.0.1:18000",
		},
	}
}

func (cfg *Config) masterTokenFile() string {
	return path.Join(cfg.DataDir, "consul-master-token.txt")
}
