package main

import (
	"crypto/md5"
	"fmt"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"os"
)

type UploadTestSuite struct{}

// Gocheck boilerplate
var _ = Suite(&UploadTestSuite{})

type KeepTestClient struct {
}

func (k KeepTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	return fmt.Sprintf("%x+%v", md5.Sum(buf), len(buf)), len(buf), nil
}

func (s *TestSuite) TestSimpleUpload(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	ioutil.WriteFile(tmpdir+"/"+"file1.txt", []byte("foo"), 0600)

	str, err := WriteTree(KeepTestClient{}, tmpdir)
	c.Check(err, IsNil)
	c.Check(str, Equals, ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:file1.txt\n")
}

func (s *TestSuite) TestSimpleUploadTwofiles(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	ioutil.WriteFile(tmpdir+"/"+"file1.txt", []byte("foo"), 0600)
	ioutil.WriteFile(tmpdir+"/"+"file2.txt", []byte("bar"), 0600)

	str, err := WriteTree(KeepTestClient{}, tmpdir)
	c.Check(err, IsNil)
	c.Check(str, Equals, ". 3858f62230ac3c915f300c664312c63f+6 0:3:file1.txt 3:3:file2.txt\n")
}

func (s *TestSuite) TestSimpleUploadSubdir(c *C) {
	tmpdir, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(tmpdir)
	}()

	os.Mkdir(tmpdir+"/"+"subdir", 0600)

	ioutil.WriteFile(tmpdir+"/"+"file1.txt", []byte("foo"), 0600)
	ioutil.WriteFile(tmpdir+"/"+"subdir/file2.txt", []byte("bar"), 0600)

	str, err := WriteTree(KeepTestClient{}, tmpdir)
	c.Check(err, IsNil)
	c.Check(str, Equals, `. acbd18db4cc2f85cedef654fccc4a4d8+6 0:3:file1.txt
./subdir acbd18db4cc2f85cedef654fccc4a4d8+6 0:3:file2.txt
`)
}
