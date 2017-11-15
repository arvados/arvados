// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"crypto/md5"
	"errors"
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
	sync.RWMutex
}

var errStub404 = errors.New("404 block not found")

func (kcs *keepClientStub) ReadAt(locator string, p []byte, off int) (int, error) {
	kcs.RLock()
	defer kcs.RUnlock()
	buf := kcs.blocks[locator[:32]]
	if buf == nil {
		return 0, errStub404
	}
	return copy(p, buf[off:]), nil
}

func (kcs *keepClientStub) PutB(p []byte) (string, int, error) {
	locator := fmt.Sprintf("%x+%d+A12345@abcde", md5.Sum(p), len(p))
	buf := make([]byte, len(p))
	copy(buf, p)
	kcs.Lock()
	defer kcs.Unlock()
	kcs.blocks[locator[:32]] = buf
	return locator, 1, nil
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
	s.fs, err = s.coll.FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
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

	// Force flush to ensure the block "12345678" gets stored, so
	// we know what to expect in the final manifest below.
	_, err = s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)

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
	c.Check(m, check.Equals, "./dir1 3858f62230ac3c915f300c664312c63f+6 25d55ad283aa400af464c76d713c07ad+8 3:3:bar 6:3:foo\n")
}

func (s *CollectionFSSuite) TestMarshalSmallBlocks(c *check.C) {
	maxBlockSize = 8
	defer func() { maxBlockSize = 2 << 26 }()

	var err error
	s.fs, err = (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
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

func (s *CollectionFSSuite) TestMkdir(c *check.C) {
	err := s.fs.Mkdir("foo/bar", 0755)
	c.Check(err, check.Equals, os.ErrNotExist)

	f, err := s.fs.OpenFile("foo/bar", os.O_CREATE, 0)
	c.Check(err, check.Equals, os.ErrNotExist)

	err = s.fs.Mkdir("foo", 0755)
	c.Check(err, check.IsNil)

	f, err = s.fs.OpenFile("foo/bar", os.O_CREATE|os.O_WRONLY, 0)
	c.Check(err, check.IsNil)
	if err == nil {
		defer f.Close()
		f.Write([]byte("foo"))
	}

	// mkdir fails if a file already exists with that name
	err = s.fs.Mkdir("foo/bar", 0755)
	c.Check(err, check.NotNil)

	err = s.fs.Remove("foo/bar")
	c.Check(err, check.IsNil)

	// mkdir succeds after the file is deleted
	err = s.fs.Mkdir("foo/bar", 0755)
	c.Check(err, check.IsNil)

	// creating a file in a nonexistent subdir should still fail
	f, err = s.fs.OpenFile("foo/bar/baz/foo.txt", os.O_CREATE|os.O_WRONLY, 0)
	c.Check(err, check.Equals, os.ErrNotExist)

	f, err = s.fs.OpenFile("foo/bar/foo.txt", os.O_CREATE|os.O_WRONLY, 0)
	c.Check(err, check.IsNil)
	if err == nil {
		defer f.Close()
		f.Write([]byte("foo"))
	}

	// creating foo/bar as a regular file should fail
	f, err = s.fs.OpenFile("foo/bar", os.O_CREATE|os.O_EXCL, 0)
	c.Check(err, check.NotNil)

	// creating foo/bar as a directory should fail
	f, err = s.fs.OpenFile("foo/bar", os.O_CREATE|os.O_EXCL, os.ModeDir)
	c.Check(err, check.NotNil)
	err = s.fs.Mkdir("foo/bar", 0755)
	c.Check(err, check.NotNil)

	m, err := s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	m = regexp.MustCompile(`\+A[^\+ ]+`).ReplaceAllLiteralString(m, "")
	c.Check(m, check.Equals, "./dir1 3858f62230ac3c915f300c664312c63f+6 3:3:bar 0:3:foo\n./foo/bar acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n")
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
	maxBlockSize = 40
	defer func() { maxBlockSize = 2 << 26 }()

	var err error
	s.fs, err = (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	const nfiles = 256
	const ngoroutines = 256

	var wg sync.WaitGroup
	for n := 0; n < nfiles; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			expect := make([]byte, 0, 64)
			wbytes := []byte("there's no simple explanation for anything important that any of us do")
			f, err := s.fs.OpenFile(fmt.Sprintf("random-%d", n), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0)
			c.Assert(err, check.IsNil)
			defer f.Close()
			for i := 0; i < ngoroutines; i++ {
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
			s.checkMemSize(c, f)
		}(n)
	}
	wg.Wait()

	root, err := s.fs.Open("/")
	c.Assert(err, check.IsNil)
	defer root.Close()
	fi, err := root.Readdir(-1)
	c.Check(err, check.IsNil)
	c.Check(len(fi), check.Equals, nfiles)

	_, err = s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	// TODO: check manifest content
}

func (s *CollectionFSSuite) TestPersist(c *check.C) {
	maxBlockSize = 1024
	defer func() { maxBlockSize = 2 << 26 }()

	var err error
	s.fs, err = (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	err = s.fs.Mkdir("d:r", 0755)
	c.Assert(err, check.IsNil)

	expect := map[string][]byte{}

	var wg sync.WaitGroup
	for _, name := range []string{"random 1", "random:2", "random\\3", "d:r/random4"} {
		buf := make([]byte, 500)
		rand.Read(buf)
		expect[name] = buf

		f, err := s.fs.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0)
		c.Assert(err, check.IsNil)
		// Note: we don't close the file until after the test
		// is done. Writes to unclosed files should persist.
		defer f.Close()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < len(buf); i += 5 {
				_, err := f.Write(buf[i : i+5])
				c.Assert(err, check.IsNil)
			}
		}()
	}
	wg.Wait()

	m, err := s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	c.Logf("%q", m)

	root, err := s.fs.Open("/")
	c.Assert(err, check.IsNil)
	defer root.Close()
	fi, err := root.Readdir(-1)
	c.Check(err, check.IsNil)
	c.Check(len(fi), check.Equals, 4)

	persisted, err := (&Collection{ManifestText: m}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	root, err = persisted.Open("/")
	c.Assert(err, check.IsNil)
	defer root.Close()
	fi, err = root.Readdir(-1)
	c.Check(err, check.IsNil)
	c.Check(len(fi), check.Equals, 4)

	for name, content := range expect {
		c.Logf("read %q", name)
		f, err := persisted.Open(name)
		c.Assert(err, check.IsNil)
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		c.Check(err, check.IsNil)
		c.Check(buf, check.DeepEquals, content)
	}
}

func (s *CollectionFSSuite) TestPersistEmptyFiles(c *check.C) {
	var err error
	s.fs, err = (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	for _, name := range []string{"dir", "zero", "zero/zero"} {
		err = s.fs.Mkdir(name, 0755)
		c.Assert(err, check.IsNil)
	}

	expect := map[string][]byte{
		"0":              nil,
		"00":             []byte{},
		"one":            []byte{1},
		"dir/0":          nil,
		"dir/two":        []byte{1, 2},
		"dir/zero":       nil,
		"zero/zero/zero": nil,
	}
	for name, data := range expect {
		f, err := s.fs.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0)
		c.Assert(err, check.IsNil)
		if data != nil {
			_, err := f.Write(data)
			c.Assert(err, check.IsNil)
		}
		f.Close()
	}

	m, err := s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	c.Logf("%q", m)

	persisted, err := (&Collection{ManifestText: m}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	for name, data := range expect {
		f, err := persisted.Open("bogus-" + name)
		c.Check(err, check.NotNil)

		f, err = persisted.Open(name)
		c.Assert(err, check.IsNil)

		if data == nil {
			data = []byte{}
		}
		buf, err := ioutil.ReadAll(f)
		c.Check(err, check.IsNil)
		c.Check(buf, check.DeepEquals, data)
	}
}

func (s *CollectionFSSuite) TestFlushFullBlocks(c *check.C) {
	maxBlockSize = 1024
	defer func() { maxBlockSize = 2 << 26 }()

	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	f, err := fs.OpenFile("50K", os.O_WRONLY|os.O_CREATE, 0)
	c.Assert(err, check.IsNil)
	defer f.Close()

	data := make([]byte, 500)
	rand.Read(data)

	for i := 0; i < 100; i++ {
		n, err := f.Write(data)
		c.Assert(n, check.Equals, len(data))
		c.Assert(err, check.IsNil)
	}

	currentMemExtents := func() (memExtents []int) {
		for idx, e := range f.(*file).inode.(*filenode).extents {
			switch e.(type) {
			case *memExtent:
				memExtents = append(memExtents, idx)
			}
		}
		return
	}
	c.Check(currentMemExtents(), check.HasLen, 1)

	m, err := fs.MarshalManifest(".")
	c.Check(m, check.Not(check.Equals), "")
	c.Check(err, check.IsNil)
	c.Check(currentMemExtents(), check.HasLen, 0)
}

func (s *CollectionFSSuite) TestBrokenManifests(c *check.C) {
	for _, txt := range []string{
		"\n",
		".\n",
		". \n",
		". d41d8cd98f00b204e9800998ecf8427e+0\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 \n",
		". 0:0:foo\n",
		".  0:0:foo\n",
		". 0:0:foo 0:0:bar\n",
		". d41d8cd98f00b204e9800998ecf8427e 0:0:foo\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo:bar\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 foo:0:foo\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:foo:foo\n",
		". d41d8cd98f00b204e9800998ecf8427e+1 0:1:foo 1:1:bar\n",
		". d41d8cd98f00b204e9800998ecf8427e+1 0:0:foo\n./foo d41d8cd98f00b204e9800998ecf8427e+1 0:0:bar\n",
		"./foo d41d8cd98f00b204e9800998ecf8427e+1 0:0:bar\n. d41d8cd98f00b204e9800998ecf8427e+1 0:0:foo\n",
	} {
		c.Logf("<-%q", txt)
		fs, err := (&Collection{ManifestText: txt}).FileSystem(s.client, s.kc)
		c.Check(fs, check.IsNil)
		c.Logf("-> %s", err)
		c.Check(err, check.NotNil)
	}
}

func (s *CollectionFSSuite) TestEdgeCaseManifests(c *check.C) {
	for _, txt := range []string{
		"",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo 0:0:foo 0:0:bar\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo 0:0:foo 0:0:bar\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/bar\n./foo d41d8cd98f00b204e9800998ecf8427e+0 0:0:bar\n",
	} {
		c.Logf("<-%q", txt)
		fs, err := (&Collection{ManifestText: txt}).FileSystem(s.client, s.kc)
		c.Check(err, check.IsNil)
		c.Check(fs, check.NotNil)
	}
}

func (s *CollectionFSSuite) checkMemSize(c *check.C, f File) {
	fn := f.(*file).inode.(*filenode)
	var memsize int64
	for _, ext := range fn.extents {
		if e, ok := ext.(*memExtent); ok {
			memsize += int64(len(e.buf))
		}
	}
	c.Check(fn.memsize, check.Equals, memsize)
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
