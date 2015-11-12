package keep

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	server.Close()

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
