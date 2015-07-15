/* Deals with getting Keep Server blocks from API Server and Keep Servers. */

package keep

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/services/datamanager/loggerutil"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ServerAddress struct {
	SSL  bool   `json:service_ssl_flag`
	Host string `json:"service_host"`
	Port int    `json:"service_port"`
	Uuid string `json:"uuid"`
}

// Info about a particular block returned by the server
type BlockInfo struct {
	Digest blockdigest.DigestWithSize
	Mtime  int64 // TODO(misha): Replace this with a timestamp.
}

// Info about a specified block given by a server
type BlockServerInfo struct {
	ServerIndex int
	Mtime       int64 // TODO(misha): Replace this with a timestamp.
}

type ServerContents struct {
	BlockDigestToInfo map[blockdigest.DigestWithSize]BlockInfo
}

type ServerResponse struct {
	Address  ServerAddress
	Contents ServerContents
}

type ReadServers struct {
	ReadAllServers           bool
	KeepServerIndexToAddress []ServerAddress
	KeepServerAddressToIndex map[ServerAddress]int
	ServerToContents         map[ServerAddress]ServerContents
	BlockToServers           map[blockdigest.DigestWithSize][]BlockServerInfo
	BlockReplicationCounts   map[int]int
}

type GetKeepServersParams struct {
	Client arvadosclient.ArvadosClient
	Logger *logger.Logger
	Limit  int
}

type KeepServiceList struct {
	ItemsAvailable int             `json:"items_available"`
	KeepServers    []ServerAddress `json:"items"`
}

var (
	// Don't access the token directly, use getDataManagerToken() to
	// make sure it's been read.
	dataManagerToken             string
	dataManagerTokenFile         string
	dataManagerTokenFileReadOnce sync.Once
)

func init() {
	flag.StringVar(&dataManagerTokenFile,
		"data-manager-token-file",
		"",
		"File with the API token we should use to contact keep servers.")
}

// TODO(misha): Change this to include the UUID as well.
func (s ServerAddress) String() string {
	return s.URL()
}

func (s ServerAddress) URL() string {
	if s.SSL {
		return fmt.Sprintf("https://%s:%d", s.Host, s.Port)
	} else {
		return fmt.Sprintf("http://%s:%d", s.Host, s.Port)
	}
}

func getDataManagerToken(arvLogger *logger.Logger) string {
	readDataManagerToken := func() {
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

	results.Summarize(params.Logger)
	log.Printf("Replication level distribution: %v",
		results.BlockReplicationCounts)

	return
}

func GetKeepServers(params GetKeepServersParams) (results ReadServers) {
	if &params.Client == nil {
		log.Fatalf("params.Client passed to GetKeepServers() should " +
			"contain a valid ArvadosClient, but instead it is nil.")
	}

	sdkParams := arvadosclient.Dict{
		"filters": [][]string{[]string{"service_type", "=", "disk"}},
	}
	if params.Limit > 0 {
		sdkParams["limit"] = params.Limit
	}

	var sdkResponse KeepServiceList
	err := params.Client.List("keep_services", sdkParams, &sdkResponse)

	if err != nil {
		loggerutil.FatalWithMessage(params.Logger,
			fmt.Sprintf("Error requesting keep disks from API server: %v", err))
	}

	if params.Logger != nil {
		params.Logger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			keepInfo := logger.GetOrCreateMap(p, "keep_info")
			keepInfo["num_keep_servers_available"] = sdkResponse.ItemsAvailable
			keepInfo["num_keep_servers_received"] = len(sdkResponse.KeepServers)
			keepInfo["keep_servers"] = sdkResponse.KeepServers
		})
	}

	log.Printf("Received keep services list: %+v", sdkResponse)

	if len(sdkResponse.KeepServers) < sdkResponse.ItemsAvailable {
		loggerutil.FatalWithMessage(params.Logger,
			fmt.Sprintf("Did not receive all available keep servers: %+v", sdkResponse))
	}

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
		// The above keepsServer variable is reused for each iteration, so
		// it would be shared across all goroutines. This would result in
		// us querying one server n times instead of n different servers
		// as we intended. To avoid this we add it as an explicit
		// parameter which gets copied. This bug and solution is described
		// in https://golang.org/doc/effective_go.html#channels
		go func(keepServer ServerAddress) {
			responseChan <- GetServerContents(params.Logger,
				keepServer,
				client)
		}(keepServer)
	}

	results.ServerToContents = make(map[ServerAddress]ServerContents)
	results.BlockToServers = make(map[blockdigest.DigestWithSize][]BlockServerInfo)

	// Read all the responses
	for i := range sdkResponse.KeepServers {
		_ = i // Here to prevent go from complaining.
		response := <-responseChan
		log.Printf("Received channel response from %v containing %d files",
			response.Address,
			len(response.Contents.BlockDigestToInfo))
		results.ServerToContents[response.Address] = response.Contents
		serverIndex := results.KeepServerAddressToIndex[response.Address]
		for _, blockInfo := range response.Contents.BlockDigestToInfo {
			results.BlockToServers[blockInfo.Digest] = append(
				results.BlockToServers[blockInfo.Digest],
				BlockServerInfo{ServerIndex: serverIndex,
					Mtime: blockInfo.Mtime})
		}
	}
	return
}

func GetServerContents(arvLogger *logger.Logger,
	keepServer ServerAddress,
	client http.Client) (response ServerResponse) {

	GetServerStatus(arvLogger, keepServer, client)

	req := CreateIndexRequest(arvLogger, keepServer)
	resp, err := client.Do(req)
	if err != nil {
		loggerutil.FatalWithMessage(arvLogger,
			fmt.Sprintf("Error fetching %s: %v. Response was %+v",
				req.URL.String(),
				err,
				resp))
	}

	return ReadServerResponse(arvLogger, keepServer, resp)
}

func GetServerStatus(arvLogger *logger.Logger,
	keepServer ServerAddress,
	client http.Client) {
	url := fmt.Sprintf("http://%s:%d/status.json",
		keepServer.Host,
		keepServer.Port)

	if arvLogger != nil {
		now := time.Now()
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			keepInfo := logger.GetOrCreateMap(p, "keep_info")
			serverInfo := make(map[string]interface{})
			serverInfo["status_request_sent_at"] = now
			serverInfo["host"] = keepServer.Host
			serverInfo["port"] = keepServer.Port

			keepInfo[keepServer.Uuid] = serverInfo
		})
	}

	resp, err := client.Get(url)
	if err != nil {
		loggerutil.FatalWithMessage(arvLogger,
			fmt.Sprintf("Error getting keep status from %s: %v", url, err))
	} else if resp.StatusCode != 200 {
		loggerutil.FatalWithMessage(arvLogger,
			fmt.Sprintf("Received error code %d in response to request "+
				"for %s status: %s",
				resp.StatusCode, url, resp.Status))
	}

	var keepStatus map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	err = decoder.Decode(&keepStatus)
	if err != nil {
		loggerutil.FatalWithMessage(arvLogger,
			fmt.Sprintf("Error decoding keep status from %s: %v", url, err))
	}

	if arvLogger != nil {
		now := time.Now()
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			keepInfo := logger.GetOrCreateMap(p, "keep_info")
			serverInfo := keepInfo[keepServer.Uuid].(map[string]interface{})
			serverInfo["status_response_processed_at"] = now
			serverInfo["status"] = keepStatus
		})
	}
}

func CreateIndexRequest(arvLogger *logger.Logger,
	keepServer ServerAddress) (req *http.Request) {
	url := fmt.Sprintf("http://%s:%d/index", keepServer.Host, keepServer.Port)
	log.Println("About to fetch keep server contents from " + url)

	if arvLogger != nil {
		now := time.Now()
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			keepInfo := logger.GetOrCreateMap(p, "keep_info")
			serverInfo := keepInfo[keepServer.Uuid].(map[string]interface{})
			serverInfo["index_request_sent_at"] = now
		})
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		loggerutil.FatalWithMessage(arvLogger,
			fmt.Sprintf("Error building http request for %s: %v", url, err))
	}

	req.Header.Add("Authorization",
		fmt.Sprintf("OAuth2 %s", getDataManagerToken(arvLogger)))
	return
}

func ReadServerResponse(arvLogger *logger.Logger,
	keepServer ServerAddress,
	resp *http.Response) (response ServerResponse) {

	if resp.StatusCode != 200 {
		loggerutil.FatalWithMessage(arvLogger,
			fmt.Sprintf("Received error code %d in response to request "+
				"for %s index: %s",
				resp.StatusCode, keepServer.String(), resp.Status))
	}

	if arvLogger != nil {
		now := time.Now()
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			keepInfo := logger.GetOrCreateMap(p, "keep_info")
			serverInfo := keepInfo[keepServer.Uuid].(map[string]interface{})
			serverInfo["index_response_received_at"] = now
		})
	}

	response.Address = keepServer
	response.Contents.BlockDigestToInfo =
		make(map[blockdigest.DigestWithSize]BlockInfo)
	reader := bufio.NewReader(resp.Body)
	numLines, numDuplicates, numSizeDisagreements := 0, 0, 0
	for {
		numLines++
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Index from %s truncated at line %d",
					keepServer.String(), numLines))
		} else if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Error reading index response from %s at line %d: %v",
					keepServer.String(), numLines, err))
		}
		if line == "\n" {
			if _, err := reader.Peek(1); err == nil {
				extra, _ := reader.ReadString('\n')
				loggerutil.FatalWithMessage(arvLogger,
					fmt.Sprintf("Index from %s had trailing data at line %d after EOF marker: %s",
						keepServer.String(), numLines+1, extra))
			} else if err != io.EOF {
				loggerutil.FatalWithMessage(arvLogger,
					fmt.Sprintf("Index from %s had read error after EOF marker at line %d: %v",
						keepServer.String(), numLines, err))
			}
			numLines--
			break
		}
		blockInfo, err := parseBlockInfoFromIndexLine(line)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Error parsing BlockInfo from index line "+
					"received from %s: %v",
					keepServer.String(),
					err))
		}

		if storedBlock, ok := response.Contents.BlockDigestToInfo[blockInfo.Digest]; ok {
			// This server returned multiple lines containing the same block digest.
			numDuplicates += 1
			// Keep the block that's newer.
			if storedBlock.Mtime < blockInfo.Mtime {
				response.Contents.BlockDigestToInfo[blockInfo.Digest] = blockInfo
			}
		} else {
			response.Contents.BlockDigestToInfo[blockInfo.Digest] = blockInfo
		}
	}

	log.Printf("%s index contained %d lines with %d duplicates with "+
		"%d size disagreements",
		keepServer.String(),
		numLines,
		numDuplicates,
		numSizeDisagreements)

	if arvLogger != nil {
		now := time.Now()
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			keepInfo := logger.GetOrCreateMap(p, "keep_info")
			serverInfo := keepInfo[keepServer.Uuid].(map[string]interface{})

			serverInfo["processing_finished_at"] = now
			serverInfo["lines_received"] = numLines
			serverInfo["duplicates_seen"] = numDuplicates
			serverInfo["size_disagreements_seen"] = numSizeDisagreements
		})
	}
	resp.Body.Close()
	return
}

func parseBlockInfoFromIndexLine(indexLine string) (blockInfo BlockInfo, err error) {
	tokens := strings.Fields(indexLine)
	if len(tokens) != 2 {
		err = fmt.Errorf("Expected 2 tokens per line but received a "+
			"line containing %v instead.",
			tokens)
	}

	var locator blockdigest.BlockLocator
	if locator, err = blockdigest.ParseBlockLocator(tokens[0]); err != nil {
		err = fmt.Errorf("%v Received error while parsing line \"%s\"",
			err, indexLine)
		return
	}
	if len(locator.Hints) > 0 {
		err = fmt.Errorf("Block locator in index line should not contain hints "+
			"but it does: %v",
			locator)
		return
	}

	blockInfo.Mtime, err = strconv.ParseInt(tokens[1], 10, 64)
	if err != nil {
		return
	}
	blockInfo.Digest =
		blockdigest.DigestWithSize{Digest: locator.Digest,
			Size: uint32(locator.Size)}
	return
}

func (readServers *ReadServers) Summarize(arvLogger *logger.Logger) {
	readServers.BlockReplicationCounts = make(map[int]int)
	for _, infos := range readServers.BlockToServers {
		replication := len(infos)
		readServers.BlockReplicationCounts[replication] += 1
	}

	if arvLogger != nil {
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			keepInfo := logger.GetOrCreateMap(p, "keep_info")
			keepInfo["distinct_blocks_stored"] = len(readServers.BlockToServers)
		})
	}

}

type TrashRequest struct {
	Locator    string `json:"locator"`
	BlockMtime int64  `json:"block_mtime"`
}

type TrashList []TrashRequest

func SendTrashLists(arvLogger *logger.Logger, kc *keepclient.KeepClient, spl map[string]TrashList) {
	count := 0
	rendezvous := make(chan bool)

	for url, v := range spl {
		count += 1
		log.Printf("Sending trash list to %v", url)

		go (func(url string, v TrashList) {
			defer (func() {
				rendezvous <- true
			})()

			pipeReader, pipeWriter := io.Pipe()
			go (func() {
				enc := json.NewEncoder(pipeWriter)
				enc.Encode(v)
				pipeWriter.Close()
			})()

			req, err := http.NewRequest("PUT", fmt.Sprintf("%s/trash", url), pipeReader)
			if err != nil {
				log.Printf("Error creating trash list request for %v error: %v", url, err.Error())
				return
			}

			// Add api token header
			req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", getDataManagerToken(arvLogger)))

			// Make the request
			var resp *http.Response
			if resp, err = kc.Client.Do(req); err != nil {
				log.Printf("Error sending trash list to %v error: %v", url, err.Error())
				return
			}

			log.Printf("Sent trash list to %v: response was HTTP %d", url, resp.Status)

			io.Copy(ioutil.Discard, resp.Body)
			resp.Body.Close()
		})(url, v)

	}

	for i := 0; i < count; i += 1 {
		<-rendezvous
	}
}
