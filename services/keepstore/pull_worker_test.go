package main

import (
	"bytes"
	"errors"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	. "gopkg.in/check.v1"
	"io"
	"net/http"
	"testing"
)

type PullWorkerTestSuite struct{}

// Gocheck boilerplate
func TestPullWorker(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&PullWorkerTestSuite{})

var testPullLists map[string]string
var processedPullLists map[string]string
var readContent string
var readError error
var putContent []byte
var putError error
var currentTestData PullWorkerTestData

func (s *PullWorkerTestSuite) SetUpTest(c *C) {
	readContent = ""
	readError = nil
	putContent = []byte("")
	putError = nil

	// When a new pull request arrives, the old one will be overwritten.
	// This behavior is verified using these two maps in the
	// "TestPullWorker_pull_list_with_two_items_latest_replacing_old"
	testPullLists = make(map[string]string)
	processedPullLists = make(map[string]string)
}

// Since keepstore does not come into picture in tests,
// we need to explicitly start the goroutine in tests.
func RunTestPullWorker(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)
	keepClient, err := keepclient.MakeKeepClient(&arv)
	c.Assert(err, Equals, nil)

	pullq = NewWorkQueue()
	go RunPullWorker(pullq, keepClient)
}

var first_pull_list = []byte(`[
		{
			"locator":"locator1",
			"servers":[
				"server_1",
				"server_2"
		 	]
		},
    {
			"locator":"locator2",
			"servers":[
				"server_3"
		 	]
		}
	]`)

var second_pull_list = []byte(`[
		{
			"locator":"locator3",
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

func (s *PullWorkerTestSuite) TestPullWorker_pull_list_with_two_locators(c *C) {
	defer teardown()

	data_manager_token = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:          "TestPullWorker_pull_list_with_two_locators",
		req:           RequestTester{"/pull", data_manager_token, "PUT", first_pull_list},
		response_code: http.StatusOK,
		response_body: "Received 2 pull requests\n",
		read_content:  "hello",
		read_error:    false,
		put_error:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_pull_list_with_one_locator(c *C) {
	defer teardown()

	data_manager_token = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:          "TestPullWorker_pull_list_with_one_locator",
		req:           RequestTester{"/pull", data_manager_token, "PUT", second_pull_list},
		response_code: http.StatusOK,
		response_body: "Received 1 pull requests\n",
		read_content:  "hola",
		read_error:    false,
		put_error:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_get_one_locator(c *C) {
	defer teardown()

	data_manager_token = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:          "TestPullWorker_error_on_get_one_locator",
		req:           RequestTester{"/pull", data_manager_token, "PUT", second_pull_list},
		response_code: http.StatusOK,
		response_body: "Received 1 pull requests\n",
		read_content:  "unused",
		read_error:    true,
		put_error:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_get_two_locators(c *C) {
	defer teardown()

	data_manager_token = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:          "TestPullWorker_error_on_get_two_locators",
		req:           RequestTester{"/pull", data_manager_token, "PUT", first_pull_list},
		response_code: http.StatusOK,
		response_body: "Received 2 pull requests\n",
		read_content:  "unused",
		read_error:    true,
		put_error:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_put_one_locator(c *C) {
	defer teardown()

	data_manager_token = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:          "TestPullWorker_error_on_put_one_locator",
		req:           RequestTester{"/pull", data_manager_token, "PUT", second_pull_list},
		response_code: http.StatusOK,
		response_body: "Received 1 pull requests\n",
		read_content:  "hello hello",
		read_error:    false,
		put_error:     true,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_put_two_locators(c *C) {
	defer teardown()

	data_manager_token = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:          "TestPullWorker_error_on_put_two_locators",
		req:           RequestTester{"/pull", data_manager_token, "PUT", first_pull_list},
		response_code: http.StatusOK,
		response_body: "Received 2 pull requests\n",
		read_content:  "hello again",
		read_error:    false,
		put_error:     true,
	}

	performTest(testData, c)
}

// When a new pull request arrives, the old one is replaced. This test
// is used to check that behavior by first putting an item on the queue,
// and then performing the test. Thus the "testPullLists" has two entries;
// however, processedPullLists will see only the newest item in the list.
func (s *PullWorkerTestSuite) TestPullWorker_pull_list_with_two_items_latest_replacing_old(c *C) {
	defer teardown()

	var firstInput = []int{1}
	pullq = NewWorkQueue()
	pullq.ReplaceQueue(makeTestWorkList(firstInput))
	testPullLists["Added_before_actual_test_item"] = string(1)

	data_manager_token = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:          "TestPullWorker_pull_list_with_two_items_latest_replacing_old",
		req:           RequestTester{"/pull", data_manager_token, "PUT", second_pull_list},
		response_code: http.StatusOK,
		response_body: "Received 1 pull requests\n",
		read_content:  "hola de nuevo",
		read_error:    false,
		put_error:     false,
	}

	performTest(testData, c)
}

func performTest(testData PullWorkerTestData, c *C) {
	RunTestPullWorker(c)

	currentTestData = testData
	testPullLists[testData.name] = testData.response_body

	// Override GetContent to mock keepclient Get functionality
	GetContent = func(signedLocator string, keepClient keepclient.KeepClient) (
		reader io.ReadCloser, contentLength int64, url string, err error) {

		processedPullLists[testData.name] = testData.response_body
		if testData.read_error {
			err = errors.New("Error getting data")
			readError = err
			return nil, 0, "", err
		} else {
			readContent = testData.read_content
			cb := &ClosingBuffer{bytes.NewBufferString(testData.read_content)}
			var rc io.ReadCloser
			rc = cb
			return rc, int64(len(testData.read_content)), "", nil
		}
	}

	// Override PutContent to mock PutBlock functionality
	PutContent = func(content []byte, locator string) (err error) {
		if testData.put_error {
			err = errors.New("Error putting data")
			putError = err
			return err
		} else {
			putContent = content
			return nil
		}
	}

	response := IssueRequest(&testData.req)
	c.Assert(testData.response_code, Equals, response.Code)
	c.Assert(testData.response_body, Equals, response.Body.String())

	expectWorkerChannelEmpty(c, pullq.NextItem)

	pullq.Close()

	if testData.name == "TestPullWorker_pull_list_with_two_items_latest_replacing_old" {
		c.Assert(len(testPullLists), Equals, 2)
		c.Assert(len(processedPullLists), Equals, 1)
		c.Assert(testPullLists["Added_before_actual_test_item"], NotNil)
		c.Assert(testPullLists["TestPullWorker_pull_list_with_two_items_latest_replacing_old"], NotNil)
		c.Assert(processedPullLists["TestPullWorker_pull_list_with_two_items_latest_replacing_old"], NotNil)
	} else {
		c.Assert(len(testPullLists), Equals, 1)
		c.Assert(len(processedPullLists), Equals, 1)
		c.Assert(testPullLists[testData.name], NotNil)
	}

	if testData.read_error {
		c.Assert(readError, NotNil)
	} else {
		c.Assert(readError, IsNil)
		c.Assert(readContent, Equals, testData.read_content)
		if testData.put_error {
			c.Assert(putError, NotNil)
		} else {
			c.Assert(putError, IsNil)
			c.Assert(string(putContent), Equals, testData.read_content)
		}
	}
}

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) {
	return
}

func expectWorkerChannelEmpty(c *C, workerChannel <-chan interface{}) {
	select {
	case item := <-workerChannel:
		c.Fatalf("Received value (%v) from channel that was expected to be empty", item)
	default:
	}
}

func expectWorkerChannelNotEmpty(c *C, workerChannel <-chan interface{}) {
	select {
	case item := <-workerChannel:
		c.Fatalf("Received value (%v) from channel that was expected to be empty", item)
	default:
	}
}
