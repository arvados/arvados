package keepclient

import (
	"fmt"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
)

func ExampleRefreshServices() {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		panic(err)
	}
	kc, err := MakeKeepClient(&arv)
	if err != nil {
		panic(err)
	}
	go kc.RefreshServices(5*time.Minute, 3*time.Second)
	fmt.Printf("LocalRoots: %#v\n", kc.LocalRoots())
}
