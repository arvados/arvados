// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"os"
	"sort"
	"strings"
	"syscall"
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/arvados/cgofuse/fuse"
	"github.com/prometheus/client_golang/prometheus"
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
	s.fs.Registry = prometheus.NewRegistry()
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

func (s *FSSuite) TestWriteMetrics(c *C) {
	// Zero to first tick
	previousMetrics := map[string]float64{}
	currentMetrics := map[string]float64{
		// Keep client metrics
		`arvados_keepclient_backend_bytes{direction="out"}`: 1024,
		`arvados_keepclient_backend_bytes{direction="in"}`:  2048,
		`arvados_keepclient_ops{op="put"}`:                  5,
		`arvados_keepclient_ops{op="get"}`:                  10,
		`arvados_keepclient_cache{event="hit"}`:             8,
		`arvados_keepclient_cache{event="miss"}`:            2,
		// FUSE metrics
		`arvados_fuse_bytes{fuseop="read"}`:            2048,
		`arvados_fuse_bytes{fuseop="write"}`:           1024,
		`arvados_fuse_ops{fuseop="read"}`:              5,
		`arvados_fuse_ops{fuseop="write"}`:             3,
		`arvados_fuse_ops{fuseop="getattr"}`:           10,
		`arvados_fuse_seconds_total{fuseop="read"}`:    0.123456,
		`arvados_fuse_seconds_total{fuseop="write"}`:   0.234567,
		`arvados_fuse_seconds_total{fuseop="getattr"}`: 0.045678,
	}

	out1 := &strings.Builder{}
	writeMetrics(out1, currentMetrics, previousMetrics, 1.0)

	lines1 := strings.Split(strings.TrimSpace(out1.String()), "\n")

	expected1 := []string{
		"crunchstat: blkio:0:0 1024 write 2048 read -- interval 1.0000 seconds 1024 write 2048 read",
		"crunchstat: fuseop:create 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:fsync 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:fsyncdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:getattr 10 count 0.045678 time -- interval 1.0000 seconds 10 count 0.045678 time",
		"crunchstat: fuseop:mkdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:mknod 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:open 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:opendir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:read 5 count 0.123456 time -- interval 1.0000 seconds 5 count 0.123456 time",
		"crunchstat: fuseop:readdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:release 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:releasedir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:rename 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:rmdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:truncate 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:unlink 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:utimens 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:write 3 count 0.234567 time -- interval 1.0000 seconds 3 count 0.234567 time",
		"crunchstat: fuseops 3 write 5 read -- interval 1.0000 seconds 3 write 5 read",
		"crunchstat: keepcache 8 hit 2 miss -- interval 1.0000 seconds 8 hit 2 miss",
		"crunchstat: keepcalls 5 put 10 get -- interval 1.0000 seconds 5 put 10 get",
		"crunchstat: net:keep0 1024 tx 2048 rx -- interval 1.0000 seconds 1024 tx 2048 rx",
	}

	sort.Strings(lines1)
	c.Check(lines1, DeepEquals, expected1)

	// First tick to second tick
	previousMetrics = map[string]float64{
		// Keep client metrics
		`arvados_keepclient_backend_bytes{direction="out"}`: 512,
		`arvados_keepclient_backend_bytes{direction="in"}`:  1024,
		`arvados_keepclient_ops{op="put"}`:                  2,
		`arvados_keepclient_ops{op="get"}`:                  5,
		`arvados_keepclient_cache{event="hit"}`:             3,
		`arvados_keepclient_cache{event="miss"}`:            1,
		// FUSE metrics
		`arvados_fuse_bytes{fuseop="read"}`:            1024,
		`arvados_fuse_bytes{fuseop="write"}`:           512,
		`arvados_fuse_ops{fuseop="read"}`:              3,
		`arvados_fuse_ops{fuseop="write"}`:             1,
		`arvados_fuse_ops{fuseop="getattr"}`:           7,
		`arvados_fuse_seconds_total{fuseop="read"}`:    0.100000,
		`arvados_fuse_seconds_total{fuseop="write"}`:   0.200000,
		`arvados_fuse_seconds_total{fuseop="getattr"}`: 0.030000,
	}
	currentMetrics = map[string]float64{
		// Keep client metrics (increased)
		`arvados_keepclient_backend_bytes{direction="out"}`: 2048,
		`arvados_keepclient_backend_bytes{direction="in"}`:  4096,
		`arvados_keepclient_ops{op="put"}`:                  7,
		`arvados_keepclient_ops{op="get"}`:                  15,
		`arvados_keepclient_cache{event="hit"}`:             11,
		`arvados_keepclient_cache{event="miss"}`:            4,
		// FUSE metrics (increased)
		`arvados_fuse_bytes{fuseop="read"}`:            4096,
		`arvados_fuse_bytes{fuseop="write"}`:           2048,
		`arvados_fuse_ops{fuseop="read"}`:              8,
		`arvados_fuse_ops{fuseop="write"}`:             5,
		`arvados_fuse_ops{fuseop="getattr"}`:           15,
		`arvados_fuse_seconds_total{fuseop="read"}`:    0.250000,
		`arvados_fuse_seconds_total{fuseop="write"}`:   0.350000,
		`arvados_fuse_seconds_total{fuseop="getattr"}`: 0.075000,
	}

	out2 := &strings.Builder{}
	writeMetrics(out2, currentMetrics, previousMetrics, 1.0)

	lines2 := strings.Split(strings.TrimSpace(out2.String()), "\n")

	expected2 := []string{
		"crunchstat: blkio:0:0 2048 write 4096 read -- interval 1.0000 seconds 1536 write 3072 read",
		"crunchstat: fuseop:create 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:fsync 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:fsyncdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:getattr 15 count 0.075000 time -- interval 1.0000 seconds 8 count 0.045000 time",
		"crunchstat: fuseop:mkdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:mknod 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:open 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:opendir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:read 8 count 0.250000 time -- interval 1.0000 seconds 5 count 0.150000 time",
		"crunchstat: fuseop:readdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:release 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:releasedir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:rename 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:rmdir 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:truncate 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:unlink 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:utimens 0 count 0.000000 time -- interval 1.0000 seconds 0 count 0.000000 time",
		"crunchstat: fuseop:write 5 count 0.350000 time -- interval 1.0000 seconds 4 count 0.150000 time",
		"crunchstat: fuseops 5 write 8 read -- interval 1.0000 seconds 4 write 5 read",
		"crunchstat: keepcache 11 hit 4 miss -- interval 1.0000 seconds 8 hit 3 miss",
		"crunchstat: keepcalls 7 put 15 get -- interval 1.0000 seconds 5 put 10 get",
		"crunchstat: net:keep0 2048 tx 4096 rx -- interval 1.0000 seconds 1536 tx 3072 rx",
	}

	sort.Strings(lines2)
	c.Check(lines2, DeepEquals, expected2)
}

func (s *FSSuite) TestGatherMetrics(c *C) {
	s.fs.registerMetrics()
	metrics := gatherMetrics(s.fs.Registry)
	c.Check(len(metrics) > 0, Equals, true)
	c.Check(metrics["arvados_fuse_ops{fuseop=\"read\"}"], NotNil)
	c.Check(metrics["arvados_keepclient_backend_bytes{direction=\"in\"}"], NotNil)
	c.Check(metrics["arvados_keepclient_backend_bytes{direction=\"out\"}"], NotNil)
	c.Check(metrics["arvados_keepclient_cache{event=\"hit\"}"], NotNil)
	c.Check(metrics["arvados_keepclient_cache{event=\"miss\"}"], NotNil)
	c.Check(metrics["arvados_keepclient_ops{op=\"get\"}"], NotNil)
	c.Check(metrics["arvados_keepclient_ops{op=\"put\"}"], NotNil)
}
