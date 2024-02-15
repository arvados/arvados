// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

type testableUnixVolume struct {
	unixVolume
	t TB
}

func (v *testableUnixVolume) TouchWithDate(locator string, lastPut time.Time) {
	err := syscall.Utime(v.blockPath(locator), &syscall.Utimbuf{Actime: lastPut.Unix(), Modtime: lastPut.Unix()})
	if err != nil {
		v.t.Fatal(err)
	}
}

func (v *testableUnixVolume) Teardown() {
	if err := os.RemoveAll(v.Root); err != nil {
		v.t.Error(err)
	}
}

func (v *testableUnixVolume) ReadWriteOperationLabelValues() (r, w string) {
	return "open", "create"
}

var _ = check.Suite(&unixVolumeSuite{})

type unixVolumeSuite struct {
	params  newVolumeParams
	volumes []*testableUnixVolume
}

func (s *unixVolumeSuite) SetUpTest(c *check.C) {
	logger := ctxlog.TestLogger(c)
	reg := prometheus.NewRegistry()
	s.params = newVolumeParams{
		UUID:        "zzzzz-nyw5e-999999999999999",
		Cluster:     testCluster(c),
		Logger:      logger,
		MetricsVecs: newVolumeMetricsVecs(reg),
		BufferPool:  newBufferPool(logger, 8, reg),
	}
}

func (s *unixVolumeSuite) TearDownTest(c *check.C) {
	for _, v := range s.volumes {
		v.Teardown()
	}
}

func (s *unixVolumeSuite) newTestableUnixVolume(c *check.C, params newVolumeParams, serialize bool) *testableUnixVolume {
	d, err := ioutil.TempDir("", "volume_test")
	c.Check(err, check.IsNil)
	var locker sync.Locker
	if serialize {
		locker = &sync.Mutex{}
	}
	v := &testableUnixVolume{
		unixVolume: unixVolume{
			Root:       d,
			locker:     locker,
			uuid:       params.UUID,
			cluster:    params.Cluster,
			logger:     params.Logger,
			volume:     params.ConfigVolume,
			metrics:    params.MetricsVecs,
			bufferPool: params.BufferPool,
		},
		t: c,
	}
	c.Check(v.check(), check.IsNil)
	s.volumes = append(s.volumes, v)
	return v
}

func (s *unixVolumeSuite) TestUnixVolumeWithGenericTests(c *check.C) {
	DoGenericVolumeTests(c, false, func(t TB, params newVolumeParams) TestableVolume {
		return s.newTestableUnixVolume(c, params, false)
	})
}

func (s *unixVolumeSuite) TestUnixVolumeWithGenericTests_ReadOnly(c *check.C) {
	DoGenericVolumeTests(c, true, func(t TB, params newVolumeParams) TestableVolume {
		return s.newTestableUnixVolume(c, params, false)
	})
}

func (s *unixVolumeSuite) TestUnixVolumeWithGenericTests_Serialized(c *check.C) {
	DoGenericVolumeTests(c, false, func(t TB, params newVolumeParams) TestableVolume {
		return s.newTestableUnixVolume(c, params, true)
	})
}

func (s *unixVolumeSuite) TestUnixVolumeWithGenericTests_Readonly_Serialized(c *check.C) {
	DoGenericVolumeTests(c, true, func(t TB, params newVolumeParams) TestableVolume {
		return s.newTestableUnixVolume(c, params, true)
	})
}

func (s *unixVolumeSuite) TestGetNotFound(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, true)
	defer v.Teardown()
	v.BlockWrite(context.Background(), TestHash, TestBlock)

	buf := bytes.NewBuffer(nil)
	_, err := v.BlockRead(context.Background(), TestHash2, buf)
	switch {
	case os.IsNotExist(err):
		break
	case err == nil:
		c.Errorf("Read should have failed, returned %+q", buf.Bytes())
	default:
		c.Errorf("Read expected ErrNotExist, got: %s", err)
	}
}

func (s *unixVolumeSuite) TestPut(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, false)
	defer v.Teardown()

	err := v.BlockWrite(context.Background(), TestHash, TestBlock)
	if err != nil {
		c.Error(err)
	}
	p := fmt.Sprintf("%s/%s/%s", v.Root, TestHash[:3], TestHash)
	if buf, err := ioutil.ReadFile(p); err != nil {
		c.Error(err)
	} else if bytes.Compare(buf, TestBlock) != 0 {
		c.Errorf("Write should have stored %s, did store %s",
			string(TestBlock), string(buf))
	}
}

func (s *unixVolumeSuite) TestPutBadVolume(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, false)
	defer v.Teardown()

	err := os.RemoveAll(v.Root)
	c.Assert(err, check.IsNil)
	err = v.BlockWrite(context.Background(), TestHash, TestBlock)
	c.Check(err, check.IsNil)
}

func (s *unixVolumeSuite) TestIsFull(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, false)
	defer v.Teardown()

	fullPath := v.Root + "/full"
	now := fmt.Sprintf("%d", time.Now().Unix())
	os.Symlink(now, fullPath)
	if !v.isFull() {
		c.Error("volume claims not to be full")
	}
	os.Remove(fullPath)

	// Test with an expired /full link.
	expired := fmt.Sprintf("%d", time.Now().Unix()-3605)
	os.Symlink(expired, fullPath)
	if v.isFull() {
		c.Error("volume should no longer be full")
	}
}

func (s *unixVolumeSuite) TestUnixVolumeGetFuncWorkerError(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, false)
	defer v.Teardown()

	v.BlockWrite(context.Background(), TestHash, TestBlock)
	mockErr := errors.New("Mock error")
	err := v.getFunc(context.Background(), v.blockPath(TestHash), func(rdr io.Reader) error {
		return mockErr
	})
	if err != mockErr {
		c.Errorf("Got %v, expected %v", err, mockErr)
	}
}

func (s *unixVolumeSuite) TestUnixVolumeGetFuncFileError(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, false)
	defer v.Teardown()

	funcCalled := false
	err := v.getFunc(context.Background(), v.blockPath(TestHash), func(rdr io.Reader) error {
		funcCalled = true
		return nil
	})
	if err == nil {
		c.Errorf("Expected error opening non-existent file")
	}
	if funcCalled {
		c.Errorf("Worker func should not have been called")
	}
}

func (s *unixVolumeSuite) TestUnixVolumeGetFuncWorkerWaitsOnMutex(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, false)
	defer v.Teardown()

	v.BlockWrite(context.Background(), TestHash, TestBlock)

	mtx := NewMockMutex()
	v.locker = mtx

	funcCalled := make(chan struct{})
	go v.getFunc(context.Background(), v.blockPath(TestHash), func(rdr io.Reader) error {
		funcCalled <- struct{}{}
		return nil
	})
	select {
	case mtx.AllowLock <- struct{}{}:
	case <-funcCalled:
		c.Fatal("Function was called before mutex was acquired")
	case <-time.After(5 * time.Second):
		c.Fatal("Timed out before mutex was acquired")
	}
	select {
	case <-funcCalled:
	case mtx.AllowUnlock <- struct{}{}:
		c.Fatal("Mutex was released before function was called")
	case <-time.After(5 * time.Second):
		c.Fatal("Timed out waiting for funcCalled")
	}
	select {
	case mtx.AllowUnlock <- struct{}{}:
	case <-time.After(5 * time.Second):
		c.Fatal("Timed out waiting for getFunc() to release mutex")
	}
}

type MockMutex struct {
	AllowLock   chan struct{}
	AllowUnlock chan struct{}
}

func NewMockMutex() *MockMutex {
	return &MockMutex{
		AllowLock:   make(chan struct{}),
		AllowUnlock: make(chan struct{}),
	}
}

// Lock waits for someone to send to AllowLock.
func (m *MockMutex) Lock() {
	<-m.AllowLock
}

// Unlock waits for someone to send to AllowUnlock.
func (m *MockMutex) Unlock() {
	<-m.AllowUnlock
}

func (s *unixVolumeSuite) TestUnixVolumeContextCancelBlockWrite(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, true)
	defer v.Teardown()
	v.locker.Lock()
	defer v.locker.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := v.BlockWrite(ctx, TestHash, TestBlock)
	if err != context.Canceled {
		c.Errorf("BlockWrite() returned %s -- expected short read / canceled", err)
	}
}

func (s *unixVolumeSuite) TestUnixVolumeContextCancelBlockRead(c *check.C) {
	v := s.newTestableUnixVolume(c, s.params, true)
	defer v.Teardown()
	err := v.BlockWrite(context.Background(), TestHash, TestBlock)
	if err != nil {
		c.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	v.locker.Lock()
	defer v.locker.Unlock()
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	n, err := v.BlockRead(ctx, TestHash, io.Discard)
	if n > 0 || err != context.Canceled {
		c.Errorf("BlockRead() returned %d, %s -- expected short read / canceled", n, err)
	}
}

func (s *unixVolumeSuite) TestStats(c *check.C) {
	vol := s.newTestableUnixVolume(c, s.params, false)
	stats := func() string {
		buf, err := json.Marshal(vol.InternalStats())
		c.Check(err, check.IsNil)
		return string(buf)
	}

	c.Check(stats(), check.Matches, `.*"StatOps":1,.*`) // (*unixVolume)check() calls Stat() once
	c.Check(stats(), check.Matches, `.*"Errors":0,.*`)

	_, err := vol.BlockRead(context.Background(), fooHash, io.Discard)
	c.Check(err, check.NotNil)
	c.Check(stats(), check.Matches, `.*"StatOps":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"Errors":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"\*(fs|os)\.PathError":[^0].*`) // os.PathError changed to fs.PathError in Go 1.16
	c.Check(stats(), check.Matches, `.*"InBytes":0,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":0,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":0,.*`)

	err = vol.BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"OutBytes":3,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":0,.*`)
	c.Check(stats(), check.Matches, `.*"UtimesOps":1,.*`)

	err = vol.BlockTouch(fooHash)
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"FlockOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"UtimesOps":2,.*`)

	buf := bytes.NewBuffer(nil)
	_, err = vol.BlockRead(context.Background(), fooHash, buf)
	c.Check(err, check.IsNil)
	c.Check(buf.String(), check.Equals, "foo")
	c.Check(stats(), check.Matches, `.*"InBytes":3,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":2,.*`)

	err = vol.BlockTrash(fooHash)
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"FlockOps":2,.*`)
}

func (s *unixVolumeSuite) TestSkipUnusedDirs(c *check.C) {
	vol := s.newTestableUnixVolume(c, s.params, false)

	err := os.Mkdir(vol.unixVolume.Root+"/aaa", 0777)
	c.Assert(err, check.IsNil)
	err = os.Mkdir(vol.unixVolume.Root+"/.aaa", 0777) // EmptyTrash should not look here
	c.Assert(err, check.IsNil)
	deleteme := vol.unixVolume.Root + "/aaa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.trash.1"
	err = ioutil.WriteFile(deleteme, []byte{1, 2, 3}, 0777)
	c.Assert(err, check.IsNil)
	skipme := vol.unixVolume.Root + "/.aaa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.trash.1"
	err = ioutil.WriteFile(skipme, []byte{1, 2, 3}, 0777)
	c.Assert(err, check.IsNil)
	vol.EmptyTrash()

	_, err = os.Stat(skipme)
	c.Check(err, check.IsNil)

	_, err = os.Stat(deleteme)
	c.Check(err, check.NotNil)
	c.Check(os.IsNotExist(err), check.Equals, true)
}
