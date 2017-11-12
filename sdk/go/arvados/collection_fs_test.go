// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"sync"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionFSSuite{})

type keepClientStub struct {
	blocks map[string][]byte
}

func (kcs *keepClientStub) ReadAt(locator string, p []byte, off int) (int, error) {
	buf := kcs.blocks[locator[:32]]
	if buf == nil {
		return 0, os.ErrNotExist
	}
	return copy(p, buf[off:]), nil
}

type CollectionFSSuite struct {
	client *Client
	coll   Collection
	fs     CollectionFileSystem
	kc     keepClient
}

func (s *CollectionFSSuite) SetUpTest(c *check.C) {
	s.client = NewClientFromEnv()
	err := s.client.RequestAndDecode(&s.coll, "GET", "arvados/v1/collections/"+arvadostest.FooAndBarFilesInDirUUID, nil, nil)
	c.Assert(err, check.IsNil)
	s.kc = &keepClientStub{
		blocks: map[string][]byte{
			"3858f62230ac3c915f300c664312c63f": []byte("foobar"),
		}}
	s.fs = s.coll.FileSystem(s.client, s.kc)
}

func (s *CollectionFSSuite) TestHttpFileSystemInterface(c *check.C) {
	_, ok := s.fs.(http.FileSystem)
	c.Check(ok, check.Equals, true)
}

func (s *CollectionFSSuite) TestReaddirFull(c *check.C) {
	f, err := s.fs.Open("/dir1")
	c.Assert(err, check.IsNil)

	st, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(2))
	c.Check(st.IsDir(), check.Equals, true)

	fis, err := f.Readdir(0)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 2)
	if len(fis) > 0 {
		c.Check(fis[0].Size(), check.Equals, int64(3))
	}
}

func (s *CollectionFSSuite) TestReaddirLimited(c *check.C) {
	f, err := s.fs.Open("./dir1")
	c.Assert(err, check.IsNil)

	fis, err := f.Readdir(1)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 1)
	if len(fis) > 0 {
		c.Check(fis[0].Size(), check.Equals, int64(3))
	}

	fis, err = f.Readdir(1)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 1)
	if len(fis) > 0 {
		c.Check(fis[0].Size(), check.Equals, int64(3))
	}

	fis, err = f.Readdir(1)
	c.Check(len(fis), check.Equals, 0)
	c.Check(err, check.NotNil)
	c.Check(err, check.Equals, io.EOF)

	f, err = s.fs.Open("dir1")
	c.Assert(err, check.IsNil)
	fis, err = f.Readdir(1)
	c.Check(len(fis), check.Equals, 1)
	c.Assert(err, check.IsNil)
	fis, err = f.Readdir(2)
	c.Check(len(fis), check.Equals, 1)
	c.Assert(err, check.IsNil)
	fis, err = f.Readdir(2)
	c.Check(len(fis), check.Equals, 0)
	c.Assert(err, check.Equals, io.EOF)
}

func (s *CollectionFSSuite) TestPathMunge(c *check.C) {
	for _, path := range []string{".", "/", "./", "///", "/../", "/./.."} {
		f, err := s.fs.Open(path)
		c.Assert(err, check.IsNil)

		st, err := f.Stat()
		c.Assert(err, check.IsNil)
		c.Check(st.Size(), check.Equals, int64(1))
		c.Check(st.IsDir(), check.Equals, true)
	}
	for _, path := range []string{"/dir1", "dir1", "./dir1", "///dir1//.//", "../dir1/../dir1/"} {
		c.Logf("%q", path)
		f, err := s.fs.Open(path)
		c.Assert(err, check.IsNil)

		st, err := f.Stat()
		c.Assert(err, check.IsNil)
		c.Check(st.Size(), check.Equals, int64(2))
		c.Check(st.IsDir(), check.Equals, true)
	}
}

func (s *CollectionFSSuite) TestNotExist(c *check.C) {
	for _, path := range []string{"/no", "no", "./no", "n/o", "/n/o"} {
		f, err := s.fs.Open(path)
		c.Assert(f, check.IsNil)
		c.Assert(err, check.NotNil)
		c.Assert(os.IsNotExist(err), check.Equals, true)
	}
}

func (s *CollectionFSSuite) TestReadOnlyFile(c *check.C) {
	f, err := s.fs.OpenFile("/dir1/foo", os.O_RDONLY, 0)
	c.Assert(err, check.IsNil)
	st, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(3))
	n, err := f.Write([]byte("bar"))
	c.Check(n, check.Equals, 0)
	c.Check(err, check.Equals, ErrReadOnlyFile)
}

func (s *CollectionFSSuite) TestCreateFile(c *check.C) {
	f, err := s.fs.OpenFile("/newfile", os.O_RDWR|os.O_CREATE, 0)
	c.Assert(err, check.IsNil)
	st, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(0))

	n, err := f.Write([]byte("bar"))
	c.Check(n, check.Equals, 3)
	c.Check(err, check.IsNil)

	c.Check(f.Close(), check.IsNil)

	f, err = s.fs.OpenFile("/newfile", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0)
	c.Check(f, check.IsNil)
	c.Assert(err, check.NotNil)

	f, err = s.fs.OpenFile("/newfile", os.O_RDWR, 0)
	c.Assert(err, check.IsNil)
	st, err = f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(3))

	c.Check(f.Close(), check.IsNil)

	// TODO: serialize to Collection, confirm manifest contents,
	// make new FileSystem, confirm file contents.
}

func (s *CollectionFSSuite) TestReadWriteFile(c *check.C) {
	maxBlockSize = 8
	defer func() { maxBlockSize = 2 << 26 }()

	f, err := s.fs.OpenFile("/dir1/foo", os.O_RDWR, 0)
	c.Assert(err, check.IsNil)
	defer f.Close()
	st, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(3))

	f2, err := s.fs.OpenFile("/dir1/foo", os.O_RDWR, 0)
	c.Assert(err, check.IsNil)
	defer f2.Close()

	buf := make([]byte, 64)
	n, err := f.Read(buf)
	c.Check(n, check.Equals, 3)
	c.Check(err, check.Equals, io.EOF)
	c.Check(string(buf[:3]), check.DeepEquals, "foo")

	pos, err := f.Seek(-2, os.SEEK_CUR)
	c.Check(pos, check.Equals, int64(1))
	c.Check(err, check.IsNil)

	// Split a storedExtent in two, and insert a memExtent
	n, err = f.Write([]byte("*"))
	c.Check(n, check.Equals, 1)
	c.Check(err, check.IsNil)

	pos, err = f.Seek(0, os.SEEK_CUR)
	c.Check(pos, check.Equals, int64(2))
	c.Check(err, check.IsNil)

	pos, err = f.Seek(0, os.SEEK_SET)
	c.Check(pos, check.Equals, int64(0))
	c.Check(err, check.IsNil)

	rbuf, err := ioutil.ReadAll(f)
	c.Check(len(rbuf), check.Equals, 3)
	c.Check(err, check.IsNil)
	c.Check(string(rbuf), check.Equals, "f*o")

	// Write multiple blocks in one call
	f.Seek(1, os.SEEK_SET)
	n, err = f.Write([]byte("0123456789abcdefg"))
	c.Check(n, check.Equals, 17)
	c.Check(err, check.IsNil)
	pos, err = f.Seek(0, os.SEEK_CUR)
	c.Check(pos, check.Equals, int64(18))
	pos, err = f.Seek(-18, os.SEEK_CUR)
	c.Check(err, check.IsNil)
	n, err = io.ReadFull(f, buf)
	c.Check(n, check.Equals, 18)
	c.Check(err, check.Equals, io.ErrUnexpectedEOF)
	c.Check(string(buf[:n]), check.Equals, "f0123456789abcdefg")

	buf2, err := ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "f0123456789abcdefg")

	// truncate to current size
	err = f.Truncate(18)
	f2.Seek(0, os.SEEK_SET)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "f0123456789abcdefg")

	// shrink to zero some data
	f.Truncate(15)
	f2.Seek(0, os.SEEK_SET)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "f0123456789abcd")

	// grow to partial block/extent
	f.Truncate(20)
	f2.Seek(0, os.SEEK_SET)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "f0123456789abcd\x00\x00\x00\x00\x00")

	f.Truncate(0)
	f2.Write([]byte("12345678abcdefghijkl"))

	// grow to block/extent boundary
	f.Truncate(64)
	f2.Seek(0, os.SEEK_SET)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(len(buf2), check.Equals, 64)
	c.Check(len(f.(*file).inode.(*filenode).extents), check.Equals, 8)

	// shrink to block/extent boundary
	err = f.Truncate(32)
	f2.Seek(0, os.SEEK_SET)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(len(buf2), check.Equals, 32)
	c.Check(len(f.(*file).inode.(*filenode).extents), check.Equals, 4)

	// shrink to partial block/extent
	err = f.Truncate(15)
	f2.Seek(0, os.SEEK_SET)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "12345678abcdefg")
	c.Check(len(f.(*file).inode.(*filenode).extents), check.Equals, 2)

	// Truncate to size=3 while f2's ptr is at 15
	err = f.Truncate(3)
	c.Check(err, check.IsNil)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "")
	f2.Seek(0, os.SEEK_SET)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "123")
	c.Check(len(f.(*file).inode.(*filenode).extents), check.Equals, 1)

	m, err := s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	m = regexp.MustCompile(`\+A[^\+ ]+`).ReplaceAllLiteralString(m, "")
	c.Check(m, check.Equals, "./dir1 3858f62230ac3c915f300c664312c63f+6 202cb962ac59075b964b07152d234b70+3 3:3:bar 6:3:foo\n")
}

func (s *CollectionFSSuite) TestMarshalSmallBlocks(c *check.C) {
	maxBlockSize = 8
	defer func() { maxBlockSize = 2 << 26 }()

	s.fs = (&Collection{}).FileSystem(s.client, s.kc)
	for _, name := range []string{"foo", "bar", "baz"} {
		f, err := s.fs.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0)
		c.Assert(err, check.IsNil)
		f.Write([]byte(name))
		f.Close()
	}

	m, err := s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	m = regexp.MustCompile(`\+A[^\+ ]+`).ReplaceAllLiteralString(m, "")
	c.Check(m, check.Equals, ". c3c23db5285662ef7172373df0003206+6 acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar 3:3:baz 6:3:foo\n")
}

func (s *CollectionFSSuite) TestConcurrentWriters(c *check.C) {
	maxBlockSize = 8
	defer func() { maxBlockSize = 2 << 26 }()

	var wg sync.WaitGroup
	for n := 0; n < 128; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := s.fs.OpenFile("/dir1/foo", os.O_RDWR, 0)
			c.Assert(err, check.IsNil)
			defer f.Close()
			for i := 0; i < 6502; i++ {
				switch rand.Int() & 3 {
				case 0:
					f.Truncate(int64(rand.Intn(64)))
				case 1:
					f.Seek(int64(rand.Intn(64)), os.SEEK_SET)
				case 2:
					_, err := f.Write([]byte("beep boop"))
					c.Check(err, check.IsNil)
				case 3:
					_, err := ioutil.ReadAll(f)
					c.Check(err, check.IsNil)
				}
			}
		}()
	}
	wg.Wait()

	f, err := s.fs.OpenFile("/dir1/foo", os.O_RDWR, 0)
	c.Assert(err, check.IsNil)
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	c.Check(err, check.IsNil)
	c.Logf("after lots of random r/w/seek/trunc, buf is %q", buf)
}

func (s *CollectionFSSuite) TestRandomWrites(c *check.C) {
	maxBlockSize = 8
	defer func() { maxBlockSize = 2 << 26 }()

	var wg sync.WaitGroup
	for n := 0; n < 128; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			expect := make([]byte, 0, 64)
			wbytes := []byte("there's no simple explanation for anything important that any of us do")
			f, err := s.fs.OpenFile(fmt.Sprintf("random-%d", n), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0)
			c.Assert(err, check.IsNil)
			defer f.Close()
			for i := 0; i < 6502; i++ {
				trunc := rand.Intn(65)
				woff := rand.Intn(trunc + 1)
				wbytes = wbytes[:rand.Intn(64-woff+1)]
				for buf, i := expect[:cap(expect)], len(expect); i < trunc; i++ {
					buf[i] = 0
				}
				expect = expect[:trunc]
				if trunc < woff+len(wbytes) {
					expect = expect[:woff+len(wbytes)]
				}
				copy(expect[woff:], wbytes)
				f.Truncate(int64(trunc))
				pos, err := f.Seek(int64(woff), os.SEEK_SET)
				c.Check(pos, check.Equals, int64(woff))
				c.Check(err, check.IsNil)
				n, err := f.Write(wbytes)
				c.Check(n, check.Equals, len(wbytes))
				c.Check(err, check.IsNil)
				pos, err = f.Seek(0, os.SEEK_SET)
				c.Check(pos, check.Equals, int64(0))
				c.Check(err, check.IsNil)
				buf, err := ioutil.ReadAll(f)
				c.Check(string(buf), check.Equals, string(expect))
				c.Check(err, check.IsNil)
			}
		}(n)
	}
	wg.Wait()

	root, err := s.fs.Open("/")
	c.Assert(err, check.IsNil)
	defer root.Close()
	fi, err := root.Readdir(-1)
	c.Check(err, check.IsNil)
	c.Logf("Readdir(): %#v", fi)

	m, err := s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	c.Logf("%s", m)
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
