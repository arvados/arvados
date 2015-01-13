/* Deals with getting Keep Server blocks from API Server and Keep Servers. */

package keep

import (
	"bufio"
	"flag"
	"fmt"
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"git.curoverse.com/arvados.git/services/datamanager/loggerutil"
	"log"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type ServerAddress struct {
	Host string `json:"service_host"`
	Port int `json:"service_port"`
}

// Info about a particular block returned by the server
type BlockInfo struct {
	Digest     blockdigest.BlockDigest
	Size       int
	Mtime      int  // TODO(misha): Replace this with a timestamp.
}

// Info about a specified block given by a server
type BlockServerInfo struct {
	ServerIndex int
	Size        int
	Mtime       int  // TODO(misha): Replace this with a timestamp.
}

type ServerContents struct {
	BlockDigestToInfo map[blockdigest.BlockDigest]BlockInfo
}

type ServerResponse struct {
	Address ServerAddress
	Contents ServerContents
}

type ReadServers struct {
	ReadAllServers            bool
	KeepServerIndexToAddress  []ServerAddress
	KeepServerAddressToIndex  map[ServerAddress]int
	ServerToContents          map[ServerAddress]ServerContents
	BlockToServers            map[blockdigest.BlockDigest][]BlockServerInfo
	BlockReplicationCounts    map[int]int
}

type GetKeepServersParams struct {
	Client arvadosclient.ArvadosClient
	Logger *logger.Logger
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

func getDataManagerToken(arvLogger *logger.Logger) (string) {
	readDataManagerToken := func () {
		if dataManagerTokenFile == "" {
			flag.Usage()
			loggerutil.FatalWithMessage(arvLogger,
				"Data Manager Token needed, but data manager token file not specified.")
		} else {
			rawRead, err := ioutil.ReadFile(dataManagerTokenFile)
			if err != nil {
				loggerutil.FatalWithMessage(arvLogger,
					fmt.Sprintf("Unexpected error reading token file %s: %v",
						dataManagerTokenFile,
						err))
			}
			dataManagerToken = strings.TrimSpace(string(rawRead))
		}
	}

	dataManagerTokenFileReadOnce.Do(readDataManagerToken)
	return dataManagerToken
}

func GetKeepServersAndSummarize(params GetKeepServersParams) (results ReadServers) {
	results = GetKeepServers(params)
	log.Printf("Returned %d keep disks", len(results.ServerToContents))

	ComputeBlockReplicationCounts(&results)
	log.Printf("Replication level distribution: %v",
		results.BlockReplicationCounts)

	return
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
	err := params.Client.Call("GET", "keep_services", "", "accessible", sdkParams, &sdkResponse)

	if err != nil {
		loggerutil.FatalWithMessage(params.Logger,
			fmt.Sprintf("Error requesting keep disks from API server: %v", err))
	}

	// TODO(misha): Rewrite this block, stop using ContainsAllAvailableItems()
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

	if params.Logger != nil {
		properties,_ := params.Logger.Edit()
		keepInfo := make(map[string]interface{})

		keepInfo["num_keep_servers_available"] = sdkResponse.ItemsAvailable
		keepInfo["num_keep_servers_received"] = len(sdkResponse.KeepServers)
		keepInfo["keep_servers"] = sdkResponse.KeepServers

		properties["keep_info"] = keepInfo

		params.Logger.Record()
	}

	log.Printf("Received keep services list: %v", sdkResponse)

	results.KeepServerIndexToAddress = sdkResponse.KeepServers
	results.KeepServerAddressToIndex = make(map[ServerAddress]int)
	for i, address := range results.KeepServerIndexToAddress {
		results.KeepServerAddressToIndex[address] = i
	}

	log.Printf("Got Server Addresses: %v", results)

	// This is safe for concurrent use
	client := http.Client{}

	// Send off all the index requests concurrently
	responseChan := make(chan ServerResponse)
	for _, keepServer := range sdkResponse.KeepServers {
		go GetServerContents(params.Logger, keepServer, client, responseChan)
	}

	results.ServerToContents = make(map[ServerAddress]ServerContents)
	results.BlockToServers = make(map[blockdigest.BlockDigest][]BlockServerInfo)

	// Read all the responses
	for i := range sdkResponse.KeepServers {
		_ = i  // Here to prevent go from complaining.
		response := <- responseChan
		log.Printf("Received channel response from %v containing %d files",
			response.Address,
			len(response.Contents.BlockDigestToInfo))
		results.ServerToContents[response.Address] = response.Contents
		serverIndex := results.KeepServerAddressToIndex[response.Address]
		for _, blockInfo := range response.Contents.BlockDigestToInfo {
			results.BlockToServers[blockInfo.Digest] = append(
				results.BlockToServers[blockInfo.Digest],
				BlockServerInfo{ServerIndex: serverIndex,
					Size: blockInfo.Size,
					Mtime: blockInfo.Mtime})
		}
	}
	return
}

// TODO(misha): Break this function apart into smaller, easier to
// understand functions.
func GetServerContents(arvLogger *logger.Logger,
	keepServer ServerAddress,
	client http.Client,
	responseChan chan<- ServerResponse) () {
	// Create and send request.
	url := fmt.Sprintf("http://%s:%d/index", keepServer.Host, keepServer.Port)
	log.Println("About to fetch keep server contents from " + url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error building http request for %s: %v", url, err)
	}

	req.Header.Add("Authorization",
		fmt.Sprintf("OAuth2 %s", getDataManagerToken(arvLogger)))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error fetching %s: %v", url, err)
	}

	// Process response.
	if resp.StatusCode != 200 {
		log.Fatalf("Received error code %d in response to request for %s: %s",
			resp.StatusCode, url, resp.Status)
	}

	response := ServerResponse{}
	response.Address = keepServer
	response.Contents.BlockDigestToInfo =
		make(map[blockdigest.BlockDigest]BlockInfo)
	scanner := bufio.NewScanner(resp.Body)
	numLines, numDuplicates, numSizeDisagreements := 0, 0, 0
	for scanner.Scan() {
		numLines++
		blockInfo, err := parseBlockInfoFromIndexLine(scanner.Text())
		if err != nil {
			log.Fatalf("Error parsing BlockInfo from index line received from %s: %v",
				url,
				err)
		}

		if storedBlock, ok := response.Contents.BlockDigestToInfo[blockInfo.Digest]; ok {
			// This server returned multiple lines containing the same block digest.
			numDuplicates += 1
			if storedBlock.Size != blockInfo.Size {
				numSizeDisagreements += 1
				// TODO(misha): Consider failing here.
				log.Printf("Saw different sizes for the same block on %s: %v %v",
					url,
					storedBlock,
					blockInfo)
			}
			// Keep the block that is bigger, or the block that's newer in
			// the case of a size tie.
			if storedBlock.Size < blockInfo.Size ||
				(storedBlock.Size == blockInfo.Size &&
				storedBlock.Mtime < blockInfo.Mtime) {
				response.Contents.BlockDigestToInfo[blockInfo.Digest] = blockInfo
			}
		} else {
			response.Contents.BlockDigestToInfo[blockInfo.Digest] = blockInfo
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Received error scanning response from %s: %v", url, err)
	} else {
		log.Printf("%s contained %d lines with %d duplicates with " +
			"%d size disagreements",
			url,
			numLines,
			numDuplicates,
			numSizeDisagreements)
	}
	resp.Body.Close()
	responseChan <- response
}

func parseBlockInfoFromIndexLine(indexLine string) (blockInfo BlockInfo, err error) {
	tokens := strings.Fields(indexLine)
	if len(tokens) != 2 {
		err = fmt.Errorf("Expected 2 tokens per line but received a " + 
			"line containing %v instead.",
			tokens)
	}

	var locator manifest.BlockLocator
	if locator, err = manifest.ParseBlockLocator(tokens[0]); err != nil {
		return
	}
	if len(locator.Hints) > 0 {
		err = fmt.Errorf("Block locator in index line should not contain hints " +
			"but it does: %v",
			locator)
		return
	}

	blockInfo.Mtime, err = strconv.Atoi(tokens[1])
	if err != nil {
		return
	}
	blockInfo.Digest = locator.Digest
	blockInfo.Size = locator.Size
	return
}

func ComputeBlockReplicationCounts(readServers *ReadServers) {
	readServers.BlockReplicationCounts = make(map[int]int)
	for _, infos := range readServers.BlockToServers {
		replication := len(infos)
		readServers.BlockReplicationCounts[replication] += 1
	}
}
