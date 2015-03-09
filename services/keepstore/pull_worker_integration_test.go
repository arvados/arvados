package main

import (
	"crypto/tls"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"net/http"
	"os"
	"strings"
	"testing"
	"encoding/json"
)

var keepClient keepclient.KeepClient

type PullWorkIntegrationTestData struct {
	Name     string
	Locator  string
	Content  string
	GetError string
}

func SetupPullWorkerIntegrationTest(t *testing.T, testData PullWorkIntegrationTestData, wantData bool) PullRequest {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	arvadostest.StartAPI()
	arvadostest.StartKeep()

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		t.Error("Error creating arv")
	}

	keepClient = keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 0,
		Using_proxy:   true,
		Client:        &http.Client{},
	}

	random_token := GenerateRandomApiToken()
	keepClient.Arvados.ApiToken = random_token
	if err != nil {
		t.Error("Error creating keepclient")
	}

	servers := GetKeepServices(t)

	pullRequest := PullRequest{
		Locator: testData.Locator,
		Servers: servers,
	}

	if wantData {
		service_roots := make(map[string]string)
		for _, addr := range pullRequest.Servers {
			service_roots[addr] = addr
		}
		keepClient.SetServiceRoots(service_roots)

		locator, _, err := keepClient.PutB([]byte(testData.Content))
		if err != nil {
			t.Errorf("Error putting test data in setup for %s %s %v", testData.Content, locator, err)
		}
	}

	return pullRequest
}

func GetKeepServices(t *testing.T) []string {
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/arvados/v1/keep_services", os.Getenv("ARVADOS_API_HOST")), nil)
	if err != nil {
		t.Errorf("Error getting keep services: ", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("OAuth2 %s", os.Getenv("ARVADOS_API_TOKEN")))

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Error getting keep services: ", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Error status code getting keep services", resp.StatusCode)
	}

	defer resp.Body.Close()
	var servers []string

	decoder := json.NewDecoder(resp.Body)

	var respJSON map[string]interface{}
	err = decoder.Decode(&respJSON)
	if err != nil {
		t.Errorf("Error decoding response for keep services: ", err)
	}

	var service_names []string
	var service_ports []string
	for _, v1 := range respJSON {
		switch v1_type := v1.(type) {
		case []interface{}:
			for _, v2 := range v1_type {
				switch v2_type := v2.(type) {
				case map[string]interface{}:
					for name, value := range v2_type {
						if name == "service_host" {
							service_names = append(service_names, fmt.Sprintf("%s", value))
						} else if name == "service_port" {
							service_ports = append(service_ports, strings.Split(fmt.Sprintf("%f", value), ".")[0])
						}
					}
				default:
				}
			}
		default:
		}
	}

	for i, port := range service_ports {
		servers = append(servers, "https://"+service_names[i]+":"+port)
	}

	return servers
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
	// Override PutContent to mock PutBlock functionality
	PutContent = func(content []byte, locator string) (err error) {
		return
	}

	err := PullItemAndProcess(pullRequest, keepClient.Arvados.ApiToken, keepClient)

	if len(testData.GetError) > 0 {
		if (err == nil) || (!strings.Contains(err.Error(), testData.GetError)) {
			t.Fail()
		}
	} else {
		t.Fail()
	}
}
