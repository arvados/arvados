package main

import (
	"encoding/json"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	ActiveUserToken = "3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"
	AdminToken = "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h"
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

	arv = makeArvadosClient()

	// keep client
	keepClient = &keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 2,
		Using_proxy:   true,
		Client:        &http.Client{},
	}

	// discover keep services
	if err := keepClient.DiscoverKeepServers(); err != nil {
		t.Fatalf("Error discovering keep services: %s", err)
	}
	for _, host := range keepClient.LocalRoots() {
		keepServers = append(keepServers, host)
	}
}

func TearDownDataManagerTest(t *testing.T) {
	arvadostest.StopKeep()
	arvadostest.StopAPI()
}

func putBlock(t *testing.T, data string) string {
	locator, _, err := keepClient.PutB([]byte(data))
	if err != nil {
		t.Fatalf("Error putting test data for %s %s %v", data, locator, err)
	}
	if locator == "" {
		t.Fatalf("No locator found after putting test data")
	}

	splits := strings.Split(locator, "+")
	return splits[0] + "+" + splits[1]
}

func getBlock(t *testing.T, locator string, data string) {
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
func createCollection(t *testing.T, data string) string {
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

// Get collection locator
var locatorMatcher = regexp.MustCompile(`^([0-9a-f]{32})\+(\d*)(.*)$`)

func getFirstLocatorFromCollection(t *testing.T, uuid string) string {
	manifest := getCollection(t, uuid)["manifest_text"].(string)

	locator := strings.Split(manifest, " ")[1]
	match := locatorMatcher.FindStringSubmatch(locator)
	if match == nil {
		t.Fatalf("No locator found in collection manifest %s", manifest)
	}

	return match[1] + "+" + match[2]
}

func getCollection(t *testing.T, uuid string) Dict {
	getback := make(Dict)
	err := arv.Get("collections", uuid, nil, &getback)
	if err != nil {
		t.Fatalf("Error getting collection %s", err)
	}
	if getback["uuid"] != uuid {
		t.Fatalf("Get collection uuid did not match original: $s, result: $s", uuid, getback["uuid"])
	}

	return getback
}

func updateCollection(t *testing.T, uuid string, paramName string, paramValue string) {
	err := arv.Update("collections", uuid, arvadosclient.Dict{
		"collection": arvadosclient.Dict{
			paramName: paramValue,
		},
	}, &arvadosclient.Dict{})

	if err != nil {
		t.Fatalf("Error updating collection %s", err)
	}
}

type Dict map[string]interface{}

func deleteCollection(t *testing.T, uuid string) {
	getback := make(Dict)
	err := arv.Delete("collections", uuid, nil, &getback)
	if err != nil {
		t.Fatalf("Error deleting collection %s", err)
	}
	if getback["uuid"] != uuid {
		t.Fatalf("Delete collection uuid did not match original: $s, result: $s", uuid, getback["uuid"])
	}
}

func dataManagerSingleRun(t *testing.T) {
	err := singlerun(arv)
	if err != nil {
		t.Fatalf("Error during singlerun %s", err)
	}
}

func getBlockIndexesForServer(t *testing.T, i int) []string {
	var indexes []string

	path := keepServers[i] + "/index"
	client := http.Client{}
	req, err := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "OAuth2 " + AdminToken)
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

	return indexes
}

func getBlockIndexes(t *testing.T) [][]string {
	var indexes [][]string

	for i := 0; i < len(keepServers); i++ {
		indexes = append(indexes, getBlockIndexesForServer(t, i))
	}
	return indexes
}

func verifyBlocks(t *testing.T, notExpected []string, expected []string) {
	blocks := getBlockIndexes(t)
	for _, block := range notExpected {
		for i := 0; i < len(blocks); i++ {
			exists := valueInArray(block, blocks[i])
			if exists {
				t.Fatalf("Found unexpected block in index %s", block)
			}
		}
	}

	//	var blockExists [][]string
	blockExists := make(map[string][]string)
	for _, block := range expected {
		var blockArray []string
		for i := 0; i < len(blocks); i++ {
			exists := valueInArray(block, blocks[i])
			if exists {
				blockArray = append(blockArray, block)
			}
		}
		blockExists[block] = blockArray
	}

	for _, block := range expected {
		if blockExists[block] == nil || len(blockExists[block]) < 2 {
			t.Fatalf("Expected to find two replicas for block %s; found %d", block, len(blockExists[block]))
		}
	}
}

func valueInArray(value string, list []string) bool {
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
func backdateBlocks(t *testing.T, oldUnusedBlockLocators []string) {
	// First get rid of any size hints in the locators
	var trimmedBlockLocators []string
	for _, block := range oldUnusedBlockLocators {
		trimmedBlockLocators = append(trimmedBlockLocators, strings.Split(block, "+")[0])
	}

	// Get the working dir so that we can read keep{n}.volume files
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working dir %s", err)
	}

	// Now cycle through the two keep volumes
	oldTime := time.Now().AddDate(0, -2, 0)
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
				if valueInArray(blockName, trimmedBlockLocators) {
					err = os.Chtimes(myname, oldTime, oldTime)
				}
			}
		}
	}
}

func getStatus(t *testing.T, path string) interface{} {
	client := http.Client{}
	req, err := http.NewRequest("GET", path, nil)
	req.Header.Add("Authorization", "OAuth2 " + AdminToken)
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

func waitUntilQueuesFinishWork(t *testing.T) {
	// Wait until PullQueue and TrashQueue finish their work
	for {
		var done [2]bool
		for i := 0; i < 2; i++ {
			s := getStatus(t, keepServers[i]+"/status.json")
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
			time.Sleep(100 * time.Millisecond)
		}
	}
}

/*
Create some blocks and backdate some of them.
Also create some collections and delete some of them.
Verify block indexes.
*/
func TestPutAndGetBlocks(t *testing.T) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	// Put some blocks which will be backdated later on
	// The first one will also be used in a collection and hence should not be deleted when datamanager runs.
	// The rest will be old and unreferenced and hence should be deleted when datamanager runs.
	var oldUnusedBlockLocators []string
	oldUnusedBlockData := "this block will have older mtime"
	for i := 0; i < 5; i++ {
		oldUnusedBlockLocators = append(oldUnusedBlockLocators, putBlock(t, fmt.Sprintf("%s%d", oldUnusedBlockData, i)))
	}
	for i := 0; i < 5; i++ {
		getBlock(t, oldUnusedBlockLocators[i], fmt.Sprintf("%s%d", oldUnusedBlockData, i))
	}

	// The rest will be old and unreferenced and hence should be deleted when datamanager runs.
	oldUsedBlockData := "this collection block will have older mtime"
	oldUsedBlockLocator := putBlock(t, oldUsedBlockData)
	getBlock(t, oldUsedBlockLocator, oldUsedBlockData)

	// Put some more blocks which will not be backdated; hence they are still new, but not in any collection.
	// Hence, even though unreferenced, these should not be deleted when datamanager runs.
	var newBlockLocators []string
	newBlockData := "this block is newer"
	for i := 0; i < 5; i++ {
		newBlockLocators = append(newBlockLocators, putBlock(t, fmt.Sprintf("%s%d", newBlockData, i)))
	}
	for i := 0; i < 5; i++ {
		getBlock(t, newBlockLocators[i], fmt.Sprintf("%s%d", newBlockData, i))
	}

	// Create a collection that would be deleted later on
	toBeDeletedCollectionUuid := createCollection(t, "some data for collection creation")
	toBeDeletedCollectionLocator := getFirstLocatorFromCollection(t, toBeDeletedCollectionUuid)

	// Create another collection that has the same data as the one of the old blocks
	oldUsedBlockCollectionUuid := createCollection(t, oldUsedBlockData)
	oldUsedBlockCollectionLocator := getFirstLocatorFromCollection(t, oldUsedBlockCollectionUuid)
	if oldUsedBlockCollectionLocator != oldUsedBlockLocator {
		t.Fatalf("Locator of the collection with the same data as old block is different %s", oldUsedBlockCollectionLocator)
	}

	// Create another collection whose replication level will be changed
	replicationCollectionUuid := createCollection(t, "replication level on this collection will be reduced")
	replicationCollectionLocator := getFirstLocatorFromCollection(t, replicationCollectionUuid)

	// Create two collections with same data; one will be deleted later on
	dataForTwoCollections := "one of these collections will be deleted"
	oneOfTwoWithSameDataUuid := createCollection(t, dataForTwoCollections)
	oneOfTwoWithSameDataLocator := getFirstLocatorFromCollection(t, oneOfTwoWithSameDataUuid)
	secondOfTwoWithSameDataUuid := createCollection(t, dataForTwoCollections)
	secondOfTwoWithSameDataLocator := getFirstLocatorFromCollection(t, secondOfTwoWithSameDataUuid)
	if oneOfTwoWithSameDataLocator != secondOfTwoWithSameDataLocator {
		t.Fatalf("Locators for both these collections expected to be same: %s %s", oneOfTwoWithSameDataLocator, secondOfTwoWithSameDataLocator)
	}

	// Verify blocks before doing any backdating / deleting.
	var expected []string
	expected = append(expected, oldUnusedBlockLocators...)
	expected = append(expected, newBlockLocators...)
	expected = append(expected, toBeDeletedCollectionLocator)
	expected = append(expected, replicationCollectionLocator)
	expected = append(expected, oneOfTwoWithSameDataLocator)
	expected = append(expected, secondOfTwoWithSameDataLocator)

	verifyBlocks(t, nil, expected)

	// Run datamanager in singlerun mode
	dataManagerSingleRun(t)
	waitUntilQueuesFinishWork(t)

	verifyBlocks(t, nil, expected)

	// Backdate the to-be old blocks and delete the collections
	backdateBlocks(t, oldUnusedBlockLocators)
	deleteCollection(t, toBeDeletedCollectionUuid)
	deleteCollection(t, secondOfTwoWithSameDataUuid)

	// Run data manager again
	dataManagerSingleRun(t)
	waitUntilQueuesFinishWork(t)

	// Get block indexes and verify that all backdated blocks except the first one used in collection are not included.
	expected = expected[:0]
	expected = append(expected, oldUsedBlockLocator)
	expected = append(expected, newBlockLocators...)
	expected = append(expected, toBeDeletedCollectionLocator)
	expected = append(expected, replicationCollectionLocator)
	expected = append(expected, oneOfTwoWithSameDataLocator)
	expected = append(expected, secondOfTwoWithSameDataLocator)

	verifyBlocks(t, oldUnusedBlockLocators, expected)

	// Reduce replication on replicationCollectionUuid collection and verify that the overreplicated blocks are untouched.

	// Default replication level is 2; first verify that the replicationCollectionLocator appears in both volumes
	for i := 0; i < len(keepServers); i++ {
		indexes := getBlockIndexesForServer(t, i)
		if !valueInArray(replicationCollectionLocator, indexes) {
			t.Fatalf("Not found block in index %s", replicationCollectionLocator)
		}
	}

	// Now reduce replication level on this collection and verify that it still appears in both volumes
	updateCollection(t, replicationCollectionUuid, "replication_desired", "1")
	collection := getCollection(t, replicationCollectionUuid)
	if collection["replication_desired"].(interface{}) != float64(1) {
		t.Fatalf("After update replication_desired is not 1; instead it is %v", collection["replication_desired"])
	}

	// Run data manager again
	time.Sleep(100 * time.Millisecond)
	dataManagerSingleRun(t)
	waitUntilQueuesFinishWork(t)

	for i := 0; i < len(keepServers); i++ {
		indexes := getBlockIndexesForServer(t, i)
		if !valueInArray(replicationCollectionLocator, indexes) {
			t.Fatalf("Not found block in index %s", replicationCollectionLocator)
		}
	}
	// Done testing reduce replication on collection

	// Verify blocks one more time
	verifyBlocks(t, oldUnusedBlockLocators, expected)
}

func TestDatamanagerSingleRunRepeatedly(t *testing.T) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	for i := 0; i < 10; i++ {
		err := singlerun(arv)
		if err != nil {
			t.Fatalf("Got an error during datamanager singlerun: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestGetStatusRepeatedly(t *testing.T) {
	t.Skip("This test still fails. Skip it until it is fixed.")

	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	for i := 0; i < 10; i++ {
		for j := 0; j < 2; j++ {
			s := getStatus(t, keepServers[j]+"/status.json")

			var pullQueueStatus interface{}
			pullQueueStatus = s.(map[string]interface{})["PullQueue"]
			var trashQueueStatus interface{}
			trashQueueStatus = s.(map[string]interface{})["TrashQueue"]

			if pullQueueStatus.(map[string]interface{})["Queued"] == nil ||
				pullQueueStatus.(map[string]interface{})["InProgress"] == nil ||
				trashQueueStatus.(map[string]interface{})["Queued"] == nil ||
				trashQueueStatus.(map[string]interface{})["InProgress"] == nil {
				t.Fatalf("PullQueue and TrashQueue status not found")
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestRunDatamanagerWithBogusServer(t *testing.T) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	arv.ApiServer = "bogus-server"

	err := singlerun(arv)
	if err == nil {
		t.Fatalf("Expected error during singlerun with bogus server")
	}
}

func TestRunDatamanagerAsNonAdminUser(t *testing.T) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	arv.ApiToken = ActiveUserToken

	err := singlerun(arv)
	if err == nil {
		t.Fatalf("Expected error during singlerun as non-admin user")
	}
}
