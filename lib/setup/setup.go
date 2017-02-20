package setup

import (
	"flag"
	"fmt"
	"os"

	"git.curoverse.com/arvados.git/lib/agent"
	"git.curoverse.com/arvados.git/sdk/go/config"
)

func Command() *Setup {
	return &Setup{
		Agent:      agent.Command(),
		PreloadDir: "/var/cache/arvados",
	}
}

type Setup struct {
	*agent.Agent
	PreloadDir string

	masterToken string
}

func (s *Setup) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.StringVar(&s.ClusterID, "cluster-id", s.ClusterID, "five-character cluster ID")
	fs.BoolVar(&s.Unseal, "unseal", s.Unseal, "unseal the vault automatically")
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
	} {
		err := f()
		if err != nil {
			return err
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
