package setup

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"git.curoverse.com/arvados.git/lib/agent"
	"git.curoverse.com/arvados.git/sdk/go/config"
	vaultAPI "github.com/hashicorp/vault/api"
)

func Command() *Setup {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("hostname: %s", err)
	}

	return &Setup{
		Agent:      agent.Command(),
		LANHost:    hostname,
		PreloadDir: "/var/cache/arvados",
	}
}

type Setup struct {
	*agent.Agent
	InitVault  bool
	LANHost    string
	PreloadDir string
	Wait       bool

	encryptKey  string
	masterToken string
	vaultCfg    *vaultAPI.Config
}

func (s *Setup) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.StringVar(&s.ClusterID, "cluster-id", s.ClusterID, "five-character cluster ID")
	fs.BoolVar(&s.InitVault, "init-vault", s.InitVault, "initialize the vault if needed")
	fs.BoolVar(&s.Unseal, "unseal", s.Unseal, "unseal the vault automatically")
	fs.BoolVar(&s.Wait, "wait", s.Wait, "wait for all nodes to come up before exiting")
	return fs.Parse(args)
}

func (s *Setup) Run() error {
	err := config.LoadFile(s, s.DefaultConfigFile())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, f := range []func() error{
		s.makeDirs,
		(&osPackage{Debian: "ca-certificates"}).install,
		(&osPackage{Debian: "nginx"}).install,
		s.installRunit,
		s.installConsul,
		s.installVault,
	} {
		err := f()
		if err != nil {
			return err
		}
	}

	wait := 2 * time.Second
	for ok := false; s.Wait && !ok; time.Sleep(wait) {
		cc, err := s.consulMaster()
		if err != nil {
			log.Printf("setup: consulMaster(): %s", err)
			continue
		}
		ok = true
		svcs, _, err := cc.Catalog().Services(nil)
		if err != nil {
			log.Printf("setup: consul.Catalog().Services(): %s", err)
			continue
		}
		for svc := range svcs {
			checks, _, err := cc.Health().Checks(svc, nil)
			if err != nil {
				log.Printf("setup: consul.Health().Checks(%q): %s", svc, err)
				continue
			}

			for _, check := range checks {
				if check.Status != "passing" {
					log.Printf("waiting for node %q service %q check %q state %q", check.Node, check.ServiceName, check.CheckID, check.Status)
					ok = false
				}
			}
		}
		if ok {
			log.Printf("All services are passing: %+v", svcs)
			// Wait to ensure any other "setup -wait"
			// processes have a chance to see the
			// all-passing state before we return (if this
			// is a test or image-building scenario, the
			// whole system might shut down and stop
			// passing as soon as we return).
			time.Sleep(2 * wait)
		}
	}
	return nil
}

func (s *Setup) makeDirs() error {
	for _, path := range []string{s.DataDir, s.UsrDir, s.UsrDir + "/bin"} {
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
