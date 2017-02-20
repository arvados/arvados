package main

import (
	"flag"
	"fmt"
	"os"

	"git.curoverse.com/arvados.git/cmd"
	"git.curoverse.com/arvados.git/lib/agent"
	"git.curoverse.com/arvados.git/lib/setup"
)

var cmds = map[string]cmd.Command{
	"agent": agent.Command(),
	"setup": setup.Command(),
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
