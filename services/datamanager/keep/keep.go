/* Deals with getting Keep Server blocks from API Server and Keep Servers. */

package keep

import (
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"log"
)

type ServerAddress struct {
	Host string `json:"service_host"`
	Port int `json:"service_port"`
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

type KeepServiceList struct {
	ItemsAvailable int `json:"items_available"`
	Items []ServerAddress `json:"items"`
}

// Methods to implement util.SdkListResponse Interface
func (k KeepServiceList) NumItemsAvailable() (numAvailable int, err error) {
	return k.ItemsAvailable, nil
}

func (k KeepServiceList) NumItemsContained() (numContained int, err error) {
	return len(k.Items), nil
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

	var sdkResponse KeepServiceList
	err := params.Client.List("keep_services", sdkParams, &sdkResponse)
	if err != nil {
		log.Fatalf("Error requesting keep disks from API server: %v", err)
	}

	log.Printf("Received keep services list: %v", sdkResponse)

	{
		var numReceived, numAvailable int
		results.ReadAllServers, numReceived, numAvailable =
			util.ContainsAllAvailableItems(sdkResponse)

		if (!results.ReadAllServers) {
			log.Printf("ERROR: Did not receive all keep server addresses.")
		}
		log.Printf("Received %d of %d available keep server addresses.",
			numReceived,
			numAvailable)
	}
	return
}
