package main

import (
	"crypto/tls"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"net/http"
	"os"
	"testing"
)

type PullWorkIntegrationTestData struct {
	Name    string
	Locator string
	Content string
}

func TestPullWorkerIntegration_GetLocator(t *testing.T) {
	arvadostest.StartAPI()
	arvadostest.StartKeep()

	testData := PullWorkIntegrationTestData{
		Name:    "TestPullWorkerIntegration_GetLocator",
		Locator: "5d41402abc4b2a76b9719d911017c592",
		Content: "hello",
	}

	performPullWorkerIntegrationTest(testData, t)
}

func performPullWorkerIntegrationTest(testData PullWorkIntegrationTestData, t *testing.T) {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	PermissionSecret = []byte("abc123")

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		t.Error("Error creating arv")
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	keepClient := keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 2,
		Using_proxy:   true,
		Client:        client,
	}

	random_token := GenerateRandomApiToken()
	keepClient.Arvados.ApiToken = random_token

	if err != nil {
		t.Error("Error creating keepclient")
	}

	pullq = NewWorkQueue()
	go RunPullWorker(pullq, keepClient)

	servers := make([]string, 1)
	servers[0] = "https://" + os.Getenv("ARVADOS_API_HOST") + "/arvados/v1/keep_services"
	pullRequest := PullRequest{
		Locator: testData.Locator,
		Servers: servers,
	}

	PullItemAndProcess(pullRequest, random_token, keepClient)

	// Override PutContent to mock PutBlock functionality
	PutContent = func(content []byte, locator string) (err error) {
		// do nothing
		return
	}

	pullq.Close()
	pullq = nil
}
