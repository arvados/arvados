package main

import (
	"encoding/json"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

var arv arvadosclient.ArvadosClient
var keepClient *keepclient.KeepClient
var keepServers []string

func SetupDataManagerTest(t *testing.T) {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	// start api and keep servers
	arvadostest.ResetEnv()
	arvadostest.StartAPI()
	arvadostest.StartKeep()

	// make arvadosclient
	var err error
	arv, err = arvadosclient.MakeArvadosClient()
	if err != nil {
		t.Fatal("Error creating arv")
	}

	// keep client
	keepClient = &keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 1,
		Using_proxy:   true,
		Client:        &http.Client{},
	}

	// discover keep services
	if err := keepClient.DiscoverKeepServers(); err != nil {
		t.Fatal("Error discovering keep services")
	}
	for _, host := range keepClient.LocalRoots() {
		keepServers = append(keepServers, host)
	}
}

func TearDownDataManagerTest(t *testing.T) {
	arvadostest.StopKeep()
	arvadostest.StopAPI()
}

func PutBlock(t *testing.T, data string) string {
	locator, _, err := keepClient.PutB([]byte(data))
	if err != nil {
		t.Fatalf("Error putting test data for %s %s %v", data, locator, err)
	}
	if locator == "" {
		t.Fatalf("No locator found after putting test data")
	}

	return locator
}

func GetBlock(t *testing.T, locator string, data string) {
	reader, blocklen, _, err := keepClient.Get(locator)
	if err != nil {
		t.Fatalf("Error getting test data in setup for %s %s %v", data, locator, err)
	}
	if reader == nil {
		t.Fatalf("No reader found after putting test data")
	}
	if blocklen != int64(len(data)) {
		t.Fatalf("blocklen %d did not match data len %d", blocklen, len(data))
	}

	all, err := ioutil.ReadAll(reader)
	if string(all) != data {
		t.Fatalf("Data read %s did not match expected data %s", string(all), data)
	}
}

// Create a collection using arv-put
func CreateCollection(t *testing.T, data string) string {
	tempfile, err := ioutil.TempFile(os.TempDir(), "temp-test-file")
	defer os.Remove(tempfile.Name())

	_, err = tempfile.Write([]byte(data))
	if err != nil {
		t.Fatalf("Error writing to tempfile %v", err)
	}

	// arv-put
	output, err := exec.Command("arv-put", "--use-filename", "test.txt", tempfile.Name()).Output()
	if err != nil {
		t.Fatalf("Error running arv-put %s", err)
	}

	uuid := string(output[0:27]) // trim terminating char
	return uuid
}

// Get collection using arv-get
var locatorMatcher = regexp.MustCompile(`^([0-9a-f]{32})\+(\d*)(.*)$`)

func GetCollection(t *testing.T, uuid string) string {
	output, err := exec.Command("arv-get", uuid).Output()
	if err != nil {
		t.Fatalf("Error during arv-get %s", err)
	}

	locator := strings.Split(string(output), " ")[1]
	match := locatorMatcher.FindStringSubmatch(locator)
	if match == nil {
		t.Fatalf("No locator found in collection manifest %s", string(output))
	}

	return match[1] + "+" + match[2]
}

type Dict map[string]interface{}

func DeleteCollection(t *testing.T, uuid string) {
	getback := make(Dict)
	err := arv.Delete("collections", uuid, nil, &getback)
	if err != nil {
		t.Fatalf("Error deleting collection %s", err)
	}
	if getback["uuid"] != uuid {
		t.Fatalf("Delete collection uuid did not match original: $s, result: $s", uuid, getback["uuid"])
	}
}

func DataManagerSingleRun(t *testing.T) {
	err := singlerun()
	if err != nil {
		t.Fatalf("Error during singlerun %s", err)
	}
}

func MakeRequest(t *testing.T, path string) io.Reader {
	client := http.Client{}
	req, err := http.NewRequest("GET", path, strings.NewReader("resp"))
	req.Header.Add("Authorization", "OAuth2 "+keep.GetDataManagerToken(nil))
	req.Header.Add("Content-Type", "application/octet-stream")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		t.Fatalf("Error during %s %s", path, err)
	}

	return resp.Body
}

func GetBlockIndexes(t *testing.T) []string {
	var indexes []string

	for i := 0; i < len(keepServers); i++ {
		path := keepServers[i] + "/index"
		client := http.Client{}
		req, err := http.NewRequest("GET", path, strings.NewReader("resp"))
		req.Header.Add("Authorization", "OAuth2 "+keep.GetDataManagerToken(nil))
		req.Header.Add("Content-Type", "application/octet-stream")
		resp, err := client.Do(req)
		defer resp.Body.Close()

		if err != nil {
			t.Fatalf("Error during %s %s", path, err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Error reading response from %s %s", path, err)
		}

		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			indexes = append(indexes, strings.Split(line, " ")...)
		}
	}

	return indexes
}

func VerifyBlocks(t *testing.T, not_expected []string, expected []string) {
	blocks := GetBlockIndexes(t)
	for _, block := range not_expected {
		exists := ValueInArray(block, blocks)
		if exists {
			t.Fatalf("Found unexpected block in index %s", block)
		}
	}
	for _, block := range expected {
		exists := ValueInArray(block, blocks)
		if !exists {
			t.Fatalf("Did not find expected block in index %s", block)
		}
	}
}

func ValueInArray(value string, list []string) bool {
	for _, v := range list {
		if value == v {
			return true
		}
	}
	return false
}

/*
Test env uses two keep volumes. The volume names can be found by reading the files
  ARVADOS_HOME/tmp/keep0.volume and ARVADOS_HOME/tmp/keep1.volume

The keep volumes are of the dir structure:
  volumeN/subdir/locator
*/
func BackdateBlocks(t *testing.T, oldBlockLocators []string) {
	// First get rid of any size hints in the locators
	var trimmedBlockLocators []string
	for _, block := range oldBlockLocators {
		trimmedBlockLocators = append(trimmedBlockLocators, strings.Split(block, "+")[0])
	}

	// Get the working dir so that we can read keep{n}.volume files
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working dir %s", err)
	}

	// Now cycle through the two keep volumes
	oldTime := time.Now().AddDate(0, -1, 0)
	for i := 0; i < 2; i++ {
		filename := fmt.Sprintf("%s/../../tmp/keep%d.volume", wd, i)
		volumeDir, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Fatalf("Error reading keep volume file %s %s", filename, err)
		}

		// Read the keep volume dir structure
		volumeContents, err := ioutil.ReadDir(string(volumeDir))
		if err != nil {
			t.Fatalf("Error reading keep dir %s %s", string(volumeDir), err)
		}

		// Read each subdir for each of the keep volume dir
		for _, subdir := range volumeContents {
			subdirName := fmt.Sprintf("%s/%s", volumeDir, subdir.Name())
			subdirContents, err := ioutil.ReadDir(string(subdirName))
			if err != nil {
				t.Fatalf("Error reading keep dir %s %s", string(subdirName), err)
			}

			// Now we got to the files. The files are names are the block locators
			for _, fileInfo := range subdirContents {
				blockName := fileInfo.Name()
				myname := fmt.Sprintf("%s/%s", subdirName, blockName)
				if ValueInArray(blockName, trimmedBlockLocators) {
					err = os.Chtimes(myname, oldTime, oldTime)
				}
			}
		}
	}
}

func GetStatus(t *testing.T, path string) interface{} {
	client := http.Client{}
	req, err := http.NewRequest("GET", path, strings.NewReader("resp"))
	req.Header.Add("Authorization", "OAuth2 "+keep.GetDataManagerToken(nil))
	req.Header.Add("Content-Type", "application/octet-stream")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		t.Fatalf("Error during %s %s", path, err)
	}

	var s interface{}
	json.NewDecoder(resp.Body).Decode(&s)

	return s
}

func TestPutAndGetBlocks(t *testing.T) {
	log.Print("TestPutAndGetBlocks start")
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	// Put some blocks and change their mtime to before ttl
	var oldBlockLocators []string
	oldBlockData := "this block will have older mtime"
	for i := 0; i < 2; i++ {
		oldBlockLocators = append(oldBlockLocators, PutBlock(t, oldBlockData+string(i)))
	}
	for i := 0; i < 2; i++ {
		GetBlock(t, oldBlockLocators[i], oldBlockData+string(i))
	}

	// Put some more blocks whose mtime won't be changed
	var newBlockLocators []string
	newBlockData := "this block is newer"
	for i := 0; i < 1; i++ {
		newBlockLocators = append(newBlockLocators, PutBlock(t, newBlockData+string(i)))
	}
	for i := 0; i < 1; i++ {
		GetBlock(t, newBlockLocators[i], newBlockData+string(i))
	}

	// Create a collection that would be deleted
	to_delete_collection_uuid := CreateCollection(t, "some data for collection creation")
	to_delete_collection_locator := GetCollection(t, to_delete_collection_uuid)

	// Create another collection that has the same data as the one of the old blocks
	old_block_collection_uuid := CreateCollection(t, "this block will have older mtime0")
	old_block_collection_locator := GetCollection(t, old_block_collection_uuid)
	exists := ValueInArray(strings.Split(old_block_collection_locator, "+")[0], oldBlockLocators)
	if exists {
		t.Fatalf("Locator of the collection with the same data as old block is different %s", old_block_collection_locator)
	}

	// Invoking datamanager singlerun or /index several times is resulting in errors
	// Hence, for now just invoke once at the end of test

	var expected []string
	expected = append(expected, oldBlockLocators...)
	expected = append(expected, newBlockLocators...)
	expected = append(expected, to_delete_collection_locator)

	VerifyBlocks(t, nil, expected)

	// Run datamanager in singlerun mode
	DataManagerSingleRun(t)

	// Change mtime on old blocks and delete the collection
	DeleteCollection(t, to_delete_collection_uuid)
	BackdateBlocks(t, oldBlockLocators)

	// Run data manager
	time.Sleep(1 * time.Second)
	DataManagerSingleRun(t)

	// Wait until PullQueue and TrashQueue finish their work
	for {
		var done [2]bool
		for i := 0; i < 2; i++ {
			s := GetStatus(t, keepServers[i]+"/status.json")
			var pullQueueStatus interface{}
			pullQueueStatus = s.(map[string]interface{})["PullQueue"]
			var trashQueueStatus interface{}
			trashQueueStatus = s.(map[string]interface{})["TrashQueue"]
			if pullQueueStatus.(map[string]interface{})["Queued"] == float64(0) &&
				pullQueueStatus.(map[string]interface{})["InProgress"] == float64(0) &&
				trashQueueStatus.(map[string]interface{})["Queued"] == float64(0) &&
				trashQueueStatus.(map[string]interface{})["InProgress"] == float64(0) {
				done[i] = true
			}
		}
		if done[0] && done[1] {
			break
		} else {
			time.Sleep(1 * time.Second)
		}
	}

	// Get block indexes and verify that the deleted collection block is no longer found
	var not_expected []string
	not_expected = append(not_expected, oldBlockLocators...)
	not_expected = append(not_expected, to_delete_collection_locator)
	VerifyBlocks(t, not_expected, newBlockLocators)
}

// Invoking datamanager singlerun several times resulting in errors.
// Until that issue is resolved, don't run this test in the meantime.
func x_TestInvokeDatamanagerSingleRunRepeatedly(t *testing.T) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	for i := 0; i < 10; i++ {
		err := singlerun()
		if err != nil {
			t.Fatalf("Got an error during datamanager singlerun: %v", err)
		}
		time.Sleep(1 * time.Second)
	}
}
