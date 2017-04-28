package configure

import (
	"flag"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/server/setup"
)

func Command() *Configure {
	return &Configure{
		Setup: setup.Command(),
	}
}

type Configure struct {
	*setup.Setup
}

func (c *Configure) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("configure", flag.ContinueOnError)
	return fs.Parse(args)
}

func (c *Configure) Run() error {
	err := config.LoadFile(c, c.DefaultConfigFile())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return c.Setup.Reconfigure()
}
