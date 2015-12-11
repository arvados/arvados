package main

import (
	"encoding/json"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/summary"
	"io/ioutil"
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
	arvadostest.StartKeep(2, false)

	var err error
	arv, err = arvadosclient.MakeArvadosClient()
	if err != nil {
		t.Fatalf("Error making arvados client: %s", err)
	}
	arv.ApiToken = arvadostest.DataManagerToken

	// keep client
	keepClient = &keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 2,
		Client:        &http.Client{},
	}

	// discover keep services
	if err = keepClient.DiscoverKeepServers(); err != nil {
		t.Fatalf("Error discovering keep services: %s", err)
	}
	keepServers = []string{}
	for _, host := range keepClient.LocalRoots() {
		keepServers = append(keepServers, host)
	}
}

func TearDownDataManagerTest(t *testing.T) {
	arvadostest.StopKeep(2)
	arvadostest.StopAPI()
	summary.WriteDataTo = ""
	collection.HeapProfileFilename = ""
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

func switchToken(t string) func() {
	orig := arv.ApiToken
	restore := func() {
		arv.ApiToken = orig
	}
	arv.ApiToken = t
	return restore
}

func getCollection(t *testing.T, uuid string) Dict {
	defer switchToken(arvadostest.AdminToken)()

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
	defer switchToken(arvadostest.AdminToken)()

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
	defer switchToken(arvadostest.AdminToken)()

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
	req.Header.Add("Authorization", "OAuth2 "+arvadostest.DataManagerToken)
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

func verifyBlocks(t *testing.T, notExpected []string, expected []string, minReplication int) {
	blocks := getBlockIndexes(t)

	for _, block := range notExpected {
		for _, idx := range blocks {
			if valueInArray(block, idx) {
				t.Fatalf("Found unexpected block %s", block)
			}
		}
	}

	for _, block := range expected {
		nFound := 0
		for _, idx := range blocks {
			if valueInArray(block, idx) {
				nFound++
			}
		}
		if nFound < minReplication {
			t.Fatalf("Found %d replicas of block %s, expected >= %d", nFound, block, minReplication)
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

// Test env uses two keep volumes. The volume names can be found by reading the files
// ARVADOS_HOME/tmp/keep0.volume and ARVADOS_HOME/tmp/keep1.volume
//
// The keep volumes are of the dir structure: volumeN/subdir/locator
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
	req.Header.Add("Authorization", "OAuth2 "+arvadostest.DataManagerToken)
	req.Header.Add("Content-Type", "application/octet-stream")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error during %s %s", path, err)
	}
	defer resp.Body.Close()

	var s interface{}
	json.NewDecoder(resp.Body).Decode(&s)

	return s
}

// Wait until PullQueue and TrashQueue are empty on all keepServers.
func waitUntilQueuesFinishWork(t *testing.T) {
	for _, ks := range keepServers {
		for done := false; !done; {
			time.Sleep(100 * time.Millisecond)
			s := getStatus(t, ks+"/status.json")
			for _, qName := range []string{"PullQueue", "TrashQueue"} {
				qStatus := s.(map[string]interface{})[qName].(map[string]interface{})
				if qStatus["Queued"].(float64)+qStatus["InProgress"].(float64) == 0 {
					done = true
				}
			}
		}
	}
}

// Create some blocks and backdate some of them.
// Also create some collections and delete some of them.
// Verify block indexes.
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
	toBeDeletedCollectionUUID := createCollection(t, "some data for collection creation")
	toBeDeletedCollectionLocator := getFirstLocatorFromCollection(t, toBeDeletedCollectionUUID)

	// Create another collection that has the same data as the one of the old blocks
	oldUsedBlockCollectionUUID := createCollection(t, oldUsedBlockData)
	oldUsedBlockCollectionLocator := getFirstLocatorFromCollection(t, oldUsedBlockCollectionUUID)
	if oldUsedBlockCollectionLocator != oldUsedBlockLocator {
		t.Fatalf("Locator of the collection with the same data as old block is different %s", oldUsedBlockCollectionLocator)
	}

	// Create another collection whose replication level will be changed
	replicationCollectionUUID := createCollection(t, "replication level on this collection will be reduced")
	replicationCollectionLocator := getFirstLocatorFromCollection(t, replicationCollectionUUID)

	// Create two collections with same data; one will be deleted later on
	dataForTwoCollections := "one of these collections will be deleted"
	oneOfTwoWithSameDataUUID := createCollection(t, dataForTwoCollections)
	oneOfTwoWithSameDataLocator := getFirstLocatorFromCollection(t, oneOfTwoWithSameDataUUID)
	secondOfTwoWithSameDataUUID := createCollection(t, dataForTwoCollections)
	secondOfTwoWithSameDataLocator := getFirstLocatorFromCollection(t, secondOfTwoWithSameDataUUID)
	if oneOfTwoWithSameDataLocator != secondOfTwoWithSameDataLocator {
		t.Fatalf("Locators for both these collections expected to be same: %s %s", oneOfTwoWithSameDataLocator, secondOfTwoWithSameDataLocator)
	}

	// create collection with empty manifest text
	emptyBlockLocator := putBlock(t, "")
	emptyCollection := createCollection(t, "")

	// Verify blocks before doing any backdating / deleting.
	var expected []string
	expected = append(expected, oldUnusedBlockLocators...)
	expected = append(expected, newBlockLocators...)
	expected = append(expected, toBeDeletedCollectionLocator)
	expected = append(expected, replicationCollectionLocator)
	expected = append(expected, oneOfTwoWithSameDataLocator)
	expected = append(expected, secondOfTwoWithSameDataLocator)
	expected = append(expected, emptyBlockLocator)

	verifyBlocks(t, nil, expected, 2)

	// Run datamanager in singlerun mode
	dataManagerSingleRun(t)
	waitUntilQueuesFinishWork(t)

	verifyBlocks(t, nil, expected, 2)

	// Backdate the to-be old blocks and delete the collections
	backdateBlocks(t, oldUnusedBlockLocators)
	deleteCollection(t, toBeDeletedCollectionUUID)
	deleteCollection(t, secondOfTwoWithSameDataUUID)
	backdateBlocks(t, []string{emptyBlockLocator})
	deleteCollection(t, emptyCollection)

	// Run data manager again
	dataManagerSingleRun(t)
	waitUntilQueuesFinishWork(t)

	// Get block indexes and verify that all backdated blocks except the first one used in collection are not included.
	expected = expected[:0]
	expected = append(expected, oldUsedBlockLocator)
	expected = append(expected, newBlockLocators...)
	expected = append(expected, toBeDeletedCollectionLocator)
	expected = append(expected, oneOfTwoWithSameDataLocator)
	expected = append(expected, secondOfTwoWithSameDataLocator)
	expected = append(expected, emptyBlockLocator) // even when unreferenced, this remains

	verifyBlocks(t, oldUnusedBlockLocators, expected, 2)

	// Reduce desired replication on replicationCollectionUUID
	// collection, and verify that Data Manager does not reduce
	// actual replication any further than that. (It might not
	// reduce actual replication at all; that's OK for this test.)

	// Reduce desired replication level.
	updateCollection(t, replicationCollectionUUID, "replication_desired", "1")
	collection := getCollection(t, replicationCollectionUUID)
	if collection["replication_desired"].(interface{}) != float64(1) {
		t.Fatalf("After update replication_desired is not 1; instead it is %v", collection["replication_desired"])
	}

	// Verify data is currently overreplicated.
	verifyBlocks(t, nil, []string{replicationCollectionLocator}, 2)

	// Run data manager again
	dataManagerSingleRun(t)
	waitUntilQueuesFinishWork(t)

	// Verify data is not underreplicated.
	verifyBlocks(t, nil, []string{replicationCollectionLocator}, 1)

	// Verify *other* collections' data is not underreplicated.
	verifyBlocks(t, oldUnusedBlockLocators, expected, 2)
}

func TestDatamanagerSingleRunRepeatedly(t *testing.T) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	for i := 0; i < 10; i++ {
		err := singlerun(arv)
		if err != nil {
			t.Fatalf("Got an error during datamanager singlerun: %v", err)
		}
	}
}

func TestGetStatusRepeatedly(t *testing.T) {
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

	arv.ApiToken = arvadostest.ActiveToken

	err := singlerun(arv)
	if err == nil {
		t.Fatalf("Expected error during singlerun as non-admin user")
	}
}

func TestPutAndGetBlocks_NoErrorDuringSingleRun(t *testing.T) {
	testOldBlocksNotDeletedOnDataManagerError(t, "", "", false, false)
}

func TestPutAndGetBlocks_ErrorDuringGetCollectionsBadWriteTo(t *testing.T) {
	testOldBlocksNotDeletedOnDataManagerError(t, "/badwritetofile", "", true, true)
}

func TestPutAndGetBlocks_ErrorDuringGetCollectionsBadHeapProfileFilename(t *testing.T) {
	testOldBlocksNotDeletedOnDataManagerError(t, "", "/badheapprofilefile", true, true)
}

// Create some blocks and backdate some of them.
// Run datamanager while producing an error condition.
// Verify that the blocks are hence not deleted.
func testOldBlocksNotDeletedOnDataManagerError(t *testing.T, writeDataTo string, heapProfileFile string, expectError bool, expectOldBlocks bool) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	// Put some blocks and backdate them.
	var oldUnusedBlockLocators []string
	oldUnusedBlockData := "this block will have older mtime"
	for i := 0; i < 5; i++ {
		oldUnusedBlockLocators = append(oldUnusedBlockLocators, putBlock(t, fmt.Sprintf("%s%d", oldUnusedBlockData, i)))
	}
	backdateBlocks(t, oldUnusedBlockLocators)

	// Run data manager
	summary.WriteDataTo = writeDataTo
	collection.HeapProfileFilename = heapProfileFile

	err := singlerun(arv)
	if !expectError {
		if err != nil {
			t.Fatalf("Got an error during datamanager singlerun: %v", err)
		}
	} else {
		if err == nil {
			t.Fatalf("Expected error during datamanager singlerun")
		}
	}
	waitUntilQueuesFinishWork(t)

	// Get block indexes and verify that all backdated blocks are not/deleted as expected
	if expectOldBlocks {
		verifyBlocks(t, nil, oldUnusedBlockLocators, 2)
	} else {
		verifyBlocks(t, oldUnusedBlockLocators, nil, 2)
	}
}

// Create a collection with multiple streams and blocks
func createMultiStreamBlockCollection(t *testing.T, data string, numStreams, numBlocks int) (string, []string) {
	defer switchToken(arvadostest.AdminToken)()

	manifest := ""
	locators := make(map[string]bool)
	for s := 0; s < numStreams; s++ {
		manifest += fmt.Sprintf("./stream%d ", s)
		for b := 0; b < numBlocks; b++ {
			locator, _, err := keepClient.PutB([]byte(fmt.Sprintf("%s in stream %d and block %d", data, s, b)))
			if err != nil {
				t.Fatalf("Error creating block %d in stream %d: %v", b, s, err)
			}
			locators[strings.Split(locator, "+A")[0]] = true
			manifest += locator + " "
		}
		manifest += "0:1:dummyfile.txt\n"
	}

	collection := make(Dict)
	err := arv.Create("collections",
		arvadosclient.Dict{"collection": arvadosclient.Dict{"manifest_text": manifest}},
		&collection)

	if err != nil {
		t.Fatalf("Error creating collection %v", err)
	}

	var locs []string
	for k := range locators {
		locs = append(locs, k)
	}

	return collection["uuid"].(string), locs
}

// Create collection with multiple streams and blocks; backdate the blocks and but do not delete the collection.
// Also, create stray block and backdate it.
// After datamanager run: expect blocks from the collection, but not the stray block.
func TestManifestWithMultipleStreamsAndBlocks(t *testing.T) {
	testManifestWithMultipleStreamsAndBlocks(t, 100, 10, "", false)
}

// Same test as TestManifestWithMultipleStreamsAndBlocks with an additional
// keepstore of a service type other than "disk". Only the "disk" type services
// will be indexed by datamanager and hence should work the same way.
func TestManifestWithMultipleStreamsAndBlocks_WithOneUnsupportedKeepServer(t *testing.T) {
	testManifestWithMultipleStreamsAndBlocks(t, 2, 2, "testblobstore", false)
}

// Test datamanager with dry-run. Expect no block to be deleted.
func TestManifestWithMultipleStreamsAndBlocks_DryRun(t *testing.T) {
	testManifestWithMultipleStreamsAndBlocks(t, 2, 2, "", true)
}

func testManifestWithMultipleStreamsAndBlocks(t *testing.T, numStreams, numBlocks int, createExtraKeepServerWithType string, isDryRun bool) {
	defer TearDownDataManagerTest(t)
	SetupDataManagerTest(t)

	// create collection whose blocks will be backdated
	collectionWithOldBlocks, oldBlocks := createMultiStreamBlockCollection(t, "old block", numStreams, numBlocks)
	if collectionWithOldBlocks == "" {
		t.Fatalf("Failed to create collection with %d blocks", numStreams*numBlocks)
	}
	if len(oldBlocks) != numStreams*numBlocks {
		t.Fatalf("Not all blocks are created: expected %v, found %v", 1000, len(oldBlocks))
	}

	// create a stray block that will be backdated
	strayOldBlock := putBlock(t, "this stray block is old")

	expected := []string{strayOldBlock}
	expected = append(expected, oldBlocks...)
	verifyBlocks(t, nil, expected, 2)

	// Backdate old blocks; but the collection still references these blocks
	backdateBlocks(t, oldBlocks)

	// also backdate the stray old block
	backdateBlocks(t, []string{strayOldBlock})

	// If requested, create an extra keepserver with the given type
	// This should be ignored during indexing and hence not change the datamanager outcome
	var extraKeepServerUUID string
	if createExtraKeepServerWithType != "" {
		extraKeepServerUUID = addExtraKeepServer(t, createExtraKeepServerWithType)
		defer deleteExtraKeepServer(extraKeepServerUUID)
	}

	// run datamanager
	dryRun = isDryRun
	dataManagerSingleRun(t)

	if dryRun {
		// verify that all blocks, including strayOldBlock, are still to be found
		verifyBlocks(t, nil, expected, 2)
	} else {
		// verify that strayOldBlock is not to be found, but the collections blocks are still there
		verifyBlocks(t, []string{strayOldBlock}, oldBlocks, 2)
	}
}

// Add one more keepstore with the given service type
func addExtraKeepServer(t *testing.T, serviceType string) string {
	defer switchToken(arvadostest.AdminToken)()

	extraKeepService := make(arvadosclient.Dict)
	err := arv.Create("keep_services",
		arvadosclient.Dict{"keep_service": arvadosclient.Dict{
			"service_host":     "localhost",
			"service_port":     "21321",
			"service_ssl_flag": false,
			"service_type":     serviceType}},
		&extraKeepService)
	if err != nil {
		t.Fatal(err)
	}

	return extraKeepService["uuid"].(string)
}

func deleteExtraKeepServer(uuid string) {
	defer switchToken(arvadostest.AdminToken)()
	arv.Delete("keep_services", uuid, nil, nil)
}
