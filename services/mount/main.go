package main

import (
	"flag"
	"log"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"github.com/billziss-gh/cgofuse/fuse"
)

func main() {
	ro := flag.Bool("ro", false, "read-only")
	flag.Parse()

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
	host.Mount("", flag.Args())
}
