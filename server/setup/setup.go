package setup

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/server/agent"
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
	RunAPI     bool
	Wait       bool

	encryptKey  string
	masterToken string
}

func (s *Setup) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.StringVar(&s.ClusterID, "cluster-id", s.ClusterID, "five-character cluster ID")
	fs.BoolVar(&s.InitVault, "init-vault", s.InitVault, "initialize the vault if needed")
	fs.BoolVar(&s.Unseal, "unseal", s.Unseal, "unseal the vault automatically")
	fs.BoolVar(&s.RunAPI, "run-api", s.RunAPI, "run API server on this node")
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
		s.installConsulTemplate,
		s.installVault,
		s.maybeConfigure,
		s.generateSelfSignedCert,
		s.installArvadosServices,
	} {
		err := f()
		if err != nil {
			return err
		}
	}
	if s.Wait {
		return s.wait()
	} else {
		return nil
	}
}

func (s *Setup) wait() error {
	checkStatus := map[string]string{}
	sleep := 2 * time.Second
	for {
		cc, err := s.ConsulMaster()
		if err != nil {
			log.Printf("setup: consulMaster(): %s", err)
			continue
		}

		apiSvcs, _, err := cc.Catalog().Service("arvados-api", "", nil)
		if err != nil {
			log.Printf("setup: consul.Catalog().Service(): %s", err)
			continue
		} else if len(apiSvcs) == 0 {
			if sleep <= 2*time.Second {
				sleep = sleep * 2
				log.Printf("setup: waiting for arvados-api service to appear")
			}
			continue
		}

		ok := true
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
				if checkStatus[check.CheckID] != check.Status {
					log.Printf("setup: node %q service %q check %q state %q", check.Node, check.ServiceName, check.CheckID, check.Status)
				}
				if check.Status != "passing" {
					ok = false
				}
				checkStatus[check.CheckID] = check.Status
			}
		}
		if ok {
			log.Printf("All services are passing: %+v", svcs)
			return nil
		}
		time.Sleep(sleep)
	}
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
