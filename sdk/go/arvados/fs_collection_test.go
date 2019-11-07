// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionFSSuite{})

type keepClientStub struct {
	blocks      map[string][]byte
	refreshable map[string]bool
	onPut       func(bufcopy []byte) // called from PutB, before acquiring lock
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
	if kcs.onPut != nil {
		kcs.onPut(buf)
	}
	kcs.Lock()
	defer kcs.Unlock()
	kcs.blocks[locator[:32]] = buf
	return locator, 1, nil
}

var localOrRemoteSignature = regexp.MustCompile(`\+[AR][^+]*`)

func (kcs *keepClientStub) LocalLocator(locator string) (string, error) {
	kcs.Lock()
	defer kcs.Unlock()
	if strings.Contains(locator, "+R") {
		if len(locator) < 32 {
			return "", fmt.Errorf("bad locator: %q", locator)
		}
		if _, ok := kcs.blocks[locator[:32]]; !ok && !kcs.refreshable[locator[:32]] {
			return "", fmt.Errorf("kcs.refreshable[%q]==false", locator)
		}
	}
	fakeSig := fmt.Sprintf("+A%x@%x", sha1.Sum(nil), time.Now().Add(time.Hour*24*14).Unix())
	return localOrRemoteSignature.ReplaceAllLiteralString(locator, fakeSig), nil
}

type CollectionFSSuite struct {
	client *Client
	coll   Collection
	fs     CollectionFileSystem
	kc     *keepClientStub
}

func (s *CollectionFSSuite) SetUpTest(c *check.C) {
	s.client = NewClientFromEnv()
	err := s.client.RequestAndDecode(&s.coll, "GET", "arvados/v1/collections/"+fixtureFooAndBarFilesInDirUUID, nil, nil)
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

func (s *CollectionFSSuite) TestColonInFilename(c *check.C) {
	fs, err := (&Collection{
		ManifestText: "./foo:foo 3858f62230ac3c915f300c664312c63f+3 0:3:bar:bar\n",
	}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	f, err := fs.Open("/foo:foo")
	c.Assert(err, check.IsNil)

	fis, err := f.Readdir(0)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 1)
	c.Check(fis[0].Name(), check.Equals, "bar:bar")
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
	f, err := s.fs.OpenFile("/new-file 1", os.O_RDWR|os.O_CREATE, 0)
	c.Assert(err, check.IsNil)
	st, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(0))

	n, err := f.Write([]byte("bar"))
	c.Check(n, check.Equals, 3)
	c.Check(err, check.IsNil)

	c.Check(f.Close(), check.IsNil)

	f, err = s.fs.OpenFile("/new-file 1", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0)
	c.Check(f, check.IsNil)
	c.Assert(err, check.NotNil)

	f, err = s.fs.OpenFile("/new-file 1", os.O_RDWR, 0)
	c.Assert(err, check.IsNil)
	st, err = f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(3))

	c.Check(f.Close(), check.IsNil)

	m, err := s.fs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	c.Check(m, check.Matches, `. 37b51d194a7513e45b56f6524f2d51f2\+3\+\S+ 0:3:new-file\\0401\n./dir1 .* 3:3:bar 0:3:foo\n`)
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

	pos, err := f.Seek(-2, io.SeekCurrent)
	c.Check(pos, check.Equals, int64(1))
	c.Check(err, check.IsNil)

	// Split a storedExtent in two, and insert a memExtent
	n, err = f.Write([]byte("*"))
	c.Check(n, check.Equals, 1)
	c.Check(err, check.IsNil)

	pos, err = f.Seek(0, io.SeekCurrent)
	c.Check(pos, check.Equals, int64(2))
	c.Check(err, check.IsNil)

	pos, err = f.Seek(0, io.SeekStart)
	c.Check(pos, check.Equals, int64(0))
	c.Check(err, check.IsNil)

	rbuf, err := ioutil.ReadAll(f)
	c.Check(len(rbuf), check.Equals, 3)
	c.Check(err, check.IsNil)
	c.Check(string(rbuf), check.Equals, "f*o")

	// Write multiple blocks in one call
	f.Seek(1, io.SeekStart)
	n, err = f.Write([]byte("0123456789abcdefg"))
	c.Check(n, check.Equals, 17)
	c.Check(err, check.IsNil)
	pos, err = f.Seek(0, io.SeekCurrent)
	c.Check(pos, check.Equals, int64(18))
	c.Check(err, check.IsNil)
	pos, err = f.Seek(-18, io.SeekCurrent)
	c.Check(pos, check.Equals, int64(0))
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
	c.Check(err, check.IsNil)
	f2.Seek(0, io.SeekStart)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "f0123456789abcdefg")

	// shrink to zero some data
	f.Truncate(15)
	f2.Seek(0, io.SeekStart)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "f0123456789abcd")

	// grow to partial block/extent
	f.Truncate(20)
	f2.Seek(0, io.SeekStart)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "f0123456789abcd\x00\x00\x00\x00\x00")

	f.Truncate(0)
	f2.Seek(0, io.SeekStart)
	f2.Write([]byte("12345678abcdefghijkl"))

	// grow to block/extent boundary
	f.Truncate(64)
	f2.Seek(0, io.SeekStart)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(len(buf2), check.Equals, 64)
	c.Check(len(f.(*filehandle).inode.(*filenode).segments), check.Equals, 8)

	// shrink to block/extent boundary
	err = f.Truncate(32)
	c.Check(err, check.IsNil)
	f2.Seek(0, io.SeekStart)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(len(buf2), check.Equals, 32)
	c.Check(len(f.(*filehandle).inode.(*filenode).segments), check.Equals, 4)

	// shrink to partial block/extent
	err = f.Truncate(15)
	c.Check(err, check.IsNil)
	f2.Seek(0, io.SeekStart)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "12345678abcdefg")
	c.Check(len(f.(*filehandle).inode.(*filenode).segments), check.Equals, 2)

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
	f2.Seek(0, io.SeekStart)
	buf2, err = ioutil.ReadAll(f2)
	c.Check(err, check.IsNil)
	c.Check(string(buf2), check.Equals, "123")
	c.Check(len(f.(*filehandle).inode.(*filenode).segments), check.Equals, 1)

	m, err := s.fs.MarshalManifest(".")
	c.Check(err, check.IsNil)
	m = regexp.MustCompile(`\+A[^\+ ]+`).ReplaceAllLiteralString(m, "")
	c.Check(m, check.Equals, "./dir1 3858f62230ac3c915f300c664312c63f+6 25d55ad283aa400af464c76d713c07ad+8 3:3:bar 6:3:foo\n")
	c.Check(s.fs.Size(), check.Equals, int64(6))
}

func (s *CollectionFSSuite) TestSeekSparse(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	f, err := fs.OpenFile("test", os.O_CREATE|os.O_RDWR, 0755)
	c.Assert(err, check.IsNil)
	defer f.Close()

	checkSize := func(size int64) {
		fi, err := f.Stat()
		c.Assert(err, check.IsNil)
		c.Check(fi.Size(), check.Equals, size)

		f, err := fs.OpenFile("test", os.O_CREATE|os.O_RDWR, 0755)
		c.Assert(err, check.IsNil)
		defer f.Close()
		fi, err = f.Stat()
		c.Check(err, check.IsNil)
		c.Check(fi.Size(), check.Equals, size)
		pos, err := f.Seek(0, io.SeekEnd)
		c.Check(err, check.IsNil)
		c.Check(pos, check.Equals, size)
	}

	f.Seek(2, io.SeekEnd)
	checkSize(0)
	f.Write([]byte{1})
	checkSize(3)

	f.Seek(2, io.SeekCurrent)
	checkSize(3)
	f.Write([]byte{})
	checkSize(5)

	f.Seek(8, io.SeekStart)
	checkSize(5)
	n, err := f.Read(make([]byte, 1))
	c.Check(n, check.Equals, 0)
	c.Check(err, check.Equals, io.EOF)
	checkSize(5)
	f.Write([]byte{1, 2, 3})
	checkSize(11)
}

func (s *CollectionFSSuite) TestMarshalCopiesRemoteBlocks(c *check.C) {
	foo := "foo"
	bar := "bar"
	hash := map[string]string{
		foo: fmt.Sprintf("%x", md5.Sum([]byte(foo))),
		bar: fmt.Sprintf("%x", md5.Sum([]byte(bar))),
	}

	fs, err := (&Collection{
		ManifestText: ". " + hash[foo] + "+3+Rzaaaa-foo@bab " + hash[bar] + "+3+A12345@ffffff 0:2:fo.txt 2:4:obar.txt\n",
	}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	manifest, err := fs.MarshalManifest(".")
	c.Check(manifest, check.Equals, "")
	c.Check(err, check.NotNil)

	s.kc.refreshable = map[string]bool{hash[bar]: true}

	for _, sigIn := range []string{"Rzaaaa-foo@bab", "A12345@abcde"} {
		fs, err = (&Collection{
			ManifestText: ". " + hash[foo] + "+3+A12345@fffff " + hash[bar] + "+3+" + sigIn + " 0:2:fo.txt 2:4:obar.txt\n",
		}).FileSystem(s.client, s.kc)
		c.Assert(err, check.IsNil)
		manifest, err := fs.MarshalManifest(".")
		c.Check(err, check.IsNil)
		// Both blocks should now have +A signatures.
		c.Check(manifest, check.Matches, `.*\+A.* .*\+A.*\n`)
		c.Check(manifest, check.Not(check.Matches), `.*\+R.*\n`)
	}
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

	// mkdir succeeds after the file is deleted
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
	if testing.Short() {
		c.Skip("slow")
	}

	maxBlockSize = 8
	defer func() { maxBlockSize = 1 << 26 }()

	var wg sync.WaitGroup
	for n := 0; n < 128; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, err := s.fs.OpenFile("/dir1/foo", os.O_RDWR, 0)
			c.Assert(err, check.IsNil)
			defer f.Close()
			for i := 0; i < 1024; i++ {
				r := rand.Uint32()
				switch {
				case r%11 == 0:
					_, err := s.fs.MarshalManifest(".")
					c.Check(err, check.IsNil)
				case r&3 == 0:
					f.Truncate(int64(rand.Intn(64)))
				case r&3 == 1:
					f.Seek(int64(rand.Intn(64)), io.SeekStart)
				case r&3 == 2:
					_, err := f.Write([]byte("beep boop"))
					c.Check(err, check.IsNil)
				case r&3 == 3:
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
	for n := 0; n < ngoroutines; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			expect := make([]byte, 0, 64)
			wbytes := []byte("there's no simple explanation for anything important that any of us do")
			f, err := s.fs.OpenFile(fmt.Sprintf("random-%d", n), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0)
			c.Assert(err, check.IsNil)
			defer f.Close()
			for i := 0; i < nfiles; i++ {
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
				pos, err := f.Seek(int64(woff), io.SeekStart)
				c.Check(pos, check.Equals, int64(woff))
				c.Check(err, check.IsNil)
				n, err := f.Write(wbytes)
				c.Check(n, check.Equals, len(wbytes))
				c.Check(err, check.IsNil)
				pos, err = f.Seek(0, io.SeekStart)
				c.Check(pos, check.Equals, int64(0))
				c.Check(err, check.IsNil)
				buf, err := ioutil.ReadAll(f)
				c.Check(string(buf), check.Equals, string(expect))
				c.Check(err, check.IsNil)
			}
		}(n)
	}
	wg.Wait()

	for n := 0; n < ngoroutines; n++ {
		f, err := s.fs.OpenFile(fmt.Sprintf("random-%d", n), os.O_RDONLY, 0)
		c.Assert(err, check.IsNil)
		f.(*filehandle).inode.(*filenode).waitPrune()
		s.checkMemSize(c, f)
		defer f.Close()
	}

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

func (s *CollectionFSSuite) TestRemove(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("dir0", 0755)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("dir1", 0755)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("dir1/dir2", 0755)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("dir1/dir3", 0755)
	c.Assert(err, check.IsNil)

	err = fs.Remove("dir0")
	c.Check(err, check.IsNil)
	err = fs.Remove("dir0")
	c.Check(err, check.Equals, os.ErrNotExist)

	err = fs.Remove("dir1/dir2/.")
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = fs.Remove("dir1/dir2/..")
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = fs.Remove("dir1")
	c.Check(err, check.Equals, ErrDirectoryNotEmpty)
	err = fs.Remove("dir1/dir2/../../../dir1")
	c.Check(err, check.Equals, ErrDirectoryNotEmpty)
	err = fs.Remove("dir1/dir3/")
	c.Check(err, check.IsNil)
	err = fs.RemoveAll("dir1")
	c.Check(err, check.IsNil)
	err = fs.RemoveAll("dir1")
	c.Check(err, check.IsNil)
}

func (s *CollectionFSSuite) TestRenameError(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("first", 0755)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("first/second", 0755)
	c.Assert(err, check.IsNil)
	f, err := fs.OpenFile("first/second/file", os.O_CREATE|os.O_WRONLY, 0755)
	c.Assert(err, check.IsNil)
	f.Write([]byte{1, 2, 3, 4, 5})
	f.Close()
	err = fs.Rename("first", "first/second/third")
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = fs.Rename("first", "first/third")
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = fs.Rename("first/second", "second")
	c.Check(err, check.IsNil)
	f, err = fs.OpenFile("second/file", 0, 0)
	c.Assert(err, check.IsNil)
	data, err := ioutil.ReadAll(f)
	c.Check(err, check.IsNil)
	c.Check(data, check.DeepEquals, []byte{1, 2, 3, 4, 5})
}

func (s *CollectionFSSuite) TestRenameDirectory(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("foo", 0755)
	c.Assert(err, check.IsNil)
	err = fs.Mkdir("bar", 0755)
	c.Assert(err, check.IsNil)
	err = fs.Rename("bar", "baz")
	c.Check(err, check.IsNil)
	err = fs.Rename("foo", "baz")
	c.Check(err, check.NotNil)
	err = fs.Rename("foo", "baz/")
	c.Check(err, check.IsNil)
	err = fs.Rename("baz/foo", ".")
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = fs.Rename("baz/foo/", ".")
	c.Check(err, check.Equals, ErrInvalidArgument)
}

func (s *CollectionFSSuite) TestRename(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	const (
		outer = 16
		inner = 16
	)
	for i := 0; i < outer; i++ {
		err = fs.Mkdir(fmt.Sprintf("dir%d", i), 0755)
		c.Assert(err, check.IsNil)
		for j := 0; j < inner; j++ {
			err = fs.Mkdir(fmt.Sprintf("dir%d/dir%d", i, j), 0755)
			c.Assert(err, check.IsNil)
			for _, fnm := range []string{
				fmt.Sprintf("dir%d/file%d", i, j),
				fmt.Sprintf("dir%d/dir%d/file%d", i, j, j),
			} {
				f, err := fs.OpenFile(fnm, os.O_CREATE|os.O_WRONLY, 0755)
				c.Assert(err, check.IsNil)
				_, err = f.Write([]byte("beep"))
				c.Assert(err, check.IsNil)
				f.Close()
			}
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < outer; i++ {
		for j := 0; j < inner; j++ {
			wg.Add(1)
			go func(i, j int) {
				defer wg.Done()
				oldname := fmt.Sprintf("dir%d/dir%d/file%d", i, j, j)
				newname := fmt.Sprintf("dir%d/newfile%d", i, inner-j-1)
				_, err := fs.Open(newname)
				c.Check(err, check.Equals, os.ErrNotExist)
				err = fs.Rename(oldname, newname)
				c.Check(err, check.IsNil)
				f, err := fs.Open(newname)
				c.Check(err, check.IsNil)
				f.Close()
			}(i, j)

			wg.Add(1)
			go func(i, j int) {
				defer wg.Done()
				// oldname does not exist
				err := fs.Rename(
					fmt.Sprintf("dir%d/dir%d/missing", i, j),
					fmt.Sprintf("dir%d/dir%d/file%d", outer-i-1, j, j))
				c.Check(err, check.ErrorMatches, `.*does not exist`)

				// newname parent dir does not exist
				err = fs.Rename(
					fmt.Sprintf("dir%d/dir%d", i, j),
					fmt.Sprintf("dir%d/missing/irrelevant", outer-i-1))
				c.Check(err, check.ErrorMatches, `.*does not exist`)

				// oldname parent dir is a file
				err = fs.Rename(
					fmt.Sprintf("dir%d/file%d/patherror", i, j),
					fmt.Sprintf("dir%d/irrelevant", i))
				c.Check(err, check.ErrorMatches, `.*not a directory`)

				// newname parent dir is a file
				err = fs.Rename(
					fmt.Sprintf("dir%d/dir%d/file%d", i, j, j),
					fmt.Sprintf("dir%d/file%d/patherror", i, inner-j-1))
				c.Check(err, check.ErrorMatches, `.*not a directory`)
			}(i, j)
		}
	}
	wg.Wait()

	f, err := fs.OpenFile("dir1/newfile3", 0, 0)
	c.Assert(err, check.IsNil)
	c.Check(f.Size(), check.Equals, int64(4))
	buf, err := ioutil.ReadAll(f)
	c.Check(buf, check.DeepEquals, []byte("beep"))
	c.Check(err, check.IsNil)
	_, err = fs.Open("dir1/dir1/file1")
	c.Check(err, check.Equals, os.ErrNotExist)
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

func (s *CollectionFSSuite) TestPersistEmptyFilesAndDirs(c *check.C) {
	var err error
	s.fs, err = (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	for _, name := range []string{"dir", "dir/zerodir", "empty", "not empty", "not empty/empty", "zero", "zero/zero"} {
		err = s.fs.Mkdir(name, 0755)
		c.Assert(err, check.IsNil)
	}

	expect := map[string][]byte{
		"0":                nil,
		"00":               {},
		"one":              {1},
		"dir/0":            nil,
		"dir/two":          {1, 2},
		"dir/zero":         nil,
		"dir/zerodir/zero": nil,
		"zero/zero/zero":   nil,
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
		_, err = persisted.Open("bogus-" + name)
		c.Check(err, check.NotNil)

		f, err := persisted.Open(name)
		c.Assert(err, check.IsNil)

		if data == nil {
			data = []byte{}
		}
		buf, err := ioutil.ReadAll(f)
		c.Check(err, check.IsNil)
		c.Check(buf, check.DeepEquals, data)
	}

	expectDir := map[string]int{
		"empty":           0,
		"not empty":       1,
		"not empty/empty": 0,
	}
	for name, expectLen := range expectDir {
		_, err := persisted.Open(name + "/bogus")
		c.Check(err, check.NotNil)

		d, err := persisted.Open(name)
		defer d.Close()
		c.Check(err, check.IsNil)
		fi, err := d.Readdir(-1)
		c.Check(err, check.IsNil)
		c.Check(fi, check.HasLen, expectLen)
	}
}

func (s *CollectionFSSuite) TestOpenFileFlags(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	f, err := fs.OpenFile("missing", os.O_WRONLY, 0)
	c.Check(f, check.IsNil)
	c.Check(err, check.ErrorMatches, `file does not exist`)

	f, err = fs.OpenFile("new", os.O_CREATE|os.O_RDONLY, 0)
	c.Assert(err, check.IsNil)
	defer f.Close()
	n, err := f.Write([]byte{1, 2, 3})
	c.Check(n, check.Equals, 0)
	c.Check(err, check.ErrorMatches, `read-only file`)
	n, err = f.Read(make([]byte, 1))
	c.Check(n, check.Equals, 0)
	c.Check(err, check.Equals, io.EOF)
	f, err = fs.OpenFile("new", os.O_RDWR, 0)
	c.Assert(err, check.IsNil)
	defer f.Close()
	_, err = f.Write([]byte{4, 5, 6})
	c.Check(err, check.IsNil)
	fi, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(fi.Size(), check.Equals, int64(3))

	f, err = fs.OpenFile("new", os.O_TRUNC|os.O_RDWR, 0)
	c.Assert(err, check.IsNil)
	defer f.Close()
	pos, err := f.Seek(0, io.SeekEnd)
	c.Check(pos, check.Equals, int64(0))
	c.Check(err, check.IsNil)
	fi, err = f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(fi.Size(), check.Equals, int64(0))
	fs.Remove("new")

	buf := make([]byte, 64)
	f, err = fs.OpenFile("append", os.O_EXCL|os.O_CREATE|os.O_RDWR|os.O_APPEND, 0)
	c.Assert(err, check.IsNil)
	f.Write([]byte{1, 2, 3})
	f.Seek(0, io.SeekStart)
	n, _ = f.Read(buf[:1])
	c.Check(n, check.Equals, 1)
	c.Check(buf[:1], check.DeepEquals, []byte{1})
	pos, err = f.Seek(0, io.SeekCurrent)
	c.Assert(err, check.IsNil)
	c.Check(pos, check.Equals, int64(1))
	f.Write([]byte{4, 5, 6})
	pos, err = f.Seek(0, io.SeekCurrent)
	c.Assert(err, check.IsNil)
	c.Check(pos, check.Equals, int64(6))
	f.Seek(0, io.SeekStart)
	n, err = f.Read(buf)
	c.Check(buf[:n], check.DeepEquals, []byte{1, 2, 3, 4, 5, 6})
	c.Check(err, check.Equals, io.EOF)
	f.Close()

	f, err = fs.OpenFile("append", os.O_RDWR|os.O_APPEND, 0)
	c.Assert(err, check.IsNil)
	pos, err = f.Seek(0, io.SeekCurrent)
	c.Check(pos, check.Equals, int64(0))
	c.Check(err, check.IsNil)
	f.Read(buf[:3])
	pos, _ = f.Seek(0, io.SeekCurrent)
	c.Check(pos, check.Equals, int64(3))
	f.Write([]byte{7, 8, 9})
	pos, err = f.Seek(0, io.SeekCurrent)
	c.Check(err, check.IsNil)
	c.Check(pos, check.Equals, int64(9))
	f.Close()

	f, err = fs.OpenFile("wronly", os.O_CREATE|os.O_WRONLY, 0)
	c.Assert(err, check.IsNil)
	n, err = f.Write([]byte{3, 2, 1})
	c.Check(n, check.Equals, 3)
	c.Check(err, check.IsNil)
	pos, _ = f.Seek(0, io.SeekCurrent)
	c.Check(pos, check.Equals, int64(3))
	pos, _ = f.Seek(0, io.SeekStart)
	c.Check(pos, check.Equals, int64(0))
	n, err = f.Read(buf)
	c.Check(n, check.Equals, 0)
	c.Check(err, check.ErrorMatches, `.*O_WRONLY.*`)
	f, err = fs.OpenFile("wronly", os.O_RDONLY, 0)
	c.Assert(err, check.IsNil)
	n, _ = f.Read(buf)
	c.Check(buf[:n], check.DeepEquals, []byte{3, 2, 1})

	f, err = fs.OpenFile("unsupported", os.O_CREATE|os.O_SYNC, 0)
	c.Check(f, check.IsNil)
	c.Check(err, check.NotNil)

	f, err = fs.OpenFile("append", os.O_RDWR|os.O_WRONLY, 0)
	c.Check(f, check.IsNil)
	c.Check(err, check.ErrorMatches, `invalid flag.*`)
}

func (s *CollectionFSSuite) TestFlushFullBlocksWritingLongFile(c *check.C) {
	defer func(cw, mbs int) {
		concurrentWriters = cw
		maxBlockSize = mbs
	}(concurrentWriters, maxBlockSize)
	concurrentWriters = 2
	maxBlockSize = 1024

	proceed := make(chan struct{})
	var started, concurrent int32
	blk2done := false
	s.kc.onPut = func([]byte) {
		atomic.AddInt32(&concurrent, 1)
		switch atomic.AddInt32(&started, 1) {
		case 1:
			// Wait until block 2 starts and finishes, and block 3 starts
			select {
			case <-proceed:
				c.Check(blk2done, check.Equals, true)
			case <-time.After(time.Second):
				c.Error("timed out")
			}
		case 2:
			time.Sleep(time.Millisecond)
			blk2done = true
		case 3:
			close(proceed)
		default:
			time.Sleep(time.Millisecond)
		}
		c.Check(atomic.AddInt32(&concurrent, -1) < int32(concurrentWriters), check.Equals, true)
	}

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
		for idx, e := range f.(*filehandle).inode.(*filenode).segments {
			switch e.(type) {
			case *memSegment:
				memExtents = append(memExtents, idx)
			}
		}
		return
	}
	f.(*filehandle).inode.(*filenode).waitPrune()
	c.Check(currentMemExtents(), check.HasLen, 1)

	m, err := fs.MarshalManifest(".")
	c.Check(m, check.Matches, `[^:]* 0:50000:50K\n`)
	c.Check(err, check.IsNil)
	c.Check(currentMemExtents(), check.HasLen, 0)
}

// Ensure blocks get flushed to disk if a lot of data is written to
// small files/directories without calling sync().
//
// Write four 512KiB files into each of 256 top-level dirs (total
// 512MiB), calling Flush() every 8 dirs. Ensure memory usage never
// exceeds 24MiB (4 concurrentWriters * 2MiB + 8 unflushed dirs *
// 2MiB).
func (s *CollectionFSSuite) TestFlushAll(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	s.kc.onPut = func([]byte) {
		// discard flushed data -- otherwise the stub will use
		// unlimited memory
		time.Sleep(time.Millisecond)
		s.kc.Lock()
		defer s.kc.Unlock()
		s.kc.blocks = map[string][]byte{}
	}
	for i := 0; i < 256; i++ {
		buf := bytes.NewBuffer(make([]byte, 524288))
		fmt.Fprintf(buf, "test file in dir%d", i)

		dir := fmt.Sprintf("dir%d", i)
		fs.Mkdir(dir, 0755)
		for j := 0; j < 2; j++ {
			f, err := fs.OpenFile(fmt.Sprintf("%s/file%d", dir, j), os.O_WRONLY|os.O_CREATE, 0)
			c.Assert(err, check.IsNil)
			defer f.Close()
			_, err = io.Copy(f, buf)
			c.Assert(err, check.IsNil)
		}

		if i%8 == 0 {
			fs.Flush("", true)
		}

		size := fs.memorySize()
		if !c.Check(size <= 1<<24, check.Equals, true) {
			c.Logf("at dir%d fs.memorySize()=%d", i, size)
			return
		}
	}
}

// Ensure short blocks at the end of a stream don't get flushed by
// Flush(false).
//
// Write 67x 1MiB files to each of 8 dirs, and check that 8 full 64MiB
// blocks have been flushed while 8x 3MiB is still buffered in memory.
func (s *CollectionFSSuite) TestFlushFullBlocksOnly(c *check.C) {
	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	var flushed int64
	s.kc.onPut = func(p []byte) {
		atomic.AddInt64(&flushed, int64(len(p)))
	}

	nDirs := int64(8)
	megabyte := make([]byte, 1<<20)
	for i := int64(0); i < nDirs; i++ {
		dir := fmt.Sprintf("dir%d", i)
		fs.Mkdir(dir, 0755)
		for j := 0; j < 67; j++ {
			f, err := fs.OpenFile(fmt.Sprintf("%s/file%d", dir, j), os.O_WRONLY|os.O_CREATE, 0)
			c.Assert(err, check.IsNil)
			defer f.Close()
			_, err = f.Write(megabyte)
			c.Assert(err, check.IsNil)
		}
	}
	c.Check(fs.memorySize(), check.Equals, int64(nDirs*67<<20))
	c.Check(flushed, check.Equals, int64(0))

	waitForFlush := func(expectUnflushed, expectFlushed int64) {
		for deadline := time.Now().Add(5 * time.Second); fs.memorySize() > expectUnflushed && time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		}
		c.Check(fs.memorySize(), check.Equals, expectUnflushed)
		c.Check(flushed, check.Equals, expectFlushed)
	}

	// Nothing flushed yet
	waitForFlush((nDirs*67)<<20, 0)

	// Flushing a non-empty dir "/" is non-recursive and there are
	// no top-level files, so this has no effect
	fs.Flush("/", false)
	waitForFlush((nDirs*67)<<20, 0)

	// Flush the full block in dir0
	fs.Flush("dir0", false)
	waitForFlush((nDirs*67-64)<<20, 64<<20)

	err = fs.Flush("dir-does-not-exist", false)
	c.Check(err, check.NotNil)

	// Flush full blocks in all dirs
	fs.Flush("", false)
	waitForFlush(nDirs*3<<20, nDirs*64<<20)

	// Flush non-full blocks, too
	fs.Flush("", true)
	waitForFlush(0, nDirs*67<<20)
}

// Even when writing lots of files/dirs from different goroutines, as
// long as Flush(dir,false) is called after writing each file,
// unflushed data should be limited to one full block per
// concurrentWriter, plus one nearly-full block at the end of each
// dir/stream.
func (s *CollectionFSSuite) TestMaxUnflushed(c *check.C) {
	nDirs := int64(8)
	maxUnflushed := (int64(concurrentWriters) + nDirs) << 26

	fs, err := (&Collection{}).FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)

	release := make(chan struct{})
	timeout := make(chan struct{})
	time.AfterFunc(10*time.Second, func() { close(timeout) })
	var putCount, concurrency int64
	var unflushed int64
	s.kc.onPut = func(p []byte) {
		defer atomic.AddInt64(&unflushed, -int64(len(p)))
		cur := atomic.AddInt64(&concurrency, 1)
		defer atomic.AddInt64(&concurrency, -1)
		pc := atomic.AddInt64(&putCount, 1)
		if pc < int64(concurrentWriters) {
			// Block until we reach concurrentWriters, to
			// make sure we're really accepting concurrent
			// writes.
			select {
			case <-release:
			case <-timeout:
				c.Error("timeout")
			}
		} else if pc == int64(concurrentWriters) {
			// Unblock the first N-1 PUT reqs.
			close(release)
		}
		c.Assert(cur <= int64(concurrentWriters), check.Equals, true)
		c.Assert(atomic.LoadInt64(&unflushed) <= maxUnflushed, check.Equals, true)
	}

	var owg sync.WaitGroup
	megabyte := make([]byte, 1<<20)
	for i := int64(0); i < nDirs; i++ {
		dir := fmt.Sprintf("dir%d", i)
		fs.Mkdir(dir, 0755)
		owg.Add(1)
		go func() {
			defer owg.Done()
			defer fs.Flush(dir, true)
			var iwg sync.WaitGroup
			defer iwg.Wait()
			for j := 0; j < 67; j++ {
				iwg.Add(1)
				go func(j int) {
					defer iwg.Done()
					f, err := fs.OpenFile(fmt.Sprintf("%s/file%d", dir, j), os.O_WRONLY|os.O_CREATE, 0)
					c.Assert(err, check.IsNil)
					defer f.Close()
					n, err := f.Write(megabyte)
					c.Assert(err, check.IsNil)
					atomic.AddInt64(&unflushed, int64(n))
					fs.Flush(dir, false)
				}(j)
			}
		}()
	}
	owg.Wait()
	fs.Flush("", true)
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
		". d41d8cd98f00b204e9800998ecf8427e+0 :0:0:foo\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 foo:0:foo\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:foo:foo\n",
		". d41d8cd98f00b204e9800998ecf8427e+1 0:1:foo 1:1:bar\n",
		". d41d8cd98f00b204e9800998ecf8427e+1 0:1:\\056\n",
		". d41d8cd98f00b204e9800998ecf8427e+1 0:1:\\056\\057\\056\n",
		". d41d8cd98f00b204e9800998ecf8427e+1 0:1:.\n",
		". d41d8cd98f00b204e9800998ecf8427e+1 0:1:..\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:..\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/..\n",
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
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:...\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:. 0:0:. 0:0:\\056 0:0:\\056\n",
		". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo/. 0:0:. 0:0:foo\\057bar\\057\\056\n",
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
	fn := f.(*filehandle).inode.(*filenode)
	var memsize int64
	for _, seg := range fn.segments {
		if e, ok := seg.(*memSegment); ok {
			memsize += int64(len(e.buf))
		}
	}
	c.Check(fn.memsize, check.Equals, memsize)
}

type CollectionFSUnitSuite struct{}

var _ = check.Suite(&CollectionFSUnitSuite{})

// expect ~2 seconds to load a manifest with 256K files
func (s *CollectionFSUnitSuite) TestLargeManifest(c *check.C) {
	if testing.Short() {
		c.Skip("slow")
	}

	const (
		dirCount  = 512
		fileCount = 512
	)

	mb := bytes.NewBuffer(make([]byte, 0, 40000000))
	for i := 0; i < dirCount; i++ {
		fmt.Fprintf(mb, "./dir%d", i)
		for j := 0; j <= fileCount; j++ {
			fmt.Fprintf(mb, " %032x+42+A%040x@%08x", j, j, j)
		}
		for j := 0; j < fileCount; j++ {
			fmt.Fprintf(mb, " %d:%d:dir%d/file%d", j*42+21, 42, j, j)
		}
		mb.Write([]byte{'\n'})
	}
	coll := Collection{ManifestText: mb.String()}
	c.Logf("%s built", time.Now())

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	c.Logf("%s Alloc=%d Sys=%d", time.Now(), memstats.Alloc, memstats.Sys)

	f, err := coll.FileSystem(nil, nil)
	c.Check(err, check.IsNil)
	c.Logf("%s loaded", time.Now())
	c.Check(f.Size(), check.Equals, int64(42*dirCount*fileCount))

	for i := 0; i < dirCount; i++ {
		for j := 0; j < fileCount; j++ {
			f.Stat(fmt.Sprintf("./dir%d/dir%d/file%d", i, j, j))
		}
	}
	c.Logf("%s Stat() x %d", time.Now(), dirCount*fileCount)

	runtime.ReadMemStats(&memstats)
	c.Logf("%s Alloc=%d Sys=%d", time.Now(), memstats.Alloc, memstats.Sys)
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
