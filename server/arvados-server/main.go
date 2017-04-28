package main

import (
	"flag"
	"fmt"
	"os"

	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/lib/configure"
	"git.curoverse.com/arvados.git/server/agent"
	"git.curoverse.com/arvados.git/server/setup"
)

var cmds = map[string]cmd.Command{
	"agent":     agent.Command(),
	"init":      setup.Command(),
	"setup":     setup.Command(),
	"configure": configure.Command(),
}

func main() {
	err := cmd.Dispatch(cmds, os.Args[0], os.Args[1:])
	if err != nil {
		if err != flag.ErrHelp {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		os.Exit(1)
	}
}
