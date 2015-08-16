package keepclient

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&IntegrationSuite{})

// IntegrationSuite tests need an API server
type IntegrationSuite struct{}

type SuccessHandler struct {
	disk map[string][]byte
	lock chan struct{}	// channel with buffer==1: full when an operation is in progress.
	ops  *int		// number of operations completed
}

func (h SuccessHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "PUT":
		buf, err := ioutil.ReadAll(req.Body)
		if err != nil {
			resp.WriteHeader(500)
			return
		}
		pdh := fmt.Sprintf("%x+%d", md5.Sum(buf), len(buf))
		h.lock <- struct{}{}
		h.disk[pdh] = buf
		if h.ops != nil {
			(*h.ops)++
		}
		<- h.lock
		resp.Write([]byte(pdh))
	case "GET":
		pdh := req.URL.Path[1:]
		h.lock <- struct{}{}
		buf, ok := h.disk[pdh]
		if h.ops != nil {
			(*h.ops)++
		}
		<- h.lock
		if !ok {
			resp.WriteHeader(http.StatusNotFound)
		} else {
			resp.Write(buf)
		}
	default:
		resp.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type rdrTest struct {
	mt   string      // manifest text
	f    string      // filename
	want interface{} // error or string to expect
}

func StubWithFakeServers(kc *KeepClient, h http.Handler) {
	localRoots := make(map[string]string)
	for i, k := range RunSomeFakeKeepServers(h, 4) {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
	}
	kc.SetServiceRoots(localRoots, localRoots, nil)
}

func (s *ServerRequiredSuite) TestCollectionReaderContent(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	arv.ApiToken = arvadostest.ActiveToken

	kc, err := MakeKeepClient(&arv)
	c.Assert(err, check.IsNil)

	{
		h := SuccessHandler{
			disk: make(map[string][]byte),
			lock: make(chan struct{}, 1),
		}
		StubWithFakeServers(kc, h)
		kc.PutB([]byte("foo"))
		kc.PutB([]byte("bar"))
		kc.PutB([]byte("Hello world\n"))
		kc.PutB([]byte(""))
	}

	mt := arvadostest.PathologicalManifest

	for _, testCase := range []rdrTest{
		{mt: mt, f: "zzzz", want: os.ErrNotExist},
		{mt: mt, f: "frob", want: os.ErrNotExist},
		{mt: mt, f: "/segmented/frob", want: os.ErrNotExist},
		{mt: mt, f: "./segmented/frob", want: os.ErrNotExist},
		{mt: mt, f: "/f", want: os.ErrNotExist},
		{mt: mt, f: "./f", want: os.ErrNotExist},
		{mt: mt, f: "foo bar//baz", want: os.ErrNotExist},
		{mt: mt, f: "foo/zero", want: ""},
		{mt: mt, f: "zero@0", want: ""},
		{mt: mt, f: "zero@1", want: ""},
		{mt: mt, f: "zero@4", want: ""},
		{mt: mt, f: "zero@9", want: ""},
		{mt: mt, f: "f", want: "f"},
		{mt: mt, f: "ooba", want: "ooba"},
		{mt: mt, f: "overlapReverse/o", want: "o"},
		{mt: mt, f: "overlapReverse/oo", want: "oo"},
		{mt: mt, f: "overlapReverse/ofoo", want: "ofoo"},
		{mt: mt, f: "foo bar/baz", want: "foo"},
		{mt: mt, f: "segmented/frob", want: "frob"},
		{mt: mt, f: "segmented/oof", want: "oof"},
	} {
		rdr, err := kc.CollectionFileReader(map[string]interface{}{"manifest_text": testCase.mt}, testCase.f)
		switch want := testCase.want.(type) {
		case error:
			c.Check(rdr, check.IsNil)
			c.Check(err, check.Equals, want)
		case string:
			buf := make([]byte, len(want))
			n, err := io.ReadFull(rdr, buf)
			c.Check(err, check.IsNil)
			for i := 0; i < 4; i++ {
				c.Check(string(buf), check.Equals, want)
				n, err = rdr.Read(buf)
				c.Check(n, check.Equals, 0)
				c.Check(err, check.Equals, io.EOF)
			}
			c.Check(rdr.Close(), check.Equals, nil)
		}
	}
}

func (s *ServerRequiredSuite) TestCollectionReaderCloseEarly(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	arv.ApiToken = arvadostest.ActiveToken

	kc, err := MakeKeepClient(&arv)
	c.Assert(err, check.IsNil)

	h := SuccessHandler{
		disk: make(map[string][]byte),
		lock: make(chan struct{}, 1),
		ops: new(int),
	}
	StubWithFakeServers(kc, h)
	kc.PutB([]byte("foo"))

	mt := ". "
	for i := 0; i < 1000; i++ {
		mt += "acbd18db4cc2f85cedef654fccc4a4d8+3 "
	}
	mt += "0:3000:foo1000.txt\n"

	// Grab the stub server's lock, ensuring our cfReader doesn't
	// get anything back from its first call to kc.Get() before we
	// have a chance to call Close().
	h.lock <- struct{}{}
	opsBeforeRead := *h.ops

	rdr, err := kc.CollectionFileReader(map[string]interface{}{"manifest_text": mt}, "foo1000.txt")
	c.Assert(err, check.IsNil)
	err = rdr.Close()
	c.Assert(err, check.IsNil)
	c.Assert(rdr.Error(), check.IsNil)

	// Release the stub server's lock. The first GET operation will proceed.
	<-h.lock

	// doGet() should close toRead before sending any more bufs to it.
	if what, ok := <-rdr.toRead;  ok {
		c.Errorf("Got %+v, expected toRead to be closed", what)
	}

	// Stub should have handled exactly one GET request.
	c.Assert(*h.ops, check.Equals, opsBeforeRead+1)
}
