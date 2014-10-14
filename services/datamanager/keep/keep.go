/* Deals with getting Keep Server blocks from API Server and Keep Servers. */

package keep

import (
	"flag"
	"fmt"
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"log"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
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
	KeepServers []ServerAddress `json:"items"`
}

// Methods to implement util.SdkListResponse Interface
func (k KeepServiceList) NumItemsAvailable() (numAvailable int, err error) {
	return k.ItemsAvailable, nil
}

func (k KeepServiceList) NumItemsContained() (numContained int, err error) {
	return len(k.KeepServers), nil
}

var (
	// Don't access the token directly, use getDataManagerToken() to
	// make sure it's been read.
	dataManagerToken                string
	dataManagerTokenFile            string
	dataManagerTokenFileReadOnce    sync.Once
)

func init() {
	flag.StringVar(&dataManagerTokenFile, 
		"data-manager-token-file",
		"",
		"File with the API token we should use to contact keep servers.")
}

func getDataManagerToken() (string) {
	readDataManagerToken := func () {
		if dataManagerTokenFile == "" {
			flag.Usage()
			log.Fatalf("Data Manager Token needed, but data manager token file not specified.")
		} else {
			rawRead, err := ioutil.ReadFile(dataManagerTokenFile)
			if err != nil {
				log.Fatalf("Unexpected error reading token file %s: %v",
					dataManagerTokenFile,
					err)
			}
			dataManagerToken = strings.TrimSpace(string(rawRead))
		}
	}

	dataManagerTokenFileReadOnce.Do(readDataManagerToken)
	return dataManagerToken
}

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

	// This is safe for concurrent use
	client := http.Client{}
	
	// TODO(misha): Do these in parallel
	for _, keepServer := range sdkResponse.KeepServers {
		url := fmt.Sprintf("http://%s:%d/index", keepServer.Host, keepServer.Port)
		log.Println("About to fetch keep server contents from " + url)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Error building http request for %s: %v", url, err)
		}

		req.Header.Add("Authorization",
			fmt.Sprintf("OAuth2 %s", getDataManagerToken()))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error fetching %s: %v", url, err)
		}

		if resp.StatusCode != 200 {
			log.Printf("%v", req)
			log.Fatalf("Received error code %d in response to request for %s: %s",
				resp.StatusCode, url, resp.Status)
		}
	}

	return
}
