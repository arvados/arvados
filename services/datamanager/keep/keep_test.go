package keep

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"

	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type KeepSuite struct{}

var _ = Suite(&KeepSuite{})

type TestHandler struct {
	request TrashList
}

func (ts *TestHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	r := json.NewDecoder(req.Body)
	r.Decode(&ts.request)
}

func (s *KeepSuite) TestSendTrashLists(c *C) {
	th := TestHandler{}
	server := httptest.NewServer(&th)
	defer server.Close()

	tl := map[string]TrashList{
		server.URL: TrashList{TrashRequest{"000000000000000000000000deadbeef", 99}}}

	arv := arvadosclient.ArvadosClient{ApiToken: "abc123"}
	kc := keepclient.KeepClient{Arvados: &arv, Client: &http.Client{}}
	kc.SetServiceRoots(map[string]string{"xxxx": server.URL},
		map[string]string{"xxxx": server.URL},
		map[string]string{})

	err := SendTrashLists(nil, &kc, tl, false)

	c.Check(err, IsNil)

	c.Check(th.request,
		DeepEquals,
		tl[server.URL])

}

type TestHandlerError struct {
}

func (tse *TestHandlerError) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	http.Error(writer, "I'm a teapot", 418)
}

func sendTrashListError(c *C, server *httptest.Server) {
	tl := map[string]TrashList{
		server.URL: TrashList{TrashRequest{"000000000000000000000000deadbeef", 99}}}

	arv := arvadosclient.ArvadosClient{ApiToken: "abc123"}
	kc := keepclient.KeepClient{Arvados: &arv, Client: &http.Client{}}
	kc.SetServiceRoots(map[string]string{"xxxx": server.URL},
		map[string]string{"xxxx": server.URL},
		map[string]string{})

	err := SendTrashLists(nil, &kc, tl, false)

	c.Check(err, NotNil)
	c.Check(err[0], NotNil)
}

func (s *KeepSuite) TestSendTrashListErrorResponse(c *C) {
	server := httptest.NewServer(&TestHandlerError{})
	sendTrashListError(c, server)
	defer server.Close()
}

func (s *KeepSuite) TestSendTrashListUnreachable(c *C) {
	sendTrashListError(c, httptest.NewUnstartedServer(&TestHandler{}))
}

type APITestData struct {
	numServers int
	serverType string
	statusCode int
}

func (s *KeepSuite) TestGetKeepServers_UnsupportedServiceType(c *C) {
	testGetKeepServersFromAPI(c, APITestData{1, "notadisk", 200}, "Found no keepservices with the service type disk")
}

func (s *KeepSuite) TestGetKeepServers_ReceivedTooFewServers(c *C) {
	testGetKeepServersFromAPI(c, APITestData{2, "disk", 200}, "Did not receive all available keep servers")
}

func (s *KeepSuite) TestGetKeepServers_ServerError(c *C) {
	testGetKeepServersFromAPI(c, APITestData{-1, "disk", -1}, "arvados API server error")
}

func testGetKeepServersFromAPI(c *C, testData APITestData, expectedError string) {
	keepServers := ServiceList{
		ItemsAvailable: testData.numServers,
		KeepServers: []ServerAddress{{
			SSL:         false,
			Host:        "example.com",
			Port:        12345,
			UUID:        "abcdefg",
			ServiceType: testData.serverType,
		}},
	}

	ksJSON, _ := json.Marshal(keepServers)
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/keep_services"] = arvadostest.StubResponse{testData.statusCode, string(ksJSON)}
	apiStub := arvadostest.ServerStub{apiStubResponses}

	api := httptest.NewServer(&apiStub)
	defer api.Close()

	arv := arvadosclient.ArvadosClient{
		Scheme:    "http",
		ApiServer: api.URL[7:],
		ApiToken:  "abc123",
		Client:    &http.Client{Transport: &http.Transport{}},
	}

	kc := keepclient.KeepClient{Arvados: &arv, Client: &http.Client{}}
	kc.SetServiceRoots(map[string]string{"xxxx": "http://example.com:23456"},
		map[string]string{"xxxx": "http://example.com:23456"},
		map[string]string{})

	params := GetKeepServersParams{
		Client: arv,
		Logger: nil,
		Limit:  10,
	}

	_, err := GetKeepServersAndSummarize(params)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, fmt.Sprintf(".*%s.*", expectedError))
}

type KeepServerTestData struct {
	// handle /status.json
	statusStatusCode int

	// handle /index
	indexStatusCode   int
	indexResponseBody string

	// expected error, if any
	expectedError string
}

func (s *KeepSuite) TestGetKeepServers_ErrorGettingKeepServerStatus(c *C) {
	testGetKeepServersAndSummarize(c, KeepServerTestData{500, 200, "ok",
		".*http://.* 500 Internal Server Error"})
}

func (s *KeepSuite) TestGetKeepServers_GettingIndex(c *C) {
	testGetKeepServersAndSummarize(c, KeepServerTestData{200, -1, "notok",
		".*redirect-loop.*"})
}

func (s *KeepSuite) TestGetKeepServers_ErrorReadServerResponse(c *C) {
	testGetKeepServersAndSummarize(c, KeepServerTestData{200, 500, "notok",
		".*http://.* 500 Internal Server Error"})
}

func (s *KeepSuite) TestGetKeepServers_ReadServerResponseTuncatedAtLineOne(c *C) {
	testGetKeepServersAndSummarize(c, KeepServerTestData{200, 200,
		"notterminatedwithnewline", "Index from http://.* truncated at line 1"})
}

func (s *KeepSuite) TestGetKeepServers_InvalidBlockLocatorPattern(c *C) {
	testGetKeepServersAndSummarize(c, KeepServerTestData{200, 200, "testing\n",
		"Error parsing BlockInfo from index line.*"})
}

func (s *KeepSuite) TestGetKeepServers_ReadServerResponseEmpty(c *C) {
	testGetKeepServersAndSummarize(c, KeepServerTestData{200, 200, "\n", ""})
}

func (s *KeepSuite) TestGetKeepServers_ReadServerResponseWithTwoBlocks(c *C) {
	testGetKeepServersAndSummarize(c, KeepServerTestData{200, 200,
		"51752ba076e461ec9ec1d27400a08548+20 1447526361\na048cc05c02ba1ee43ad071274b9e547+52 1447526362\n\n", ""})
}

func testGetKeepServersAndSummarize(c *C, testData KeepServerTestData) {
	ksStubResponses := make(map[string]arvadostest.StubResponse)
	ksStubResponses["/status.json"] = arvadostest.StubResponse{testData.statusStatusCode, string(`{}`)}
	ksStubResponses["/index"] = arvadostest.StubResponse{testData.indexStatusCode, testData.indexResponseBody}
	ksStub := arvadostest.ServerStub{ksStubResponses}
	ks := httptest.NewServer(&ksStub)
	defer ks.Close()

	ksURL, err := url.Parse(ks.URL)
	c.Check(err, IsNil)
	ksHost, port, err := net.SplitHostPort(ksURL.Host)
	ksPort, err := strconv.Atoi(port)
	c.Check(err, IsNil)

	servers_list := ServiceList{
		ItemsAvailable: 1,
		KeepServers: []ServerAddress{{
			SSL:         false,
			Host:        ksHost,
			Port:        ksPort,
			UUID:        "abcdefg",
			ServiceType: "disk",
		}},
	}
	ksJSON, _ := json.Marshal(servers_list)
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/keep_services"] = arvadostest.StubResponse{200, string(ksJSON)}
	apiStub := arvadostest.ServerStub{apiStubResponses}

	api := httptest.NewServer(&apiStub)
	defer api.Close()

	arv := arvadosclient.ArvadosClient{
		Scheme:    "http",
		ApiServer: api.URL[7:],
		ApiToken:  "abc123",
		Client:    &http.Client{Transport: &http.Transport{}},
	}

	kc := keepclient.KeepClient{Arvados: &arv, Client: &http.Client{}}
	kc.SetServiceRoots(map[string]string{"xxxx": ks.URL},
		map[string]string{"xxxx": ks.URL},
		map[string]string{})

	params := GetKeepServersParams{
		Client: arv,
		Logger: nil,
		Limit:  10,
	}

	// GetKeepServersAndSummarize
	results, err := GetKeepServersAndSummarize(params)

	if testData.expectedError == "" {
		c.Assert(err, IsNil)
		c.Assert(results, NotNil)

		blockToServers := results.BlockToServers

		blockLocators := strings.Split(testData.indexResponseBody, "\n")
		for _, loc := range blockLocators {
			locator := strings.Split(loc, " ")[0]
			if locator != "" {
				blockLocator, err := blockdigest.ParseBlockLocator(locator)
				c.Assert(err, IsNil)

				blockDigestWithSize := blockdigest.DigestWithSize{blockLocator.Digest, uint32(blockLocator.Size)}
				blockServerInfo := blockToServers[blockDigestWithSize]
				c.Assert(blockServerInfo[0].Mtime, NotNil)
			}
		}
	} else {
		c.Assert(err, ErrorMatches, testData.expectedError)
	}
}
