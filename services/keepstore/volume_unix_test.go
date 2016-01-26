package main

import (
	"bytes"
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
			root:     d,
			locker:   locker,
			readonly: readonly,
		},
		t: t,
	}
}

// PutRaw writes a Keep block directly into a UnixVolume, even if
// the volume is readonly.
func (v *TestableUnixVolume) PutRaw(locator string, data []byte) {
	defer func(orig bool) {
		v.readonly = orig
	}(v.readonly)
	v.readonly = false
	err := v.Put(locator, data)
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
	if err := os.RemoveAll(v.root); err != nil {
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

func TestGetNotFound(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()
	v.Put(TestHash, TestBlock)

	buf, err := v.Get(TestHash2)
	switch {
	case os.IsNotExist(err):
		break
	case err == nil:
		t.Errorf("Read should have failed, returned %s", string(buf))
	default:
		t.Errorf("Read expected ErrNotExist, got: %s", err)
	}
}

func TestPut(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	err := v.Put(TestHash, TestBlock)
	if err != nil {
		t.Error(err)
	}
	p := fmt.Sprintf("%s/%s/%s", v.root, TestHash[:3], TestHash)
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

	os.Chmod(v.root, 000)
	err := v.Put(TestHash, TestBlock)
	if err == nil {
		t.Error("Write should have failed")
	}
}

func TestUnixVolumeReadonly(t *testing.T) {
	v := NewTestableUnixVolume(t, false, true)
	defer v.Teardown()

	v.PutRaw(TestHash, TestBlock)

	_, err := v.Get(TestHash)
	if err != nil {
		t.Errorf("got err %v, expected nil", err)
	}

	err = v.Put(TestHash, TestBlock)
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

	fullPath := v.root + "/full"
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
	if volinfo.MountPoint != v.root {
		t.Errorf("GetNodeStatus mount_point %s, expected %s", volinfo.MountPoint, v.root)
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

	v.Put(TestHash, TestBlock)
	mockErr := errors.New("Mock error")
	err := v.getFunc(v.blockPath(TestHash), func(rdr io.Reader) error {
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
	err := v.getFunc(v.blockPath(TestHash), func(rdr io.Reader) error {
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

	v.Put(TestHash, TestBlock)

	mtx := NewMockMutex()
	v.locker = mtx

	funcCalled := make(chan struct{})
	go v.getFunc(v.blockPath(TestHash), func(rdr io.Reader) error {
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

	v.Put(TestHash, TestBlock)
	err := v.Compare(TestHash, TestBlock)
	if err != nil {
		t.Errorf("Got err %q, expected nil", err)
	}

	err = v.Compare(TestHash, []byte("baddata"))
	if err != CollisionError {
		t.Errorf("Got err %q, expected %q", err, CollisionError)
	}

	v.Put(TestHash, []byte("baddata"))
	err = v.Compare(TestHash, TestBlock)
	if err != DiskHashError {
		t.Errorf("Got err %q, expected %q", err, DiskHashError)
	}

	p := fmt.Sprintf("%s/%s/%s", v.root, TestHash[:3], TestHash)
	os.Chmod(p, 000)
	err = v.Compare(TestHash, TestBlock)
	if err == nil || strings.Index(err.Error(), "permission denied") < 0 {
		t.Errorf("Got err %q, expected %q", err, "permission denied")
	}
}

// TODO(twp): show that the underlying Read/Write operations executed
// serially and not concurrently. The easiest way to do this is
// probably to activate verbose or debug logging, capture log output
// and examine it to confirm that Reads and Writes did not overlap.
//
// TODO(twp): a proper test of I/O serialization requires that a
// second request start while the first one is still underway.
// Guaranteeing that the test behaves this way requires some tricky
// synchronization and mocking.  For now we'll just launch a bunch of
// requests simultaenously in goroutines and demonstrate that they
// return accurate results.
