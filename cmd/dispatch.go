package cmd

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"git.curoverse.com/arvados.git/sdk/go/config"
)

// A Command is a subcommand that can be invoked by Dispatch.
type Command interface {
	DefaultConfigFile() string
	ParseFlags([]string) error
	Run() error
}

// Dispatch parses flags from args, chooses an entry in cmds using the
// next argument after the parsed flags, loads the command's
// configuration file if it exists, passes any additional flags to the
// command's ParseFlags method, and -- if all of those steps complete
// without errors -- runs the command.
func Dispatch(cmds map[string]Command, prog string, args []string) error {
	fs := flag.NewFlagSet(prog, flag.ContinueOnError)
	err := fs.Parse(args)
	if err != nil {
		return err
	}

	subcmd := fs.Arg(0)
	cmd, ok := cmds[subcmd]
	if !ok {
		if subcmd != "" && subcmd != "help" {
			return fmt.Errorf("unrecognized subcommand %q", subcmd)
		}
		var subcmds []string
		for s := range cmds {
			subcmds = append(subcmds, s)
		}
		sort.Sort(sort.StringSlice(subcmds))
		return fmt.Errorf("available subcommands: %q", subcmds)
	}

	err = config.LoadFile(cmd, cmd.DefaultConfigFile())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if fs.NArg() > 1 {
		args = fs.Args()[1:]
	} else {
		args = nil
	}
	err = cmd.ParseFlags(args)
	if err != nil {
		return err
	}
	return cmd.Run()
}
