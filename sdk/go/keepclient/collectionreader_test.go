// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionReaderUnit{})

type CollectionReaderUnit struct {
	arv     *arvadosclient.ArvadosClient
	kc      *KeepClient
	handler SuccessHandler
}

func (s *CollectionReaderUnit) SetUpTest(c *check.C) {
	var err error
	s.arv, err = arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	s.arv.ApiToken = arvadostest.ActiveToken

	s.kc, err = MakeKeepClient(s.arv)
	c.Assert(err, check.IsNil)

	s.handler = SuccessHandler{
		disk: make(map[string][]byte),
		lock: make(chan struct{}, 1),
		ops:  new(int),
	}
	localRoots := make(map[string]string)
	for i, k := range RunSomeFakeKeepServers(s.handler, 4) {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
	}
	s.kc.SetServiceRoots(localRoots, localRoots, nil)
}

type SuccessHandler struct {
	disk map[string][]byte
	lock chan struct{} // channel with buffer==1: full when an operation is in progress.
	ops  *int          // number of operations completed
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
		<-h.lock
		resp.Write([]byte(pdh))
	case "GET":
		pdh := req.URL.Path[1:]
		h.lock <- struct{}{}
		buf, ok := h.disk[pdh]
		if h.ops != nil {
			(*h.ops)++
		}
		<-h.lock
		if !ok {
			resp.WriteHeader(http.StatusNotFound)
		} else {
			resp.Header().Set("Content-Length", fmt.Sprintf("%d", len(buf)))
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

func (s *CollectionReaderUnit) TestCollectionReaderContent(c *check.C) {
	s.kc.PutB([]byte("foo"))
	s.kc.PutB([]byte("bar"))
	s.kc.PutB([]byte("Hello world\n"))
	s.kc.PutB([]byte(""))

	mt := arvadostest.PathologicalManifest

	for _, testCase := range []rdrTest{
		{mt: mt, f: "zzzz", want: os.ErrNotExist},
		{mt: mt, f: "frob", want: os.ErrNotExist},
		{mt: mt, f: "/segmented/frob", want: "frob"},
		{mt: mt, f: "./segmented/frob", want: "frob"},
		{mt: mt, f: "/f", want: "f"},
		{mt: mt, f: "./f", want: "f"},
		{mt: mt, f: "foo bar//baz", want: "foo"},
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
		c.Logf("%#v", testCase)
		rdr, err := s.kc.CollectionFileReader(map[string]interface{}{"manifest_text": testCase.mt}, testCase.f)
		switch want := testCase.want.(type) {
		case error:
			c.Check(rdr, check.IsNil)
			c.Check(err, check.Equals, want)
		case string:
			buf := make([]byte, len(want))
			_, err := io.ReadFull(rdr, buf)
			c.Check(err, check.IsNil)
			for i := 0; i < 4; i++ {
				c.Check(string(buf), check.Equals, want)
				n, err := rdr.Read(buf)
				c.Check(n, check.Equals, 0)
				c.Check(err, check.Equals, io.EOF)
			}

			for a := len(want) - 2; a >= 0; a-- {
				for b := a + 1; b <= len(want); b++ {
					offset, err := rdr.Seek(int64(a), io.SeekStart)
					c.Logf("...a=%d, b=%d", a, b)
					c.Check(err, check.IsNil)
					c.Check(offset, check.Equals, int64(a))
					buf := make([]byte, b-a)
					n, err := io.ReadFull(rdr, buf)
					c.Check(err, check.IsNil)
					c.Check(n, check.Equals, b-a)
					c.Check(string(buf), check.Equals, want[a:b])
				}
			}
			offset, err := rdr.Seek(-1, io.SeekStart)
			c.Check(err, check.NotNil)
			c.Check(offset, check.Equals, int64(len(want)))

			c.Check(rdr.Close(), check.Equals, nil)
		}
	}
}

func (s *CollectionReaderUnit) TestCollectionReaderManyBlocks(c *check.C) {
	h := md5.New()
	buf := make([]byte, 4096)
	locs := make([]string, len(buf))
	testdata := make([]byte, 0, len(buf)*len(buf))
	filesize := 0
	for i := range locs {
		_, err := rand.Read(buf[:i])
		c.Assert(err, check.IsNil)
		h.Write(buf[:i])
		locs[i], _, err = s.kc.PutB(buf[:i])
		c.Assert(err, check.IsNil)
		filesize += i
		testdata = append(testdata, buf[:i]...)
	}
	manifest := "./random " + strings.Join(locs, " ") + " 0:" + strconv.Itoa(filesize) + ":bytes.bin\n"
	dataMD5 := h.Sum(nil)

	checkMD5 := md5.New()
	rdr, err := s.kc.CollectionFileReader(map[string]interface{}{"manifest_text": manifest}, "random/bytes.bin")
	c.Assert(err, check.IsNil)
	defer rdr.Close()

	_, err = io.Copy(checkMD5, rdr)
	c.Check(err, check.IsNil)
	_, err = rdr.Read(make([]byte, 1))
	c.Check(err, check.Equals, io.EOF)
	c.Check(checkMD5.Sum(nil), check.DeepEquals, dataMD5)

	size, err := rdr.Seek(0, io.SeekEnd)
	c.Check(err, check.IsNil)
	buf = make([]byte, len(testdata))
	copy(buf, testdata)
	curPos := size
	for i := 0; i < 16; i++ {
		offset := rand.Intn(len(buf) - 1)
		count := rand.Intn(len(buf) - offset)
		if rand.Intn(2) == 0 {
			curPos, _ = rdr.Seek(int64(offset)-curPos, io.SeekCurrent)
		} else {
			curPos, _ = rdr.Seek(int64(offset), io.SeekStart)
		}
		c.Check(curPos, check.Equals, int64(offset))
		for count > 0 {
			n, err := rdr.Read(buf[offset : offset+count])
			c.Assert(err, check.IsNil)
			c.Assert(n > 0, check.Equals, true)
			offset += n
			count -= n
		}
		curPos, err = rdr.Seek(0, io.SeekCurrent)
		c.Check(err, check.IsNil)
		c.Check(curPos, check.Equals, int64(offset))
	}
	c.Check(md5.Sum(buf), check.DeepEquals, md5.Sum(testdata))
	c.Check(buf[:1000], check.DeepEquals, testdata[:1000])

	expectPos := curPos + size + 12345
	curPos, err = rdr.Seek(size+12345, io.SeekCurrent)
	c.Check(err, check.IsNil)
	c.Check(curPos, check.Equals, expectPos)

	curPos, err = rdr.Seek(8-curPos, io.SeekCurrent)
	c.Check(err, check.IsNil)
	c.Check(curPos, check.Equals, int64(8))

	curPos, err = rdr.Seek(-9, io.SeekCurrent)
	c.Check(err, check.NotNil)
	c.Check(curPos, check.Equals, int64(8))
}

func (s *CollectionReaderUnit) TestCollectionReaderCloseEarly(c *check.C) {
	// Disable cache
	s.kc.gatewayStack = &keepViaHTTP{s.kc}

	s.kc.PutB([]byte("foo"))
	s.kc.PutB([]byte("bar"))
	s.kc.PutB([]byte("baz"))

	mt := ". "
	for i := 0; i < 300; i++ {
		mt += "acbd18db4cc2f85cedef654fccc4a4d8+3 37b51d194a7513e45b56f6524f2d51f2+3 73feffa4b7f6bb68e44cf984c85f6e88+3 "
	}
	mt += "0:2700:foo900.txt\n"

	// Grab the stub server's lock, ensuring our cfReader doesn't
	// get anything back from its first call to kc.Get() before we
	// have a chance to call Close().
	s.handler.lock <- struct{}{}
	opsBeforeRead := *s.handler.ops

	rdr, err := s.kc.CollectionFileReader(map[string]interface{}{"manifest_text": mt}, "foo900.txt")
	c.Assert(err, check.IsNil)

	firstReadDone := make(chan struct{})
	go func() {
		n, err := rdr.Read(make([]byte, 3))
		c.Check(n, check.Equals, 3)
		c.Check(err, check.IsNil)
		close(firstReadDone)
	}()

	// Release the stub server's lock. The first GET operation will proceed.
	<-s.handler.lock

	// Make sure our first read operation consumes the data
	// received from the first GET.
	<-firstReadDone

	err = rdr.Close()
	c.Check(err, check.IsNil)

	// Stub should have handled exactly one GET request.
	c.Check(*s.handler.ops, check.Equals, opsBeforeRead+1)
}

func (s *CollectionReaderUnit) TestCollectionReaderDataError(c *check.C) {
	manifest := ". ffffffffffffffffffffffffffffffff+1 0:1:notfound.txt\n"
	buf := make([]byte, 1)
	rdr, err := s.kc.CollectionFileReader(map[string]interface{}{"manifest_text": manifest}, "notfound.txt")
	c.Check(err, check.IsNil)
	for i := 0; i < 2; i++ {
		_, err = io.ReadFull(rdr, buf)
		c.Check(err, check.NotNil)
		c.Check(err, check.Not(check.Equals), io.EOF)
	}
	c.Check(rdr.Close(), check.IsNil)
}
