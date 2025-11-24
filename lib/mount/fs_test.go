// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"os"
	"strings"
	"syscall"
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/arvados/cgofuse/fuse"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&FSSuite{})

type FSSuite struct {
	fs *keepFS
}

func (s *FSSuite) SetUpTest(c *C) {
	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	c.Assert(err, IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, IsNil)
	s.fs = &keepFS{
		Client:     client,
		KeepClient: kc,
		Logger:     ctxlog.TestLogger(c),
	}
	s.fs.Init()
}

func (s *FSSuite) TearDownTest(c *C) {
	s.fs.Destroy()
}

func (s *FSSuite) TestFuseInterface(c *C) {
	var _ fuse.FileSystemInterface = s.fs
}

func (s *FSSuite) TestOpendir(c *C) {
	errc, fh := s.fs.Opendir("/by_id")
	c.Check(errc, Equals, 0)
	c.Check(fh, Not(Equals), uint64(0))
	c.Check(fh, Not(Equals), invalidFH)
	errc, fh = s.fs.Opendir("/bogus")
	c.Check(errc, Equals, -fuse.ENOENT)
	c.Check(fh, Equals, invalidFH)
}

func (s *FSSuite) TestMknod_ReadOnly(c *C) {
	s.fs.ReadOnly = true
	path := "/by_id/" + arvadostest.FooCollection + "/z"
	errc := s.fs.Mknod(path, syscall.S_IFREG, 0)
	c.Check(errc, Equals, -fuse.EROFS)
}

func (s *FSSuite) TestMknod(c *C) {
	path := "/by_id/" + arvadostest.FooCollection + "/z"
	_, err := s.fs.root.Stat(path)
	c.Assert(err, Equals, os.ErrNotExist)

	// Should return error if mode indicates unsupported file type
	for _, mode := range []uint32{
		syscall.S_IFCHR,
		syscall.S_IFBLK,
		syscall.S_IFIFO,
		syscall.S_IFSOCK,
	} {
		errc := s.fs.Mknod(path, mode, 0)
		c.Check(errc, Equals, -fuse.ENOSYS)
		_, err := s.fs.root.Stat(path)
		c.Check(err, Equals, os.ErrNotExist)
	}

	// Should create file and return 0 if mode indicates regular
	// file
	errc := s.fs.Mknod(path, syscall.S_IFREG|0o644, 0)
	c.Check(errc, Equals, 0)
	_, err = s.fs.root.Stat(path)
	c.Check(err, IsNil)

	// Special case: "Zero file type is equivalent to type
	// S_IFREG." cf. mknod(2)
	errc = s.fs.Mknod(path+"2", 0o644, 0)
	c.Check(errc, Equals, 0)
	_, err = s.fs.root.Stat(path + "2")
	c.Check(err, IsNil)

	// Should return error if target exists
	errc = s.fs.Mknod(path, syscall.S_IFREG|0o644, 0)
	c.Check(errc, Equals, -fuse.EEXIST)
}

func (s *FSSuite) TestFormatMetrics(c *C) {
	// Zero to first tick
	previousMetrics := map[string]float64{}
	currentMetrics := map[string]float64{
		`arvados_fuse_ops{fuseop="read"}`:              5,
		`arvados_fuse_ops{fuseop="write"}`:             3,
		`arvados_fuse_ops{fuseop="getattr"}`:           10,
		`arvados_fuse_seconds_total{fuseop="read"}`:    0.123456,
		`arvados_fuse_seconds_total{fuseop="write"}`:   0.234567,
		`arvados_fuse_seconds_total{fuseop="getattr"}`: 0.045678,
	}
	lines := s.fs.formatMetrics(currentMetrics, previousMetrics, 1.0)

	c.Check(len(lines), Equals, 19) // 1 summary + 18 operations
	c.Check(lines[0], Equals, "crunchstat: fuseops 3 write 5 read -- interval 1.0000 seconds 3 write 5 read")
	c.Check(lines[1], Equals, "crunchstat: fuseop:getattr 10 count 0.045678 time -- interval 1.0000 seconds 10 count 0.045678 time")

	// Check read and write lines
	var readLine, writeLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "crunchstat: fuseop:read ") {
			readLine = line
		}
		if strings.HasPrefix(line, "crunchstat: fuseop:write ") {
			writeLine = line
		}
	}
	c.Check(readLine, Equals, "crunchstat: fuseop:read 5 count 0.123456 time -- interval 1.0000 seconds 5 count 0.123456 time")
	c.Check(writeLine, Equals, "crunchstat: fuseop:write 3 count 0.234567 time -- interval 1.0000 seconds 3 count 0.234567 time")

	// First tick to second tick
	previousMetrics = map[string]float64{
		`arvados_fuse_ops{fuseop="read"}`:              3,
		`arvados_fuse_ops{fuseop="write"}`:             1,
		`arvados_fuse_ops{fuseop="getattr"}`:           7,
		`arvados_fuse_seconds_total{fuseop="read"}`:    0.100000,
		`arvados_fuse_seconds_total{fuseop="write"}`:   0.200000,
		`arvados_fuse_seconds_total{fuseop="getattr"}`: 0.030000,
	}
	currentMetrics = map[string]float64{
		`arvados_fuse_ops{fuseop="read"}`:              8,
		`arvados_fuse_ops{fuseop="write"}`:             5,
		`arvados_fuse_ops{fuseop="getattr"}`:           15,
		`arvados_fuse_seconds_total{fuseop="read"}`:    0.250000,
		`arvados_fuse_seconds_total{fuseop="write"}`:   0.350000,
		`arvados_fuse_seconds_total{fuseop="getattr"}`: 0.075000,
	}
	lines = s.fs.formatMetrics(currentMetrics, previousMetrics, 1.0)

	// Check summary line shows totals and deltas
	c.Check(lines[0], Equals, "crunchstat: fuseops 5 write 8 read -- interval 1.0000 seconds 4 write 5 read")

	// Check individual operations and deltas
	for _, line := range lines {
		if strings.HasPrefix(line, "crunchstat: fuseop:read ") {
			c.Check(line, Equals, "crunchstat: fuseop:read 8 count 0.250000 time -- interval 1.0000 seconds 5 count 0.150000 time")
		}
		if strings.HasPrefix(line, "crunchstat: fuseop:write ") {
			c.Check(line, Equals, "crunchstat: fuseop:write 5 count 0.350000 time -- interval 1.0000 seconds 4 count 0.150000 time")
		}
		if strings.HasPrefix(line, "crunchstat: fuseop:getattr ") {
			c.Check(line, Equals, "crunchstat: fuseop:getattr 15 count 0.075000 time -- interval 1.0000 seconds 8 count 0.045000 time")
		}
	}
}
