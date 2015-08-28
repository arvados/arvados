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
	"time"
)

type PullWorkerTestSuite struct{}

// Gocheck boilerplate
func TestPullWorker(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&PullWorkerTestSuite{})

var testPullLists map[string]string
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
			"locator":"acbd18db4cc2f85cedef654fccc4a4d8+3",
			"servers":[
				"server_1",
				"server_2"
		 	]
		},{
			"locator":"37b51d194a7513e45b56f6524f2d51f2+3",
			"servers":[
				"server_3"
		 	]
		}
	]`)

var second_pull_list = []byte(`[
		{
			"locator":"73feffa4b7f6bb68e44cf984c85f6e88+3",
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

// In this case, the item will not be placed on pullq
func (s *PullWorkerTestSuite) TestPullWorker_invalid_data_manager_token(c *C) {
	defer teardown()

	data_manager_token = "DATA MANAGER TOKEN"

	testData := PullWorkerTestData{
		name:          "TestPullWorker_pull_list_with_two_locators",
		req:           RequestTester{"/pull", "invalid_data_manager_token", "PUT", first_pull_list},
		response_code: http.StatusUnauthorized,
		response_body: "Unauthorized\n",
		read_content:  "hello",
		read_error:    false,
		put_error:     false,
	}

	performTest(testData, c)
}

func performTest(testData PullWorkerTestData, c *C) {
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	RunTestPullWorker(c)
	defer pullq.Close()

	currentTestData = testData
	testPullLists[testData.name] = testData.response_body

	processedPullLists := make(map[string]string)

	// Override GetContent to mock keepclient Get functionality
	defer func(orig func(string, *keepclient.KeepClient) (io.ReadCloser, int64, string, error)) {
		GetContent = orig
	}(GetContent)
	GetContent = func(signedLocator string, keepClient *keepclient.KeepClient) (reader io.ReadCloser, contentLength int64, url string, err error) {
		c.Assert(getStatusItem("PullQueue", "InProgress"), Equals, float64(1))
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
	defer func(orig func([]byte, string) error) { PutContent = orig }(PutContent)
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

	c.Assert(getStatusItem("PullQueue", "InProgress"), Equals, float64(0))
	c.Assert(getStatusItem("PullQueue", "Queued"), Equals, float64(0))

	response := IssueRequest(&testData.req)
	c.Assert(response.Code, Equals, testData.response_code)
	c.Assert(response.Body.String(), Equals, testData.response_body)

	expectEqualWithin(c, time.Second, 0, func() interface{} {
		st := pullq.Status()
		return st.InProgress + st.Queued
	})

	if testData.name == "TestPullWorker_pull_list_with_two_items_latest_replacing_old" {
		c.Assert(len(testPullLists), Equals, 2)
		c.Assert(len(processedPullLists), Equals, 1)
		c.Assert(testPullLists["Added_before_actual_test_item"], NotNil)
		c.Assert(testPullLists["TestPullWorker_pull_list_with_two_items_latest_replacing_old"], NotNil)
		c.Assert(processedPullLists["TestPullWorker_pull_list_with_two_items_latest_replacing_old"], NotNil)
	} else {
		if testData.response_code == http.StatusOK {
			c.Assert(len(testPullLists), Equals, 1)
			c.Assert(len(processedPullLists), Equals, 1)
			c.Assert(testPullLists[testData.name], NotNil)
		} else {
			c.Assert(len(testPullLists), Equals, 1)
			c.Assert(len(processedPullLists), Equals, 0)
		}
	}

	if testData.read_error {
		c.Assert(readError, NotNil)
	} else if testData.response_code == http.StatusOK {
		c.Assert(readError, IsNil)
		c.Assert(readContent, Equals, testData.read_content)
		if testData.put_error {
			c.Assert(putError, NotNil)
		} else {
			c.Assert(putError, IsNil)
			c.Assert(string(putContent), Equals, testData.read_content)
		}
	}

	expectChannelEmpty(c, pullq.NextItem)
}

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) {
	return
}
