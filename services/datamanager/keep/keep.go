/* Deals with getting Keep Server blocks from API Server and Keep Servers. */

package keep

import (
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"log"
)

type ServerAddress struct {
	Host string
	Port int
}

type ServerContents struct {
	BlockDigestToSize map[string]int
}

type ReadServers struct {
	ReadAllServers bool
	AddressToContents map[ServerAddress]ServerContents
}

type GetKeepServersParams struct {
	Client arvadosclient.ArvadosClient
	Limit int
}

// TODO(misha): Send Keep requests in parallel
func GetKeepServers(params GetKeepServersParams) (results ReadServers) {
	if &params.Client == nil {
		log.Fatalf("params.Client passed to GetKeepServers() should " +
			"contain a valid ArvadosClient, but instead it is nil.")
	}

	sdkParams := arvadosclient.Dict{}
	if params.Limit > 0 {
		sdkParams["limit"] = params.Limit
	}

	var sdkResponse map[string]interface{}
	err := params.Client.List("keep_disks", sdkParams, &sdkResponse)
	if err != nil {
		log.Fatalf("Error requesting keep disks from API server: %v", err)
	}

	{
		var numReceived, numAvailable int
		results.ReadAllServers, numReceived, numAvailable =
			util.SdkListResponseContainsAllAvailableItems(sdkResponse)

		if (!results.ReadAllServers) {
			log.Printf("ERROR: Did not receive all keep server addresses.")
		}
		log.Printf("Received %d of %d available keep server addresses.",
			numReceived,
			numAvailable)
	}

	if addressChannel, err := util.IterateSdkListItems(sdkResponse); err != nil {
		log.Fatalf("Error trying to iterate keep server addresses returned " +
			"by SDK: %v", err)
	} else {
		results.AddressToContents = make(map[ServerAddress]ServerContents)
		for addressMap := range addressChannel {
			log.Printf("%v", addressMap)
			address := ServerAddress{Host: addressMap["service_host"].(string),
				Port: addressMap["service_port"].(int)}
			contents := ServerContents{BlockDigestToSize: make(map[string]int)}
			results.AddressToContents[address] = contents
		}
	}

	return
}
