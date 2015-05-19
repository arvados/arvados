package main

import (
	"bytes"
	"errors"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

var keepClient *keepclient.KeepClient

type PullWorkIntegrationTestData struct {
	Name     string
	Locator  string
	Content  string
	GetError string
}

func SetupPullWorkerIntegrationTest(t *testing.T, testData PullWorkIntegrationTestData, wantData bool) PullRequest {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	// start api and keep servers
	arvadostest.StartAPI()
	arvadostest.StartKeep()

	// make arvadosclient
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		t.Error("Error creating arv")
	}

	// keep client
	keepClient = &keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 1,
		Using_proxy:   true,
		Client:        &http.Client{},
	}

	// discover keep services
	var servers []string
	if err := keepClient.DiscoverKeepServers(); err != nil {
		t.Error("Error discovering keep services")
	}
	for _, host := range keepClient.LocalRoots() {
		servers = append(servers, host)
	}

	// Put content if the test needs it
	if wantData {
		locator, _, err := keepClient.PutB([]byte(testData.Content))
		if err != nil {
			t.Errorf("Error putting test data in setup for %s %s %v", testData.Content, locator, err)
		}
		if locator == "" {
			t.Errorf("No locator found after putting test data")
		}
	}

	// Create pullRequest for the test
	pullRequest := PullRequest{
		Locator: testData.Locator,
		Servers: servers,
	}
	return pullRequest
}

// Do a get on a block that is not existing in any of the keep servers.
// Expect "block not found" error.
func TestPullWorkerIntegration_GetNonExistingLocator(t *testing.T) {
	testData := PullWorkIntegrationTestData{
		Name:     "TestPullWorkerIntegration_GetLocator",
		Locator:  "5d41402abc4b2a76b9719d911017c592",
		Content:  "hello",
		GetError: "Block not found",
	}

	pullRequest := SetupPullWorkerIntegrationTest(t, testData, false)

	performPullWorkerIntegrationTest(testData, pullRequest, t)
}

// Do a get on a block that exists on one of the keep servers.
// The setup method will create this block before doing the get.
func TestPullWorkerIntegration_GetExistingLocator(t *testing.T) {
	testData := PullWorkIntegrationTestData{
		Name:     "TestPullWorkerIntegration_GetLocator",
		Locator:  "5d41402abc4b2a76b9719d911017c592",
		Content:  "hello",
		GetError: "",
	}

	pullRequest := SetupPullWorkerIntegrationTest(t, testData, true)

	performPullWorkerIntegrationTest(testData, pullRequest, t)
}

// Perform the test.
// The test directly invokes the "PullItemAndProcess" rather than
// putting an item on the pullq so that the errors can be verified.
func performPullWorkerIntegrationTest(testData PullWorkIntegrationTestData, pullRequest PullRequest, t *testing.T) {

	// Override PutContent to mock PutBlock functionality
	defer func(orig func([]byte, string)(error)) { PutContent = orig }(PutContent)
	PutContent = func(content []byte, locator string) (err error) {
		if string(content) != testData.Content {
			t.Errorf("PutContent invoked with unexpected data. Expected: %s; Found: %s", testData.Content, content)
		}
		return
	}

	// Override GetContent to mock keepclient Get functionality
	defer func(orig func(string, *keepclient.KeepClient)(io.ReadCloser, int64, string, error)) { GetContent = orig }(GetContent)
	GetContent = func(signedLocator string, keepClient *keepclient.KeepClient) (
		reader io.ReadCloser, contentLength int64, url string, err error) {
		if testData.GetError != "" {
			return nil, 0, "", errors.New(testData.GetError)
		}
		rdr := &ClosingBuffer{bytes.NewBufferString(testData.Content)}
		return rdr, int64(len(testData.Content)), "", nil
	}

	keepClient.Arvados.ApiToken = GenerateRandomApiToken()
	err := PullItemAndProcess(pullRequest, keepClient.Arvados.ApiToken, keepClient)

	if len(testData.GetError) > 0 {
		if (err == nil) || (!strings.Contains(err.Error(), testData.GetError)) {
			t.Errorf("Got error %v, expected %v", err, testData.GetError)
		}
	} else {
		if err != nil {
			t.Errorf("Got error %v, expected nil", err)
		}
	}
}
