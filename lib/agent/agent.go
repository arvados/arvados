package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type Agent struct {
	// 5 alphanumeric chars. Must be either xx*, yy*, zz*, or
	// globally unique.
	ClusterID string

	// "runit" or "systemd"
	DaemonSupervisor string

	// Hostnames or IP addresses of control hosts. Use at least 3
	// in production. System functions only when a majority are
	// alive.
	ControlHosts []string
	Ports        PortsConfig
	DataDir      string
	UsrDir       string
	RunitSvDir   string

	ArvadosAptRepo RepoConfig

	// Unseal the vault automatically at startup
	Unseal bool
}

type PortsConfig struct {
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

type RepoConfig struct {
	Enabled bool
	URL     string
	Release string
}

func Command() *Agent {
	var repoConf RepoConfig
	if rel, err := ioutil.ReadFile("/etc/os-release"); err == nil {
		rel := string(rel)
		for _, try := range []string{"jessie", "precise", "xenial"} {
			if !strings.Contains(rel, try) {
				continue
			}
			repoConf = RepoConfig{
				Enabled: true,
				URL:     "http://apt.arvados.org/",
				Release: try,
			}
			break
		}
	}
	ds := "runit"
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		ds = "systemd"
	}
	return &Agent{
		ClusterID:        "zzzzz",
		DaemonSupervisor: ds,
		ArvadosAptRepo:   repoConf,
		ControlHosts:     []string{"127.0.0.1"},
		Ports: PortsConfig{
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
		Unseal:     true,
	}
}

func (*Agent) ParseFlags(args []string) error {
	return nil
}

func (a *Agent) Run() error {
	return fmt.Errorf("not implemented: %T.Run()", a)
}

func (*Agent) DefaultConfigFile() string {
	return "/etc/arvados/agent/agent.yml"
}
