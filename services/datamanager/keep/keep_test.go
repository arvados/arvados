package keep

import (
	"encoding/json"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"testing"
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

func (this *TestHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	r := json.NewDecoder(req.Body)
	r.Decode(&this.request)
}

func (s *KeepSuite) TestSendTrashLists(c *C) {
	th := TestHandler{}
	server := httptest.NewServer(&th)

	tl := map[string]TrashList{
		server.URL: TrashList{TrashRequest{"000000000000000000000000deadbeef", 99}}}

	kc := keepclient.KeepClient{Client: &http.Client{}}
	kc.SetServiceRoots(map[string]string{"xxxx": server.URL},
		map[string]string{"xxxx": server.URL},
		map[string]string{})

	err := SendTrashLists("", &kc, tl)
	server.Close()

	c.Check(err[0], IsNil)

	c.Check(th.request,
		DeepEquals,
		tl[server.URL])

}

type TestHandlerError struct {
}

func (this *TestHandlerError) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	http.Error(writer, "I'm a teapot", 418)
}

func sendTrashListError(c *C, server *httptest.Server) {
	tl := map[string]TrashList{
		server.URL: TrashList{TrashRequest{"000000000000000000000000deadbeef", 99}}}

	kc := keepclient.KeepClient{Client: &http.Client{}}
	kc.SetServiceRoots(map[string]string{"xxxx": server.URL},
		map[string]string{"xxxx": server.URL},
		map[string]string{})

	err := SendTrashLists("", &kc, tl)

	c.Check(err[0], NotNil)
}

func (s *KeepSuite) TestSendTrashListErrorResponse(c *C) {
	sendTrashListError(c, httptest.NewServer(&TestHandlerError{}))
}

func (s *KeepSuite) TestSendTrashListUnreachable(c *C) {
	sendTrashListError(c, httptest.NewUnstartedServer(&TestHandler{}))
}
