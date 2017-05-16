package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	. "gopkg.in/check.v1"
)

var _ = Suite(&PullWorkerTestSuite{})

type PullWorkerTestSuite struct {
	testPullLists map[string]string
	readContent   string
	readError     error
	putContent    []byte
	putError      error
}

func (s *PullWorkerTestSuite) SetUpTest(c *C) {
	theConfig.systemAuthToken = "arbitrary data manager token"
	s.readContent = ""
	s.readError = nil
	s.putContent = []byte{}
	s.putError = nil

	// When a new pull request arrives, the old one will be overwritten.
	// This behavior is verified using these two maps in the
	// "TestPullWorkerPullList_with_two_items_latest_replacing_old"
	s.testPullLists = make(map[string]string)

	KeepVM = MakeTestVolumeManager(2)

	// Normally the pull queue and workers are started by main()
	// -- tests need to set up their own.
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)
	keepClient, err := keepclient.MakeKeepClient(arv)
	c.Assert(err, IsNil)
	pullq = NewWorkQueue()
	go RunPullWorker(pullq, keepClient)
}

func (s *PullWorkerTestSuite) TearDownTest(c *C) {
	KeepVM.Close()
	KeepVM = nil
	pullq.Close()
	pullq = nil
	teardown()
	theConfig = DefaultConfig()
	theConfig.Start()
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

// Ensure MountUUID in a pull list is correctly translated to a Volume
// argument passed to writePulledBlock().
func (s *PullWorkerTestSuite) TestSpecifyMountUUID(c *C) {
	defer func(f func(Volume, []byte, string)) {
		writePulledBlock = f
	}(writePulledBlock)

	for _, spec := range []struct {
		sendUUID     string
		expectVolume Volume
	}{
		{
			sendUUID:     "",
			expectVolume: nil,
		},
		{
			sendUUID:     KeepVM.Mounts()[0].UUID,
			expectVolume: KeepVM.Mounts()[0].volume,
		},
	} {
		writePulledBlock = func(v Volume, _ []byte, _ string) {
			c.Check(v, Equals, spec.expectVolume)
		}

		resp := IssueRequest(&RequestTester{
			uri:      "/pull",
			apiToken: theConfig.systemAuthToken,
			method:   "PUT",
			requestBody: []byte(`[{
				"locator":"acbd18db4cc2f85cedef654fccc4a4d8+3",
				"servers":["server_1","server_2"],
				"mountuuid":"` + spec.sendUUID + `"}]`),
		})
		c.Assert(resp.Code, Equals, http.StatusOK)
		expectEqualWithin(c, time.Second, 0, func() interface{} {
			st := pullq.Status()
			return st.InProgress + st.Queued
		})
	}
}

func (s *PullWorkerTestSuite) TestPullWorkerPullList_with_two_locators(c *C) {
	testData := PullWorkerTestData{
		name:         "TestPullWorkerPullList_with_two_locators",
		req:          RequestTester{"/pull", theConfig.systemAuthToken, "PUT", firstPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 2 pull requests\n",
		readContent:  "hello",
		readError:    false,
		putError:     false,
	}

	s.performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorkerPullList_with_one_locator(c *C) {
	testData := PullWorkerTestData{
		name:         "TestPullWorkerPullList_with_one_locator",
		req:          RequestTester{"/pull", theConfig.systemAuthToken, "PUT", secondPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 1 pull requests\n",
		readContent:  "hola",
		readError:    false,
		putError:     false,
	}

	s.performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_get_one_locator(c *C) {
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_get_one_locator",
		req:          RequestTester{"/pull", theConfig.systemAuthToken, "PUT", secondPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 1 pull requests\n",
		readContent:  "unused",
		readError:    true,
		putError:     false,
	}

	s.performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_get_two_locators(c *C) {
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_get_two_locators",
		req:          RequestTester{"/pull", theConfig.systemAuthToken, "PUT", firstPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 2 pull requests\n",
		readContent:  "unused",
		readError:    true,
		putError:     false,
	}

	s.performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_put_one_locator(c *C) {
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_put_one_locator",
		req:          RequestTester{"/pull", theConfig.systemAuthToken, "PUT", secondPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 1 pull requests\n",
		readContent:  "hello hello",
		readError:    false,
		putError:     true,
	}

	s.performTest(testData, c)
}

func (s *PullWorkerTestSuite) TestPullWorker_error_on_put_two_locators(c *C) {
	testData := PullWorkerTestData{
		name:         "TestPullWorker_error_on_put_two_locators",
		req:          RequestTester{"/pull", theConfig.systemAuthToken, "PUT", firstPullList},
		responseCode: http.StatusOK,
		responseBody: "Received 2 pull requests\n",
		readContent:  "hello again",
		readError:    false,
		putError:     true,
	}

	s.performTest(testData, c)
}

// In this case, the item will not be placed on pullq
func (s *PullWorkerTestSuite) TestPullWorker_invalidToken(c *C) {
	testData := PullWorkerTestData{
		name:         "TestPullWorkerPullList_with_two_locators",
		req:          RequestTester{"/pull", "invalidToken", "PUT", firstPullList},
		responseCode: http.StatusUnauthorized,
		responseBody: "Unauthorized\n",
		readContent:  "hello",
		readError:    false,
		putError:     false,
	}

	s.performTest(testData, c)
}

func (s *PullWorkerTestSuite) performTest(testData PullWorkerTestData, c *C) {
	s.testPullLists[testData.name] = testData.responseBody

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
			s.readError = err
			return
		}
		s.readContent = testData.readContent
		reader = ioutil.NopCloser(bytes.NewBufferString(testData.readContent))
		contentLength = int64(len(testData.readContent))
		return
	}

	// Override writePulledBlock to mock PutBlock functionality
	defer func(orig func(Volume, []byte, string)) { writePulledBlock = orig }(writePulledBlock)
	writePulledBlock = func(v Volume, content []byte, locator string) {
		if testData.putError {
			s.putError = errors.New("Error putting data")
			return
		}
		s.putContent = content
	}

	c.Check(getStatusItem("PullQueue", "InProgress"), Equals, float64(0))
	c.Check(getStatusItem("PullQueue", "Queued"), Equals, float64(0))

	response := IssueRequest(&testData.req)
	c.Assert(response.Code, Equals, testData.responseCode)
	c.Assert(response.Body.String(), Equals, testData.responseBody)

	expectEqualWithin(c, time.Second, 0, func() interface{} {
		st := pullq.Status()
		return st.InProgress + st.Queued
	})

	if testData.name == "TestPullWorkerPullList_with_two_items_latest_replacing_old" {
		c.Assert(len(s.testPullLists), Equals, 2)
		c.Assert(len(processedPullLists), Equals, 1)
		c.Assert(s.testPullLists["Added_before_actual_test_item"], NotNil)
		c.Assert(s.testPullLists["TestPullWorkerPullList_with_two_items_latest_replacing_old"], NotNil)
		c.Assert(processedPullLists["TestPullWorkerPullList_with_two_items_latest_replacing_old"], NotNil)
	} else {
		if testData.responseCode == http.StatusOK {
			c.Assert(len(s.testPullLists), Equals, 1)
			c.Assert(len(processedPullLists), Equals, 1)
			c.Assert(s.testPullLists[testData.name], NotNil)
		} else {
			c.Assert(len(s.testPullLists), Equals, 1)
			c.Assert(len(processedPullLists), Equals, 0)
		}
	}

	if testData.readError {
		c.Assert(s.readError, NotNil)
	} else if testData.responseCode == http.StatusOK {
		c.Assert(s.readError, IsNil)
		c.Assert(s.readContent, Equals, testData.readContent)
		if testData.putError {
			c.Assert(s.putError, NotNil)
		} else {
			c.Assert(s.putError, IsNil)
			c.Assert(string(s.putContent), Equals, testData.readContent)
		}
	}

	expectChannelEmpty(c, pullq.NextItem)
}
