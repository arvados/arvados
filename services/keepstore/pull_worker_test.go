package main

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestPullWorker(t *testing.T) {
	defer teardown()

	// Since keepstore does not come into picture in tests,
	// we need to explicitly start the goroutine in tests.
	go RunPullWorker(pullq.NextItem)

	data_manager_token = "DATA MANAGER TOKEN"

	first_pull_list := []byte(`[
		{
			"locator":"locator1_to_verify_first_pull_list",
			"servers":[
				"server_1",
				"server_2"
		 	]
		},
    {
			"locator":"locator2_to_verify_first_pull_list",
			"servers":[
				"server_1"
		 	]
		}
	]`)

	second_pull_list := []byte(`[
		{
			"locator":"locator_to_verify_second_pull_list",
			"servers":[
				"server_1",
        "server_2"
		 	]
		}
	]`)

	type PullWorkerTestData struct {
		name          string
		req           RequestTester
		response_code int
		response_body string
		read_content  string
		read_error    bool
		put_error     bool
	}
	var testcases = []PullWorkerTestData{
		{
			"Pull request 1 from the data manager in worker",
			RequestTester{"/pull", data_manager_token, "PUT", first_pull_list},
			http.StatusOK,
			"Received 2 pull requests\n",
			"hello",
			false,
			false,
		},
		{
			"Pull request 2 from the data manager in worker",
			RequestTester{"/pull", data_manager_token, "PUT", second_pull_list},
			http.StatusOK,
			"Received 1 pull requests\n",
			"hola",
			false,
			false,
		},
		{
			"Pull request with error on get",
			RequestTester{"/pull", data_manager_token, "PUT", second_pull_list},
			http.StatusOK,
			"Received 1 pull requests\n",
			"unused",
			true,
			false,
		},
		{
			"Pull request with error on put",
			RequestTester{"/pull", data_manager_token, "PUT", second_pull_list},
			http.StatusOK,
			"Received 1 pull requests\n",
			"unused",
			false,
			true,
		},
	}

	for _, testData := range testcases {
		// Override GetContent to mock keepclient functionality
		GetContent = func(addr string, locator string) ([]byte, error) {
			if testData.read_error {
				return nil, errors.New("Error getting data")
			} else {
				return []byte(testData.read_content), nil
			}
		}

		// Override PutContent to mock PutBlock functionality
		PutContent = func(content []byte, locator string) (err error) {
			if testData.put_error {
				return errors.New("Error putting data")
			} else {
				return nil
			}
		}

		response := IssueRequest(&testData.req)
		ExpectStatusCode(t, testData.name, testData.response_code, response)
		ExpectBody(t, testData.name, testData.response_body, response)

		// give the channel a second to read and process all pull list entries
		time.Sleep(1000 * time.Millisecond)

		expectChannelEmpty(t, pullq.NextItem)
	}
}
