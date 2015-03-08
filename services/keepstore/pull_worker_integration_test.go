package main

import (
	"crypto/tls"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"net/http"
	"os"
	"strings"
	"testing"
)

var keepClient keepclient.KeepClient

type PullWorkIntegrationTestData struct {
	Name     string
	Locator  string
	Content  string
	GetError string
}

func SetupPullWorkerIntegrationTest(t *testing.T, testData PullWorkIntegrationTestData, wantData bool) PullRequest {
	arvadostest.StartAPI()
	arvadostest.StartKeep()

	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		t.Error("Error creating arv")
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	keepClient = keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 1,
		Using_proxy:   true,
		Client:        client,
	}

	random_token := GenerateRandomApiToken()
	keepClient.Arvados.ApiToken = random_token

	if err != nil {
		t.Error("Error creating keepclient")
	}

	servers := make([]string, 1)
	servers[0] = "https://" + os.Getenv("ARVADOS_API_HOST")
	pullRequest := PullRequest{
		Locator: testData.Locator,
		Servers: servers,
	}

	service_roots := make(map[string]string)
	for _, addr := range pullRequest.Servers {
		service_roots[addr] = addr
	}
	keepClient.SetServiceRoots(service_roots)

	if wantData {
		locator, _, err := keepClient.PutB([]byte(testData.Content))
		if err != nil {
			t.Errorf("Error putting test data in setup for %s %s", testData.Content, locator)
		}
	}
	return pullRequest
}

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

func performPullWorkerIntegrationTest(testData PullWorkIntegrationTestData, pullRequest PullRequest, t *testing.T) {
	err := PullItemAndProcess(pullRequest, keepClient.Arvados.ApiToken, keepClient)

	if len(testData.GetError) > 0 {
		if (err == nil) || (!strings.Contains(err.Error(), testData.GetError)) {
			t.Fail()
		}
	} else {
		t.Fail()
	}

	// Override PutContent to mock PutBlock functionality
	PutContent = func(content []byte, locator string) (err error) {
		// do nothing
		return
	}
}
