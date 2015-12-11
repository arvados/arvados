package main

import (
	"bytes"
	"errors"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	. "gopkg.in/check.v1"
	"io"
	"net/http"
	"time"
)

var _ = Suite(&PullWorkerTestSuite{})

type PullWorkerTestSuite struct{}

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
	// "TestPullWorkerPullList_with_two_items_latest_replacing_old"
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

var firstPullList = []byte(`[
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

var secondPullList = []byte(`[
		{
			"locator":"73feffa4b7f6bb68e44cf984c85f6e88+3",
			"servers":[
				"server_1",
				"server_2"
		 	]
		}
	]`)

type PullWorkerTestData struct {
	name         string
	req          RequestTester
	responseCode int
	responseBody string
	readContent  string
	readError    bool
	putError     bool
}

func (s *PullWorkerTestSuite) TestPullWorkerPullList_with_two_locators(c *C) {
	defer teardown()

	dataManagerToken = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:         "TestPullWorkerPullList_with_two_locators",
		req:          RequestTester{"/pull", dataManagerToken, "PUT", firstPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 2 pull requests\n",
		readContent:  "hello",
		readError:    false,
		putError:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorkerPullList_with_one_locator(c *C) {
	defer teardown()

	dataManagerToken = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:         "TestPullWorkerPullList_with_one_locator",
		req:          RequestTester{"/pull", dataManagerToken, "PUT", secondPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 1 pull requests\n",
		readContent:  "hola",
		readError:    false,
		putError:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_get_one_locator(c *C) {
	defer teardown()

	dataManagerToken = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_get_one_locator",
		req:          RequestTester{"/pull", dataManagerToken, "PUT", secondPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 1 pull requests\n",
		readContent:  "unused",
		readError:    true,
		putError:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_get_two_locators(c *C) {
	defer teardown()

	dataManagerToken = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_get_two_locators",
		req:          RequestTester{"/pull", dataManagerToken, "PUT", firstPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 2 pull requests\n",
		readContent:  "unused",
		readError:    true,
		putError:     false,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_put_one_locator(c *C) {
	defer teardown()

	dataManagerToken = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_put_one_locator",
		req:          RequestTester{"/pull", dataManagerToken, "PUT", secondPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 1 pull requests\n",
		readContent:  "hello hello",
		readError:    false,
		putError:     true,
	}

	performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_put_two_locators(c *C) {
	defer teardown()

	dataManagerToken = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_put_two_locators",
		req:          RequestTester{"/pull", dataManagerToken, "PUT", firstPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 2 pull requests\n",
		readContent:  "hello again",
		readError:    false,
		putError:     true,
	}

	performTest(testData, c)
}

// When a new pull request arrives, the old one is replaced. This test
// is used to check that behavior by first putting an item on the queue,
// and then performing the test. Thus the "testPullLists" has two entries;
// however, processedPullLists will see only the newest item in the list.
func (s *PullWorkerTestSuite) TestPullWorkerPullList_with_two_items_latest_replacing_old(c *C) {
	defer teardown()

	var firstInput = []int{1}
	pullq = NewWorkQueue()
	pullq.ReplaceQueue(makeTestWorkList(firstInput))
	testPullLists["Added_before_actual_test_item"] = string(1)

	dataManagerToken = "DATA MANAGER TOKEN"
	testData := PullWorkerTestData{
		name:         "TestPullWorkerPullList_with_two_items_latest_replacing_old",
		req:          RequestTester{"/pull", dataManagerToken, "PUT", secondPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 1 pull requests\n",
		readContent:  "hola de nuevo",
		readError:    false,
		putError:     false,
	}

	performTest(testData, c)
}

// In this case, the item will not be placed on pullq
func (s *PullWorkerTestSuite) TestPullWorker_invalid_dataManagerToken(c *C) {
	defer teardown()

	dataManagerToken = "DATA MANAGER TOKEN"

	testData := PullWorkerTestData{
		name:         "TestPullWorkerPullList_with_two_locators",
		req:          RequestTester{"/pull", "invalid_dataManagerToken", "PUT", firstPullList},
		responseCode: http.StatusUnauthorized,
		responseBody: "Unauthorized\n",
		readContent:  "hello",
		readError:    false,
		putError:     false,
	}

	performTest(testData, c)
}

func performTest(testData PullWorkerTestData, c *C) {
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	RunTestPullWorker(c)
	defer pullq.Close()

	currentTestData = testData
	testPullLists[testData.name] = testData.responseBody

	processedPullLists := make(map[string]string)

	// Override GetContent to mock keepclient Get functionality
	defer func(orig func(string, *keepclient.KeepClient) (io.ReadCloser, int64, string, error)) {
		GetContent = orig
	}(GetContent)
	GetContent = func(signedLocator string, keepClient *keepclient.KeepClient) (reader io.ReadCloser, contentLength int64, url string, err error) {
		c.Assert(getStatusItem("PullQueue", "InProgress"), Equals, float64(1))
		processedPullLists[testData.name] = testData.responseBody
		if testData.readError {
			err = errors.New("Error getting data")
			readError = err
			return nil, 0, "", err
		}
		readContent = testData.readContent
		cb := &ClosingBuffer{bytes.NewBufferString(testData.readContent)}
		var rc io.ReadCloser
		rc = cb
		return rc, int64(len(testData.readContent)), "", nil
	}

	// Override PutContent to mock PutBlock functionality
	defer func(orig func([]byte, string) error) { PutContent = orig }(PutContent)
	PutContent = func(content []byte, locator string) (err error) {
		if testData.putError {
			err = errors.New("Error putting data")
			putError = err
			return err
		}
		putContent = content
		return nil
	}

	c.Assert(getStatusItem("PullQueue", "InProgress"), Equals, float64(0))
	c.Assert(getStatusItem("PullQueue", "Queued"), Equals, float64(0))

	response := IssueRequest(&testData.req)
	c.Assert(response.Code, Equals, testData.responseCode)
	c.Assert(response.Body.String(), Equals, testData.responseBody)

	expectEqualWithin(c, time.Second, 0, func() interface{} {
		st := pullq.Status()
		return st.InProgress + st.Queued
	})

	if testData.name == "TestPullWorkerPullList_with_two_items_latest_replacing_old" {
		c.Assert(len(testPullLists), Equals, 2)
		c.Assert(len(processedPullLists), Equals, 1)
		c.Assert(testPullLists["Added_before_actual_test_item"], NotNil)
		c.Assert(testPullLists["TestPullWorkerPullList_with_two_items_latest_replacing_old"], NotNil)
		c.Assert(processedPullLists["TestPullWorkerPullList_with_two_items_latest_replacing_old"], NotNil)
	} else {
		if testData.responseCode == http.StatusOK {
			c.Assert(len(testPullLists), Equals, 1)
			c.Assert(len(processedPullLists), Equals, 1)
			c.Assert(testPullLists[testData.name], NotNil)
		} else {
			c.Assert(len(testPullLists), Equals, 1)
			c.Assert(len(processedPullLists), Equals, 0)
		}
	}

	if testData.readError {
		c.Assert(readError, NotNil)
	} else if testData.responseCode == http.StatusOK {
		c.Assert(readError, IsNil)
		c.Assert(readContent, Equals, testData.readContent)
		if testData.putError {
			c.Assert(putError, NotNil)
		} else {
			c.Assert(putError, IsNil)
			c.Assert(string(putContent), Equals, testData.readContent)
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
