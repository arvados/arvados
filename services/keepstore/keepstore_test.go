// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	. "gopkg.in/check.v1"
)

func TestGocheck(t *testing.T) {
	TestingT(t)
}

const (
	fooHash = "acbd18db4cc2f85cedef654fccc4a4d8"
	barHash = "37b51d194a7513e45b56f6524f2d51f2"
)

var testServiceURL = func() arvados.URL {
	return arvados.URL{Host: "localhost:12345", Scheme: "http"}
}()

func authContext(token string) context.Context {
	return auth.NewContext(context.TODO(), &auth.Credentials{Tokens: []string{token}})
}

func testCluster(t TB) *arvados.Cluster {
	cfg, err := config.NewLoader(bytes.NewBufferString("Clusters: {zzzzz: {}}"), ctxlog.TestLogger(t)).Load()
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		t.Fatal(err)
	}
	cluster.SystemRootToken = arvadostest.SystemRootToken
	cluster.ManagementToken = arvadostest.ManagementToken
	return cluster
}

func testKeepstore(t TB, cluster *arvados.Cluster, reg *prometheus.Registry) (*keepstore, context.CancelFunc) {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx = ctxlog.Context(ctx, ctxlog.TestLogger(t))
	ks, err := newKeepstore(ctx, cluster, cluster.SystemRootToken, reg, testServiceURL)
	if err != nil {
		t.Fatal(err)
	}
	return ks, cancel
}

var _ = Suite(&keepstoreSuite{})

type keepstoreSuite struct {
	cluster *arvados.Cluster
}

func (s *keepstoreSuite) SetUpTest(c *C) {
	s.cluster = testCluster(c)
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "stub"},
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "stub"},
	}
}

func (s *keepstoreSuite) TestBlockRead_ChecksumMismatch(c *C) {
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()

	ctx := authContext(arvadostest.ActiveTokenV2)

	fooHash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	err := ks.mountsW[0].BlockWrite(ctx, fooHash, []byte("bar"))
	c.Assert(err, IsNil)

	_, err = ks.BlockWrite(ctx, arvados.BlockWriteOptions{
		Hash: fooHash,
		Data: []byte("foo"),
	})
	c.Check(err, ErrorMatches, "hash collision")

	buf := bytes.NewBuffer(nil)
	_, err = ks.BlockRead(ctx, arvados.BlockReadOptions{
		Locator: ks.signLocator(arvadostest.ActiveTokenV2, fooHash+"+3"),
		WriteTo: buf,
	})
	c.Check(err, ErrorMatches, "checksum mismatch in stored data")
	c.Check(buf.String(), Not(Equals), "foo")
	c.Check(buf.Len() < 3, Equals, true)

	err = ks.mountsW[1].BlockWrite(ctx, fooHash, []byte("foo"))
	c.Assert(err, IsNil)

	buf = bytes.NewBuffer(nil)
	_, err = ks.BlockRead(ctx, arvados.BlockReadOptions{
		Locator: ks.signLocator(arvadostest.ActiveTokenV2, fooHash+"+3"),
		WriteTo: buf,
	})
	c.Check(err, ErrorMatches, "checksum mismatch in stored data")
	c.Check(buf.Len() < 3, Equals, true)
}

func (s *keepstoreSuite) TestBlockReadWrite_SigningDisabled(c *C) {
	origKey := s.cluster.Collections.BlobSigningKey
	s.cluster.Collections.BlobSigning = false
	s.cluster.Collections.BlobSigningKey = ""
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()

	resp, err := ks.BlockWrite(authContext("abcde"), arvados.BlockWriteOptions{
		Hash: fooHash,
		Data: []byte("foo"),
	})
	c.Assert(err, IsNil)
	c.Check(resp.Locator, Equals, fooHash+"+3")
	locUnsigned := resp.Locator
	ttl := time.Hour
	locSigned := arvados.SignLocator(locUnsigned, arvadostest.ActiveTokenV2, time.Now().Add(ttl), ttl, []byte(origKey))
	c.Assert(locSigned, Not(Equals), locUnsigned)

	for _, locator := range []string{locUnsigned, locSigned} {
		for _, token := range []string{"", "xyzzy", arvadostest.ActiveTokenV2} {
			c.Logf("=== locator %q token %q", locator, token)
			ctx := authContext(token)
			buf := bytes.NewBuffer(nil)
			_, err := ks.BlockRead(ctx, arvados.BlockReadOptions{
				Locator: locator,
				WriteTo: buf,
			})
			c.Check(err, IsNil)
			c.Check(buf.String(), Equals, "foo")
		}
	}
}

func (s *keepstoreSuite) TestBlockRead_OrderedByStorageClassPriority(c *C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-111111111111111": {
			Driver:         "stub",
			Replication:    1,
			StorageClasses: map[string]bool{"class1": true}},
		"zzzzz-nyw5e-222222222222222": {
			Driver:         "stub",
			Replication:    1,
			StorageClasses: map[string]bool{"class2": true, "class3": true}},
	}

	// "foobar" is just some data that happens to result in
	// rendezvous order {111, 222}
	data := []byte("foobar")
	hash := fmt.Sprintf("%x", md5.Sum(data))

	for _, trial := range []struct {
		priority1 int // priority of class1, thus vol1
		priority2 int // priority of class2
		priority3 int // priority of class3 (vol2 priority will be max(priority2, priority3))
		expectLog string
	}{
		{100, 50, 50, "111 read 385\n"},              // class1 has higher priority => try vol1 first, no need to try vol2
		{100, 100, 100, "111 read 385\n"},            // same priority, vol2 is first in rendezvous order => try vol1 first and succeed
		{66, 99, 33, "222 read 385\n111 read 385\n"}, // class2 has higher priority => try vol2 first, then try vol1
		{66, 33, 99, "222 read 385\n111 read 385\n"}, // class3 has highest priority => vol2 has highest => try vol2 first, then try vol1
	} {
		c.Logf("=== %+v", trial)

		s.cluster.StorageClasses = map[string]arvados.StorageClassConfig{
			"class1": {Priority: trial.priority1},
			"class2": {Priority: trial.priority2},
			"class3": {Priority: trial.priority3},
		}
		ks, cancel := testKeepstore(c, s.cluster, nil)
		defer cancel()

		ctx := authContext(arvadostest.ActiveTokenV2)
		resp, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
			Hash:           hash,
			Data:           data,
			StorageClasses: []string{"class1"},
		})
		c.Assert(err, IsNil)

		// Combine logs into one. (We only want the logs from
		// the BlockRead below, not from BlockWrite above.)
		stubLog := &stubLog{}
		for _, mnt := range ks.mounts {
			mnt.volume.(*stubVolume).stubLog = stubLog
		}

		n, err := ks.BlockRead(ctx, arvados.BlockReadOptions{
			Locator: resp.Locator,
			WriteTo: io.Discard,
		})
		c.Assert(n, Equals, len(data))
		c.Assert(err, IsNil)
		c.Check(stubLog.String(), Equals, trial.expectLog)
	}
}

func (s *keepstoreSuite) TestBlockWrite_NoWritableVolumes(c *C) {
	for uuid, v := range s.cluster.Volumes {
		v.ReadOnly = true
		s.cluster.Volumes[uuid] = v
	}
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()
	for _, mnt := range ks.mounts {
		mnt.volume.(*stubVolume).blockWrite = func(context.Context, string, []byte) error {
			c.Error("volume BlockWrite called")
			return errors.New("fail")
		}
	}
	ctx := authContext(arvadostest.ActiveTokenV2)

	_, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
		Hash: fooHash,
		Data: []byte("foo")})
	c.Check(err, NotNil)
	c.Check(err.(interface{ HTTPStatus() int }).HTTPStatus(), Equals, http.StatusInsufficientStorage)
}

func (s *keepstoreSuite) TestBlockWrite_MultipleStorageClasses(c *C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-111111111111111": {
			Driver:         "stub",
			Replication:    1,
			StorageClasses: map[string]bool{"class1": true}},
		"zzzzz-nyw5e-121212121212121": {
			Driver:         "stub",
			Replication:    1,
			StorageClasses: map[string]bool{"class1": true, "class2": true}},
		"zzzzz-nyw5e-222222222222222": {
			Driver:         "stub",
			Replication:    1,
			StorageClasses: map[string]bool{"class2": true}},
	}

	// testData is a block that happens to have rendezvous order 111, 121, 222
	testData := []byte("qux")
	testHash := fmt.Sprintf("%x+%d", md5.Sum(testData), len(testData))

	s.cluster.StorageClasses = map[string]arvados.StorageClassConfig{
		"class1": {},
		"class2": {},
		"class3": {},
	}

	ctx := authContext(arvadostest.ActiveTokenV2)
	for idx, trial := range []struct {
		classes   string // desired classes
		expectLog string
	}{
		{"class1", "" +
			"111 read d85\n" +
			"121 read d85\n" +
			"111 write d85\n" +
			"111 read d85\n" +
			"111 touch d85\n"},
		{"class2", "" +
			"121 read d85\n" + // write#1
			"222 read d85\n" +
			"121 write d85\n" +
			"121 read d85\n" + // write#2
			"121 touch d85\n"},
		{"class1,class2", "" +
			"111 read d85\n" + // write#1
			"121 read d85\n" +
			"222 read d85\n" +
			"121 write d85\n" +
			"111 write d85\n" +
			"111 read d85\n" + // write#2
			"111 touch d85\n" +
			"121 read d85\n" +
			"121 touch d85\n"},
		{"class1,class2,class404", "" +
			"111 read d85\n" + // write#1
			"121 read d85\n" +
			"222 read d85\n" +
			"121 write d85\n" +
			"111 write d85\n" +
			"111 read d85\n" + // write#2
			"111 touch d85\n" +
			"121 read d85\n" +
			"121 touch d85\n"},
	} {
		c.Logf("=== %d: %+v", idx, trial)

		ks, cancel := testKeepstore(c, s.cluster, nil)
		defer cancel()
		stubLog := &stubLog{}
		for _, mnt := range ks.mounts {
			mnt.volume.(*stubVolume).stubLog = stubLog
		}

		// Check that we chose the right block data
		rvz := ks.rendezvous(testHash, ks.mountsW)
		c.Assert(rvz[0].UUID[24:], Equals, "111")
		c.Assert(rvz[1].UUID[24:], Equals, "121")
		c.Assert(rvz[2].UUID[24:], Equals, "222")

		for i := 0; i < 2; i++ {
			_, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
				Hash:           testHash,
				Data:           testData,
				StorageClasses: strings.Split(trial.classes, ","),
			})
			c.Check(err, IsNil)
		}
		c.Check(stubLog.String(), Equals, trial.expectLog)
	}
}

func (s *keepstoreSuite) TestBlockTrash(c *C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "stub"},
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "stub"},
		"zzzzz-nyw5e-222222222222222": {Replication: 1, Driver: "stub", ReadOnly: true},
		"zzzzz-nyw5e-333333333333333": {Replication: 1, Driver: "stub", ReadOnly: true, AllowTrashWhenReadOnly: true},
	}
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()

	var vol []*stubVolume
	for _, mount := range ks.mountsR {
		vol = append(vol, mount.volume.(*stubVolume))
	}
	sort.Slice(vol, func(i, j int) bool {
		return vol[i].params.UUID < vol[j].params.UUID
	})

	ctx := context.Background()
	loc := fooHash + "+3"
	tOld := time.Now().Add(-s.cluster.Collections.BlobSigningTTL.Duration() - time.Second)

	clear := func() {
		for _, vol := range vol {
			err := vol.BlockTrash(fooHash)
			if !os.IsNotExist(err) {
				c.Assert(err, IsNil)
			}
		}
	}
	writeit := func(volidx int) {
		err := vol[volidx].BlockWrite(ctx, fooHash, []byte("foo"))
		c.Assert(err, IsNil)
		err = vol[volidx].blockTouchWithTime(fooHash, tOld)
		c.Assert(err, IsNil)
	}
	trashit := func() error {
		return ks.BlockTrash(ctx, loc)
	}
	checkexists := func(volidx int) bool {
		err := vol[volidx].BlockRead(ctx, fooHash, brdiscard)
		if !os.IsNotExist(err) {
			c.Check(err, IsNil)
		}
		return err == nil
	}

	clear()
	c.Check(trashit(), Equals, os.ErrNotExist)

	// one old replica => trash it
	clear()
	writeit(0)
	c.Check(trashit(), IsNil)
	c.Check(checkexists(0), Equals, false)

	// one old replica + one new replica => keep new, trash old
	clear()
	writeit(0)
	writeit(1)
	c.Check(vol[1].blockTouchWithTime(fooHash, time.Now()), IsNil)
	c.Check(trashit(), IsNil)
	c.Check(checkexists(0), Equals, false)
	c.Check(checkexists(1), Equals, true)

	// two old replicas => trash both
	clear()
	writeit(0)
	writeit(1)
	c.Check(trashit(), IsNil)
	c.Check(checkexists(0), Equals, false)
	c.Check(checkexists(1), Equals, false)

	// four old replicas => trash all except readonly volume with
	// AllowTrashWhenReadOnly==false
	clear()
	writeit(0)
	writeit(1)
	writeit(2)
	writeit(3)
	c.Check(trashit(), IsNil)
	c.Check(checkexists(0), Equals, false)
	c.Check(checkexists(1), Equals, false)
	c.Check(checkexists(2), Equals, true)
	c.Check(checkexists(3), Equals, false)

	// two old replicas but one returns an error => return the
	// only non-404 backend error
	clear()
	vol[0].blockTrash = func(hash string) error {
		return errors.New("fake error")
	}
	writeit(0)
	writeit(3)
	c.Check(trashit(), ErrorMatches, "fake error")
	c.Check(checkexists(0), Equals, true)
	c.Check(checkexists(1), Equals, false)
	c.Check(checkexists(2), Equals, false)
	c.Check(checkexists(3), Equals, false)
}

func (s *keepstoreSuite) TestBlockWrite_OnlyOneBuffer(c *C) {
	s.cluster.API.MaxKeepBlobBuffers = 1
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()
	ok := make(chan struct{})
	go func() {
		defer close(ok)
		ctx := authContext(arvadostest.ActiveTokenV2)
		_, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
			Hash: fooHash,
			Data: []byte("foo")})
		c.Check(err, IsNil)
	}()
	select {
	case <-ok:
	case <-time.After(time.Second):
		c.Fatal("PUT deadlocks with MaxKeepBlobBuffers==1")
	}
}

func (s *keepstoreSuite) TestBufferPoolLeak(c *C) {
	s.cluster.API.MaxKeepBlobBuffers = 4
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()

	ctx := authContext(arvadostest.ActiveTokenV2)
	var wg sync.WaitGroup
	for range make([]int, 20) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
				Hash: fooHash,
				Data: []byte("foo")})
			c.Check(err, IsNil)
			_, err = ks.BlockRead(ctx, arvados.BlockReadOptions{
				Locator: resp.Locator,
				WriteTo: io.Discard})
			c.Check(err, IsNil)
		}()
	}
	ok := make(chan struct{})
	go func() {
		wg.Wait()
		close(ok)
	}()
	select {
	case <-ok:
	case <-time.After(time.Second):
		c.Fatal("read/write sequence deadlocks, likely buffer pool leak")
	}
}

func (s *keepstoreSuite) TestPutStorageClasses(c *C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "stub"}, // "default" is implicit
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "stub", StorageClasses: map[string]bool{"special": true, "extra": true}},
		"zzzzz-nyw5e-222222222222222": {Replication: 1, Driver: "stub", StorageClasses: map[string]bool{"readonly": true}, ReadOnly: true},
	}
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()
	ctx := authContext(arvadostest.ActiveTokenV2)

	for _, trial := range []struct {
		ask            []string
		expectReplicas int
		expectClasses  map[string]int
	}{
		{nil,
			1,
			map[string]int{"default": 1}},
		{[]string{},
			1,
			map[string]int{"default": 1}},
		{[]string{"default"},
			1,
			map[string]int{"default": 1}},
		{[]string{"default", "default"},
			1,
			map[string]int{"default": 1}},
		{[]string{"special"},
			1,
			map[string]int{"extra": 1, "special": 1}},
		{[]string{"special", "readonly"},
			1,
			map[string]int{"extra": 1, "special": 1}},
		{[]string{"special", "nonexistent"},
			1,
			map[string]int{"extra": 1, "special": 1}},
		{[]string{"extra", "special"},
			1,
			map[string]int{"extra": 1, "special": 1}},
		{[]string{"default", "special"},
			2,
			map[string]int{"default": 1, "extra": 1, "special": 1}},
	} {
		c.Logf("success case %#v", trial)
		resp, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
			Hash:           fooHash,
			Data:           []byte("foo"),
			StorageClasses: trial.ask,
		})
		if !c.Check(err, IsNil) {
			continue
		}
		c.Check(resp.Replicas, Equals, trial.expectReplicas)
		if len(trial.expectClasses) == 0 {
			// any non-empty value is correct
			c.Check(resp.StorageClasses, Not(HasLen), 0)
		} else {
			c.Check(resp.StorageClasses, DeepEquals, trial.expectClasses)
		}
	}

	for _, ask := range [][]string{
		{"doesnotexist"},
		{"doesnotexist", "readonly"},
		{"readonly"},
	} {
		c.Logf("failure case %s", ask)
		_, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{
			Hash:           fooHash,
			Data:           []byte("foo"),
			StorageClasses: ask,
		})
		c.Check(err, NotNil)
	}
}

func (s *keepstoreSuite) TestUntrashHandlerWithNoWritableVolumes(c *C) {
	for uuid, v := range s.cluster.Volumes {
		v.ReadOnly = true
		s.cluster.Volumes[uuid] = v
	}
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()

	for _, mnt := range ks.mounts {
		err := mnt.BlockWrite(context.Background(), fooHash, []byte("foo"))
		c.Assert(err, IsNil)
		err = mnt.BlockRead(context.Background(), fooHash, brdiscard)
		c.Assert(err, IsNil)
	}

	err := ks.BlockUntrash(context.Background(), fooHash)
	c.Check(os.IsNotExist(err), Equals, true)

	for _, mnt := range ks.mounts {
		err := mnt.BlockRead(context.Background(), fooHash, brdiscard)
		c.Assert(err, IsNil)
	}
}

func (s *keepstoreSuite) TestBlockWrite_SkipReadOnly(c *C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "stub"},
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "stub", ReadOnly: true},
		"zzzzz-nyw5e-222222222222222": {Replication: 1, Driver: "stub", ReadOnly: true, AllowTrashWhenReadOnly: true},
	}
	ks, cancel := testKeepstore(c, s.cluster, nil)
	defer cancel()
	ctx := authContext(arvadostest.ActiveTokenV2)

	for i := range make([]byte, 32) {
		data := []byte(fmt.Sprintf("block %d", i))
		_, err := ks.BlockWrite(ctx, arvados.BlockWriteOptions{Data: data})
		c.Assert(err, IsNil)
	}
	c.Check(ks.mounts["zzzzz-nyw5e-000000000000000"].volume.(*stubVolume).stubLog.String(), Matches, "(?ms).*write.*")
	c.Check(ks.mounts["zzzzz-nyw5e-111111111111111"].volume.(*stubVolume).stubLog.String(), HasLen, 0)
	c.Check(ks.mounts["zzzzz-nyw5e-222222222222222"].volume.(*stubVolume).stubLog.String(), HasLen, 0)
}

func (s *keepstoreSuite) TestParseLocator(c *C) {
	for _, trial := range []struct {
		locator string
		ok      bool
		expect  locatorInfo
	}{
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ok: true},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1234",
			ok: true, expect: locatorInfo{size: 1234}},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1234+Abcdef@abcdef",
			ok: true, expect: locatorInfo{size: 1234, signed: true}},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1234+Rzzzzz-abcdef",
			ok: true, expect: locatorInfo{size: 1234, remote: true}},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+12345+Zexample+Rzzzzz-abcdef",
			ok: true, expect: locatorInfo{size: 12345, remote: true}},
		// invalid: hash length != 32
		{locator: "",
			ok: false},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ok: false},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1234",
			ok: false},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabb",
			ok: false},
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabb+1234",
			ok: false},
		// invalid: first hint is not size
		{locator: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+Abcdef+1234",
			ok: false},
	} {
		c.Logf("=== %s", trial.locator)
		li, err := parseLocator(trial.locator)
		if !trial.ok {
			c.Check(err, NotNil)
			continue
		}
		c.Check(err, IsNil)
		c.Check(li.hash, Equals, trial.locator[:32])
		c.Check(li.size, Equals, trial.expect.size)
		c.Check(li.signed, Equals, trial.expect.signed)
		c.Check(li.remote, Equals, trial.expect.remote)
	}
}

func init() {
	driver["stub"] = func(params newVolumeParams) (volume, error) {
		v := &stubVolume{
			params:  params,
			data:    make(map[string]stubData),
			stubLog: &stubLog{},
		}
		return v, nil
	}
}

type stubLog struct {
	sync.Mutex
	bytes.Buffer
}

func (sl *stubLog) Printf(format string, args ...interface{}) {
	if sl == nil {
		return
	}
	sl.Lock()
	defer sl.Unlock()
	fmt.Fprintf(sl, format+"\n", args...)
}

type stubData struct {
	mtime time.Time
	data  []byte
	trash time.Time
}

type stubVolume struct {
	params  newVolumeParams
	data    map[string]stubData
	stubLog *stubLog
	mtx     sync.Mutex

	// The following funcs enable tests to insert delays and
	// failures. Each volume operation begins by calling the
	// corresponding func (if non-nil). If the func returns an
	// error, that error is returned to caller. Otherwise, the
	// stub continues normally.
	blockRead    func(ctx context.Context, hash string, writeTo io.WriterAt) error
	blockWrite   func(ctx context.Context, hash string, data []byte) error
	deviceID     func() string
	blockTouch   func(hash string) error
	blockTrash   func(hash string) error
	blockUntrash func(hash string) error
	index        func(ctx context.Context, prefix string, writeTo io.Writer) error
	mtime        func(hash string) (time.Time, error)
	emptyTrash   func()
}

func (v *stubVolume) log(op, hash string) {
	// Note this intentionally crashes if UUID or hash is short --
	// if keepstore ever does that, tests should fail.
	v.stubLog.Printf("%s %s %s", v.params.UUID[24:27], op, hash[:3])
}

func (v *stubVolume) BlockRead(ctx context.Context, hash string, writeTo io.WriterAt) error {
	v.log("read", hash)
	if v.blockRead != nil {
		err := v.blockRead(ctx, hash, writeTo)
		if err != nil {
			return err
		}
	}
	v.mtx.Lock()
	ent, ok := v.data[hash]
	v.mtx.Unlock()
	if !ok || !ent.trash.IsZero() {
		return os.ErrNotExist
	}
	wrote := 0
	for writesize := 1000; wrote < len(ent.data); writesize = writesize * 2 {
		data := ent.data[wrote:]
		if len(data) > writesize {
			data = data[:writesize]
		}
		n, err := writeTo.WriteAt(data, int64(wrote))
		wrote += n
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *stubVolume) BlockWrite(ctx context.Context, hash string, data []byte) error {
	v.log("write", hash)
	if v.blockWrite != nil {
		if err := v.blockWrite(ctx, hash, data); err != nil {
			return err
		}
	}
	v.mtx.Lock()
	defer v.mtx.Unlock()
	v.data[hash] = stubData{
		mtime: time.Now(),
		data:  append([]byte(nil), data...),
	}
	return nil
}

func (v *stubVolume) DeviceID() string {
	return fmt.Sprintf("%p", v)
}

func (v *stubVolume) BlockTouch(hash string) error {
	v.log("touch", hash)
	if v.blockTouch != nil {
		if err := v.blockTouch(hash); err != nil {
			return err
		}
	}
	v.mtx.Lock()
	defer v.mtx.Unlock()
	ent, ok := v.data[hash]
	if !ok || !ent.trash.IsZero() {
		return os.ErrNotExist
	}
	ent.mtime = time.Now()
	v.data[hash] = ent
	return nil
}

// Set mtime to the (presumably old) specified time.
func (v *stubVolume) blockTouchWithTime(hash string, t time.Time) error {
	v.log("touchwithtime", hash)
	v.mtx.Lock()
	defer v.mtx.Unlock()
	ent, ok := v.data[hash]
	if !ok {
		return os.ErrNotExist
	}
	ent.mtime = t
	v.data[hash] = ent
	return nil
}

func (v *stubVolume) BlockTrash(hash string) error {
	v.log("trash", hash)
	if v.blockTrash != nil {
		if err := v.blockTrash(hash); err != nil {
			return err
		}
	}
	v.mtx.Lock()
	defer v.mtx.Unlock()
	ent, ok := v.data[hash]
	if !ok || !ent.trash.IsZero() {
		return os.ErrNotExist
	}
	ent.trash = time.Now().Add(v.params.Cluster.Collections.BlobTrashLifetime.Duration())
	v.data[hash] = ent
	return nil
}

func (v *stubVolume) BlockUntrash(hash string) error {
	v.log("untrash", hash)
	if v.blockUntrash != nil {
		if err := v.blockUntrash(hash); err != nil {
			return err
		}
	}
	v.mtx.Lock()
	defer v.mtx.Unlock()
	ent, ok := v.data[hash]
	if !ok || ent.trash.IsZero() {
		return os.ErrNotExist
	}
	ent.trash = time.Time{}
	v.data[hash] = ent
	return nil
}

func (v *stubVolume) Index(ctx context.Context, prefix string, writeTo io.Writer) error {
	v.stubLog.Printf("%s index %s", v.params.UUID, prefix)
	if v.index != nil {
		if err := v.index(ctx, prefix, writeTo); err != nil {
			return err
		}
	}
	buf := &bytes.Buffer{}
	v.mtx.Lock()
	for hash, ent := range v.data {
		if ent.trash.IsZero() && strings.HasPrefix(hash, prefix) {
			fmt.Fprintf(buf, "%s+%d %d\n", hash, len(ent.data), ent.mtime.UnixNano())
		}
	}
	v.mtx.Unlock()
	_, err := io.Copy(writeTo, buf)
	return err
}

func (v *stubVolume) Mtime(hash string) (time.Time, error) {
	v.log("mtime", hash)
	if v.mtime != nil {
		if t, err := v.mtime(hash); err != nil {
			return t, err
		}
	}
	v.mtx.Lock()
	defer v.mtx.Unlock()
	ent, ok := v.data[hash]
	if !ok || !ent.trash.IsZero() {
		return time.Time{}, os.ErrNotExist
	}
	return ent.mtime, nil
}

func (v *stubVolume) EmptyTrash() {
	v.stubLog.Printf("%s emptytrash", v.params.UUID)
	v.mtx.Lock()
	defer v.mtx.Unlock()
	for hash, ent := range v.data {
		if !ent.trash.IsZero() && time.Now().After(ent.trash) {
			delete(v.data, hash)
		}
	}
}
