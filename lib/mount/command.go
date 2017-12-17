package mount

import (
	"flag"
	"log"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"github.com/curoverse/cgofuse/fuse"
)

func Run(prog string, args []string) int {
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	ro := flags.Bool("ro", false, "read-only")
	err := flags.Parse(args)
	if err != nil {
		log.Print(err)
		return 2
	}

	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	if err != nil {
		log.Fatal(err)
	}
	kc, err := keepclient.MakeKeepClient(ac)
	if err != nil {
		log.Fatal(err)
	}
	host := fuse.NewFileSystemHost(&keepFS{
		Client:     client,
		KeepClient: kc,
		ReadOnly:   *ro,
	})
	notOK := host.Mount("", flags.Args())
	if notOK {
		return 1
	} else {
		return 0
	}
}
