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
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	check "gopkg.in/check.v1"
)

type TestableUnixVolume struct {
	UnixVolume
	t TB
}

func NewTestableUnixVolume(t TB, serialize bool, readonly bool) *TestableUnixVolume {
	d, err := ioutil.TempDir("", "volume_test")
	if err != nil {
		t.Fatal(err)
	}
	var locker sync.Locker
	if serialize {
		locker = &sync.Mutex{}
	}
	return &TestableUnixVolume{
		UnixVolume: UnixVolume{
			Root:     d,
			ReadOnly: readonly,
			locker:   locker,
		},
		t: t,
	}
}

// PutRaw writes a Keep block directly into a UnixVolume, even if
// the volume is readonly.
func (v *TestableUnixVolume) PutRaw(locator string, data []byte) {
	defer func(orig bool) {
		v.ReadOnly = orig
	}(v.ReadOnly)
	v.ReadOnly = false
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
		v.t.Fatal(err)
	}
}

// serialize = false; readonly = false
func TestUnixVolumeWithGenericTests(t *testing.T) {
	DoGenericVolumeTests(t, func(t TB) TestableVolume {
		return NewTestableUnixVolume(t, false, false)
	})
}

// serialize = false; readonly = true
func TestUnixVolumeWithGenericTestsReadOnly(t *testing.T) {
	DoGenericVolumeTests(t, func(t TB) TestableVolume {
		return NewTestableUnixVolume(t, false, true)
	})
}

// serialize = true; readonly = false
func TestUnixVolumeWithGenericTestsSerialized(t *testing.T) {
	DoGenericVolumeTests(t, func(t TB) TestableVolume {
		return NewTestableUnixVolume(t, true, false)
	})
}

// serialize = false; readonly = false
func TestUnixVolumeHandlersWithGenericVolumeTests(t *testing.T) {
	DoHandlersWithGenericVolumeTests(t, func(t TB) (*RRVolumeManager, []TestableVolume) {
		vols := make([]Volume, 2)
		testableUnixVols := make([]TestableVolume, 2)

		for i := range vols {
			v := NewTestableUnixVolume(t, false, false)
			vols[i] = v
			testableUnixVols[i] = v
		}

		return MakeRRVolumeManager(vols), testableUnixVols
	})
}

func TestReplicationDefault1(t *testing.T) {
	v := &UnixVolume{
		Root:     "/",
		ReadOnly: true,
	}
	if err := v.Start(); err != nil {
		t.Error(err)
	}
	if got := v.Replication(); got != 1 {
		t.Errorf("Replication() returned %d, expected 1 if no config given", got)
	}
}

func TestGetNotFound(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()
	v.Put(context.Background(), TestHash, TestBlock)

	buf := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), TestHash2, buf)
	switch {
	case os.IsNotExist(err):
		break
	case err == nil:
		t.Errorf("Read should have failed, returned %+q", buf[:n])
	default:
		t.Errorf("Read expected ErrNotExist, got: %s", err)
	}
}

func TestPut(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	err := v.Put(context.Background(), TestHash, TestBlock)
	if err != nil {
		t.Error(err)
	}
	p := fmt.Sprintf("%s/%s/%s", v.Root, TestHash[:3], TestHash)
	if buf, err := ioutil.ReadFile(p); err != nil {
		t.Error(err)
	} else if bytes.Compare(buf, TestBlock) != 0 {
		t.Errorf("Write should have stored %s, did store %s",
			string(TestBlock), string(buf))
	}
}

func TestPutBadVolume(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	os.Chmod(v.Root, 000)
	err := v.Put(context.Background(), TestHash, TestBlock)
	if err == nil {
		t.Error("Write should have failed")
	}
}

func TestUnixVolumeReadonly(t *testing.T) {
	v := NewTestableUnixVolume(t, false, true)
	defer v.Teardown()

	v.PutRaw(TestHash, TestBlock)

	buf := make([]byte, BlockSize)
	_, err := v.Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Errorf("got err %v, expected nil", err)
	}

	err = v.Put(context.Background(), TestHash, TestBlock)
	if err != MethodDisabledError {
		t.Errorf("got err %v, expected MethodDisabledError", err)
	}

	err = v.Touch(TestHash)
	if err != MethodDisabledError {
		t.Errorf("got err %v, expected MethodDisabledError", err)
	}

	err = v.Trash(TestHash)
	if err != MethodDisabledError {
		t.Errorf("got err %v, expected MethodDisabledError", err)
	}
}

func TestIsFull(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	fullPath := v.Root + "/full"
	now := fmt.Sprintf("%d", time.Now().Unix())
	os.Symlink(now, fullPath)
	if !v.IsFull() {
		t.Errorf("%s: claims not to be full", v)
	}
	os.Remove(fullPath)

	// Test with an expired /full link.
	expired := fmt.Sprintf("%d", time.Now().Unix()-3605)
	os.Symlink(expired, fullPath)
	if v.IsFull() {
		t.Errorf("%s: should no longer be full", v)
	}
}

func TestNodeStatus(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	// Get node status and make a basic sanity check.
	volinfo := v.Status()
	if volinfo.MountPoint != v.Root {
		t.Errorf("GetNodeStatus mount_point %s, expected %s", volinfo.MountPoint, v.Root)
	}
	if volinfo.DeviceNum == 0 {
		t.Errorf("uninitialized device_num in %v", volinfo)
	}
	if volinfo.BytesFree == 0 {
		t.Errorf("uninitialized bytes_free in %v", volinfo)
	}
	if volinfo.BytesUsed == 0 {
		t.Errorf("uninitialized bytes_used in %v", volinfo)
	}
}

func TestUnixVolumeGetFuncWorkerError(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	v.Put(context.Background(), TestHash, TestBlock)
	mockErr := errors.New("Mock error")
	err := v.getFunc(context.Background(), v.blockPath(TestHash), func(rdr io.Reader) error {
		return mockErr
	})
	if err != mockErr {
		t.Errorf("Got %v, expected %v", err, mockErr)
	}
}

func TestUnixVolumeGetFuncFileError(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	funcCalled := false
	err := v.getFunc(context.Background(), v.blockPath(TestHash), func(rdr io.Reader) error {
		funcCalled = true
		return nil
	})
	if err == nil {
		t.Errorf("Expected error opening non-existent file")
	}
	if funcCalled {
		t.Errorf("Worker func should not have been called")
	}
}

func TestUnixVolumeGetFuncWorkerWaitsOnMutex(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
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
		t.Fatal("Function was called before mutex was acquired")
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out before mutex was acquired")
	}
	select {
	case <-funcCalled:
	case mtx.AllowUnlock <- struct{}{}:
		t.Fatal("Mutex was released before function was called")
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for funcCalled")
	}
	select {
	case mtx.AllowUnlock <- struct{}{}:
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for getFunc() to release mutex")
	}
}

func TestUnixVolumeCompare(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	v.Put(context.Background(), TestHash, TestBlock)
	err := v.Compare(context.Background(), TestHash, TestBlock)
	if err != nil {
		t.Errorf("Got err %q, expected nil", err)
	}

	err = v.Compare(context.Background(), TestHash, []byte("baddata"))
	if err != CollisionError {
		t.Errorf("Got err %q, expected %q", err, CollisionError)
	}

	v.Put(context.Background(), TestHash, []byte("baddata"))
	err = v.Compare(context.Background(), TestHash, TestBlock)
	if err != DiskHashError {
		t.Errorf("Got err %q, expected %q", err, DiskHashError)
	}

	p := fmt.Sprintf("%s/%s/%s", v.Root, TestHash[:3], TestHash)
	os.Chmod(p, 000)
	err = v.Compare(context.Background(), TestHash, TestBlock)
	if err == nil || strings.Index(err.Error(), "permission denied") < 0 {
		t.Errorf("Got err %q, expected %q", err, "permission denied")
	}
}

func TestUnixVolumeContextCancelPut(t *testing.T) {
	v := NewTestableUnixVolume(t, true, false)
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
		t.Errorf("Put() returned %s -- expected short read / canceled", err)
	}
}

func TestUnixVolumeContextCancelGet(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()
	bpath := v.blockPath(TestHash)
	v.PutRaw(TestHash, TestBlock)
	os.Remove(bpath)
	err := syscall.Mkfifo(bpath, 0600)
	if err != nil {
		t.Fatalf("Mkfifo %s: %s", bpath, err)
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
		t.Errorf("Get() returned %d, %s -- expected short read / canceled", n, err)
	}
}

var _ = check.Suite(&UnixVolumeSuite{})

type UnixVolumeSuite struct {
	volume *TestableUnixVolume
}

func (s *UnixVolumeSuite) TearDownTest(c *check.C) {
	if s.volume != nil {
		s.volume.Teardown()
	}
}

func (s *UnixVolumeSuite) TestStats(c *check.C) {
	s.volume = NewTestableUnixVolume(c, false, false)
	stats := func() string {
		buf, err := json.Marshal(s.volume.InternalStats())
		c.Check(err, check.IsNil)
		return string(buf)
	}

	c.Check(stats(), check.Matches, `.*"StatOps":0,.*`)
	c.Check(stats(), check.Matches, `.*"Errors":0,.*`)

	loc := "acbd18db4cc2f85cedef654fccc4a4d8"
	_, err := s.volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.NotNil)
	c.Check(stats(), check.Matches, `.*"StatOps":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"Errors":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"\*os\.PathError":[^0].*`)
	c.Check(stats(), check.Matches, `.*"InBytes":0,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":0,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":0,.*`)

	err = s.volume.Put(context.Background(), loc, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"OutBytes":3,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":0,.*`)
	c.Check(stats(), check.Matches, `.*"UtimesOps":0,.*`)

	err = s.volume.Touch(loc)
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"FlockOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":1,.*`)
	c.Check(stats(), check.Matches, `.*"UtimesOps":1,.*`)

	_, err = s.volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.IsNil)
	err = s.volume.Compare(context.Background(), loc, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"InBytes":6,.*`)
	c.Check(stats(), check.Matches, `.*"OpenOps":3,.*`)

	err = s.volume.Trash(loc)
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"FlockOps":2,.*`)
}

func (s *UnixVolumeSuite) TestConfig(c *check.C) {
	var cfg Config
	err := yaml.Unmarshal([]byte(`
Volumes:
  - Type: Directory
    StorageClasses: ["class_a", "class_b"]
`), &cfg)

	c.Check(err, check.IsNil)
	c.Check(cfg.Volumes[0].GetStorageClasses(), check.DeepEquals, []string{"class_a", "class_b"})
}
