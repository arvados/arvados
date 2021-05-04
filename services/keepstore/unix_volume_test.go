// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

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

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

type TestableUnixVolume struct {
	UnixVolume
	t TB
}

// PutRaw writes a Keep block directly into a UnixVolume, even if
// the volume is readonly.
func (v *TestableUnixVolume) PutRaw(locator string, data []byte) {
	defer func(orig bool) {
		v.volume.ReadOnly = orig
	}(v.volume.ReadOnly)
	v.volume.ReadOnly = false
	err := v.Put(context.Background(), locator, data)
	if err != nil {
		v.t.Fatal(err)
	}
}

func (v *TestableUnixVolume) TouchWithDate(locator string, lastPut time.Time) {
	err := syscall.Utime(v.blockPath(locator), &syscall.Utimbuf{lastPut.Unix(), lastPut.Unix()})
	if err != nil {
		v.t.Fatal(err)
	}
}

func (v *TestableUnixVolume) Teardown() {
	if err := os.RemoveAll(v.Root); err != nil {
		v.t.Error(err)
	}
}

func (v *TestableUnixVolume) ReadWriteOperationLabelValues() (r, w string) {
	return "open", "create"
}

var _ = check.Suite(&UnixVolumeSuite{})

type UnixVolumeSuite struct {
	cluster *arvados.Cluster
	volumes []*TestableUnixVolume
	metrics *volumeMetricsVecs
}

func (s *UnixVolumeSuite) SetUpTest(c *check.C) {
	s.cluster = testCluster(c)
	s.metrics = newVolumeMetricsVecs(prometheus.NewRegistry())
}

func (s *UnixVolumeSuite) TearDownTest(c *check.C) {
	for _, v := range s.volumes {
		v.Teardown()
	}
}

func (s *UnixVolumeSuite) newTestableUnixVolume(c *check.C, cluster *arvados.Cluster, volume arvados.Volume, metrics *volumeMetricsVecs, serialize bool) *TestableUnixVolume {
	d, err := ioutil.TempDir("", "volume_test")
	c.Check(err, check.IsNil)
	var locker sync.Locker
	if serialize {
		locker = &sync.Mutex{}
	}
	v := &TestableUnixVolume{
		UnixVolume: UnixVolume{
			Root:    d,
			locker:  locker,
			cluster: cluster,
			logger:  ctxlog.TestLogger(c),
			volume:  volume,
			metrics: metrics,
		},
		t: c,
	}
	c.Check(v.check(), check.IsNil)
	s.volumes = append(s.volumes, v)
	return v
}

// serialize = false; readonly = false
func (s *UnixVolumeSuite) TestUnixVolumeWithGenericTests(c *check.C) {
	DoGenericVolumeTests(c, false, func(t TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume {
		return s.newTestableUnixVolume(c, cluster, volume, metrics, false)
	})
}

// serialize = false; readonly = true
func (s *UnixVolumeSuite) TestUnixVolumeWithGenericTestsReadOnly(c *check.C) {
	DoGenericVolumeTests(c, true, func(t TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume {
		return s.newTestableUnixVolume(c, cluster, volume, metrics, true)
	})
}

// serialize = true; readonly = false
func (s *UnixVolumeSuite) TestUnixVolumeWithGenericTestsSerialized(c *check.C) {
	DoGenericVolumeTests(c, false, func(t TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume {
		return s.newTestableUnixVolume(c, cluster, volume, metrics, false)
	})
}

// serialize = true; readonly = true
func (s *UnixVolumeSuite) TestUnixVolumeHandlersWithGenericVolumeTests(c *check.C) {
	DoGenericVolumeTests(c, true, func(t TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume {
		return s.newTestableUnixVolume(c, cluster, volume, metrics, true)
	})
}

func (s *UnixVolumeSuite) TestGetNotFound(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()
	v.Put(context.Background(), TestHash, TestBlock)

	buf := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), TestHash2, buf)
	switch {
	case os.IsNotExist(err):
		break
	case err == nil:
		c.Errorf("Read should have failed, returned %+q", buf[:n])
	default:
		c.Errorf("Read expected ErrNotExist, got: %s", err)
	}
}

func (s *UnixVolumeSuite) TestPut(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()

	err := v.Put(context.Background(), TestHash, TestBlock)
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

func (s *UnixVolumeSuite) TestPutBadVolume(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()

	err := os.RemoveAll(v.Root)
	c.Assert(err, check.IsNil)
	err = v.Put(context.Background(), TestHash, TestBlock)
	c.Check(err, check.IsNil)
}

func (s *UnixVolumeSuite) TestUnixVolumeReadonly(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{ReadOnly: true, Replication: 1}, s.metrics, false)
	defer v.Teardown()

	v.PutRaw(TestHash, TestBlock)

	buf := make([]byte, BlockSize)
	_, err := v.Get(context.Background(), TestHash, buf)
	if err != nil {
		c.Errorf("got err %v, expected nil", err)
	}

	err = v.Put(context.Background(), TestHash, TestBlock)
	if err != MethodDisabledError {
		c.Errorf("got err %v, expected MethodDisabledError", err)
	}

	err = v.Touch(TestHash)
	if err != MethodDisabledError {
		c.Errorf("got err %v, expected MethodDisabledError", err)
	}

	err = v.Trash(TestHash)
	if err != MethodDisabledError {
		c.Errorf("got err %v, expected MethodDisabledError", err)
	}
}

func (s *UnixVolumeSuite) TestIsFull(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()

	fullPath := v.Root + "/full"
	now := fmt.Sprintf("%d", time.Now().Unix())
	os.Symlink(now, fullPath)
	if !v.IsFull() {
		c.Errorf("%s: claims not to be full", v)
	}
	os.Remove(fullPath)

	// Test with an expired /full link.
	expired := fmt.Sprintf("%d", time.Now().Unix()-3605)
	os.Symlink(expired, fullPath)
	if v.IsFull() {
		c.Errorf("%s: should no longer be full", v)
	}
}

func (s *UnixVolumeSuite) TestNodeStatus(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()

	// Get node status and make a basic sanity check.
	volinfo := v.Status()
	if volinfo.MountPoint != v.Root {
		c.Errorf("GetNodeStatus mount_point %s, expected %s", volinfo.MountPoint, v.Root)
	}
	if volinfo.DeviceNum == 0 {
		c.Errorf("uninitialized device_num in %v", volinfo)
	}
	if volinfo.BytesFree == 0 {
		c.Errorf("uninitialized bytes_free in %v", volinfo)
	}
	if volinfo.BytesUsed == 0 {
		c.Errorf("uninitialized bytes_used in %v", volinfo)
	}
}

func (s *UnixVolumeSuite) TestUnixVolumeGetFuncWorkerError(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()

	v.Put(context.Background(), TestHash, TestBlock)
	mockErr := errors.New("Mock error")
	err := v.getFunc(context.Background(), v.blockPath(TestHash), func(rdr io.Reader) error {
		return mockErr
	})
	if err != mockErr {
		c.Errorf("Got %v, expected %v", err, mockErr)
	}
}

func (s *UnixVolumeSuite) TestUnixVolumeGetFuncFileError(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
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

func (s *UnixVolumeSuite) TestUnixVolumeGetFuncWorkerWaitsOnMutex(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()

	v.Put(context.Background(), TestHash, TestBlock)

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

func (s *UnixVolumeSuite) TestUnixVolumeCompare(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()

	v.Put(context.Background(), TestHash, TestBlock)
	err := v.Compare(context.Background(), TestHash, TestBlock)
	if err != nil {
		c.Errorf("Got err %q, expected nil", err)
	}

	err = v.Compare(context.Background(), TestHash, []byte("baddata"))
	if err != CollisionError {
		c.Errorf("Got err %q, expected %q", err, CollisionError)
	}

	v.Put(context.Background(), TestHash, []byte("baddata"))
	err = v.Compare(context.Background(), TestHash, TestBlock)
	if err != DiskHashError {
		c.Errorf("Got err %q, expected %q", err, DiskHashError)
	}

	if os.Getuid() == 0 {
		c.Log("skipping 'permission denied' check when running as root")
	} else {
		p := fmt.Sprintf("%s/%s/%s", v.Root, TestHash[:3], TestHash)
		err = os.Chmod(p, 000)
		c.Assert(err, check.IsNil)
		err = v.Compare(context.Background(), TestHash, TestBlock)
		c.Check(err, check.ErrorMatches, ".*permission denied.*")
	}
}

func (s *UnixVolumeSuite) TestUnixVolumeContextCancelPut(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, true)
	defer v.Teardown()
	v.locker.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
		time.Sleep(50 * time.Millisecond)
		v.locker.Unlock()
	}()
	err := v.Put(ctx, TestHash, TestBlock)
	if err != context.Canceled {
		c.Errorf("Put() returned %s -- expected short read / canceled", err)
	}
}

func (s *UnixVolumeSuite) TestUnixVolumeContextCancelGet(c *check.C) {
	v := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	defer v.Teardown()
	bpath := v.blockPath(TestHash)
	v.PutRaw(TestHash, TestBlock)
	os.Remove(bpath)
	err := syscall.Mkfifo(bpath, 0600)
	if err != nil {
		c.Fatalf("Mkfifo %s: %s", bpath, err)
	}
	defer os.Remove(bpath)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	buf := make([]byte, len(TestBlock))
	n, err := v.Get(ctx, TestHash, buf)
	if n == len(TestBlock) || err != context.Canceled {
		c.Errorf("Get() returned %d, %s -- expected short read / canceled", n, err)
	}
}

func (s *UnixVolumeSuite) TestStats(c *check.C) {
	vol := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)
	stats := func() string {
		buf, err := json.Marshal(vol.InternalStats())
		c.Check(err, check.IsNil)
		return string(buf)
	}

	c.Check(stats(), check.Matches, `.*"StatOps":1,.*`) // (*UnixVolume)check() calls Stat() once
	c.Check(stats(), check.Matches, `.*"Errors":0,.*`)

	loc := "acbd18db4cc2f85cedef654fccc4a4d8"
	_, err := vol.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.NotNil)
	c.Check(stats(), check.Matches, `.*"StatOps":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"Errors":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"\*(fs|os)\.PathError":[^0].*`) // os.PathError changed to fs.PathError in Go 1.16
	c.Check(stats(), check.Matches, `.*"InBytes":0,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":0,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":0,.*`)

	err = vol.Put(context.Background(), loc, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"OutBytes":3,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":0,.*`)
	c.Check(stats(), check.Matches, `.*"UtimesOps":1,.*`)

	err = vol.Touch(loc)
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"FlockOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"UtimesOps":2,.*`)

	_, err = vol.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.IsNil)
	err = vol.Compare(context.Background(), loc, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"InBytes":6,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":3,.*`)

	err = vol.Trash(loc)
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"FlockOps":2,.*`)
}

func (s *UnixVolumeSuite) TestSkipUnusedDirs(c *check.C) {
	vol := s.newTestableUnixVolume(c, s.cluster, arvados.Volume{Replication: 1}, s.metrics, false)

	err := os.Mkdir(vol.UnixVolume.Root+"/aaa", 0777)
	c.Assert(err, check.IsNil)
	err = os.Mkdir(vol.UnixVolume.Root+"/.aaa", 0777) // EmptyTrash should not look here
	c.Assert(err, check.IsNil)
	deleteme := vol.UnixVolume.Root + "/aaa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.trash.1"
	err = ioutil.WriteFile(deleteme, []byte{1, 2, 3}, 0777)
	c.Assert(err, check.IsNil)
	skipme := vol.UnixVolume.Root + "/.aaa/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.trash.1"
	err = ioutil.WriteFile(skipme, []byte{1, 2, 3}, 0777)
	c.Assert(err, check.IsNil)
	vol.EmptyTrash()

	_, err = os.Stat(skipme)
	c.Check(err, check.IsNil)

	_, err = os.Stat(deleteme)
	c.Check(err, check.NotNil)
	c.Check(os.IsNotExist(err), check.Equals, true)
}
