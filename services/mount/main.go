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
	var coll arvados.Collection
	flag.StringVar(&coll.UUID, "id", "", "collection `uuid` or pdh")
	flag.Parse()
	client := arvados.NewClientFromEnv()
	err := client.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+coll.UUID, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	ac, err := arvadosclient.New(client)
	if err != nil {
		log.Fatal(err)
	}
	kc, err := keepclient.MakeKeepClient(ac)
	if err != nil {
		log.Fatal(err)
	}
	fs, err := coll.FileSystem(client, kc)
	if err != nil {
		log.Fatal(err)
	}
	host := fuse.NewFileSystemHost(&keepFS{root: fs})
	host.Mount("", flag.Args())
}
