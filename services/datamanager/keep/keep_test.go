package keep

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
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

	err := SendTrashLists(&kc, tl)

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

	err := SendTrashLists(&kc, tl)

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

type APIStub struct {
	respStatus   int
	responseBody string
}

func (h *APIStub) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(h.respStatus)
	resp.Write([]byte(h.responseBody))
}

type KeepServerStub struct {
}

func (ts *KeepServerStub) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	http.Error(resp, "oops", 500)
}

func (s *KeepSuite) TestGetKeepServers_UnsupportedServiceType(c *C) {
	keepServers := ServiceList{
		ItemsAvailable: 1,
		KeepServers: []ServerAddress{{
			SSL:         false,
			Host:        "example.com",
			Port:        12345,
			UUID:        "abcdefg",
			ServiceType: "nondisk",
		}},
	}

	ksJSON, _ := json.Marshal(keepServers)
	apiStub := APIStub{200, string(ksJSON)}

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
	c.Assert(err, ErrorMatches, ".*Unsupported service type.*")
}

func (s *KeepSuite) TestGetKeepServers_ReceivedTooFewServers(c *C) {
	keepServers := ServiceList{
		ItemsAvailable: 2,
		KeepServers: []ServerAddress{{
			SSL:         false,
			Host:        "example.com",
			Port:        12345,
			UUID:        "abcdefg",
			ServiceType: "disk",
		}},
	}

	ksJSON, _ := json.Marshal(keepServers)
	apiStub := APIStub{200, string(ksJSON)}

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
	c.Assert(err, ErrorMatches, ".*Did not receive all available keep servers.*")
}

func (s *KeepSuite) TestGetKeepServers_ErrorGettingKeepServerStatus(c *C) {
	ksStub := KeepServerStub{}
	ks := httptest.NewServer(&ksStub)
	defer ks.Close()

	ksURL, err := url.Parse(ks.URL)
	c.Check(err, IsNil)
	ksURLParts := strings.Split(ksURL.Host, ":")
	ksPort, err := strconv.Atoi(ksURLParts[1])
	c.Check(err, IsNil)

	servers_list := ServiceList{
		ItemsAvailable: 1,
		KeepServers: []ServerAddress{{
			SSL:         false,
			Host:        strings.Split(ksURL.Host, ":")[0],
			Port:        ksPort,
			UUID:        "abcdefg",
			ServiceType: "disk",
		}},
	}
	ksJSON, _ := json.Marshal(servers_list)
	apiStub := APIStub{200, string(ksJSON)}

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

	// This fails during GetServerStatus
	_, err = GetKeepServersAndSummarize(params)
	c.Assert(err, ErrorMatches, ".*Error during GetServerContents; no host info found.*")
}
