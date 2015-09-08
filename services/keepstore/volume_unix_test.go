package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
)

type TestableUnixVolume struct {
	UnixVolume
	t *testing.T
}

func NewTestableUnixVolume(t *testing.T, serialize bool, readonly bool) *TestableUnixVolume {
	d, err := ioutil.TempDir("", "volume_test")
	if err != nil {
		t.Fatal(err)
	}
	return &TestableUnixVolume{
		UnixVolume: UnixVolume{
			root:      d,
			serialize: serialize,
			readonly:  readonly,
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

func TestUnixVolumeWithGenericTests(t *testing.T) {
	DoGenericVolumeTests(t, func(t *testing.T) TestableVolume {
		return NewTestableUnixVolume(t, false, false)
	})
}

func TestGet(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()
	v.Put(TEST_HASH, TEST_BLOCK)

	buf, err := v.Get(TEST_HASH)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(buf, TEST_BLOCK) != 0 {
		t.Errorf("expected %s, got %s", string(TEST_BLOCK), string(buf))
	}
}

func TestGetNotFound(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()
	v.Put(TEST_HASH, TEST_BLOCK)

	buf, err := v.Get(TEST_HASH_2)
	switch {
	case os.IsNotExist(err):
		break
	case err == nil:
		t.Errorf("Read should have failed, returned %s", string(buf))
	default:
		t.Errorf("Read expected ErrNotExist, got: %s", err)
	}
}

func TestIndexTo(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	v.Put(TEST_HASH, TEST_BLOCK)
	v.Put(TEST_HASH_2, TEST_BLOCK_2)
	v.Put(TEST_HASH_3, TEST_BLOCK_3)

	buf := new(bytes.Buffer)
	v.IndexTo("", buf)
	index_rows := strings.Split(string(buf.Bytes()), "\n")
	sort.Strings(index_rows)
	sorted_index := strings.Join(index_rows, "\n")
	m, err := regexp.MatchString(
		`^\n`+TEST_HASH+`\+\d+ \d+\n`+
			TEST_HASH_3+`\+\d+ \d+\n`+
			TEST_HASH_2+`\+\d+ \d+$`,
		sorted_index)
	if err != nil {
		t.Error(err)
	} else if !m {
		t.Errorf("Got index %q for empty prefix", sorted_index)
	}

	for _, prefix := range []string{"f", "f15", "f15ac"} {
		buf = new(bytes.Buffer)
		v.IndexTo(prefix, buf)
		m, err := regexp.MatchString(`^`+TEST_HASH_2+`\+\d+ \d+\n$`, string(buf.Bytes()))
		if err != nil {
			t.Error(err)
		} else if !m {
			t.Errorf("Got index %q for prefix %q", string(buf.Bytes()), prefix)
		}
	}
}

func TestPut(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	err := v.Put(TEST_HASH, TEST_BLOCK)
	if err != nil {
		t.Error(err)
	}
	p := fmt.Sprintf("%s/%s/%s", v.root, TEST_HASH[:3], TEST_HASH)
	if buf, err := ioutil.ReadFile(p); err != nil {
		t.Error(err)
	} else if bytes.Compare(buf, TEST_BLOCK) != 0 {
		t.Errorf("Write should have stored %s, did store %s",
			string(TEST_BLOCK), string(buf))
	}
}

func TestPutBadVolume(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	os.Chmod(v.root, 000)
	err := v.Put(TEST_HASH, TEST_BLOCK)
	if err == nil {
		t.Error("Write should have failed")
	}
}

func TestUnixVolumeReadonly(t *testing.T) {
	v := NewTestableUnixVolume(t, false, true)
	defer v.Teardown()

	v.PutRaw(TEST_HASH, TEST_BLOCK)

	_, err := v.Get(TEST_HASH)
	if err != nil {
		t.Errorf("got err %v, expected nil", err)
	}

	err = v.Put(TEST_HASH, TEST_BLOCK)
	if err != MethodDisabledError {
		t.Errorf("got err %v, expected MethodDisabledError", err)
	}

	err = v.Touch(TEST_HASH)
	if err != MethodDisabledError {
		t.Errorf("got err %v, expected MethodDisabledError", err)
	}

	err = v.Delete(TEST_HASH)
	if err != MethodDisabledError {
		t.Errorf("got err %v, expected MethodDisabledError", err)
	}
}

// TestPutTouch
//     Test that when applying PUT to a block that already exists,
//     the block's modification time is updated.
func TestPutTouch(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	if err := v.Put(TEST_HASH, TEST_BLOCK); err != nil {
		t.Error(err)
	}

	// We'll verify { t0 < threshold < t1 }, where t0 is the
	// existing block's timestamp on disk before Put() and t1 is
	// its timestamp after Put().
	threshold := time.Now().Add(-time.Second)

	// Set the stored block's mtime far enough in the past that we
	// can see the difference between "timestamp didn't change"
	// and "timestamp granularity is too low".
	v.TouchWithDate(TEST_HASH, time.Now().Add(-20*time.Second))

	// Make sure v.Mtime() agrees the above Utime really worked.
	if t0, err := v.Mtime(TEST_HASH); err != nil || t0.IsZero() || !t0.Before(threshold) {
		t.Errorf("Setting mtime failed: %v, %v", t0, err)
	}

	// Write the same block again.
	if err := v.Put(TEST_HASH, TEST_BLOCK); err != nil {
		t.Error(err)
	}

	// Verify threshold < t1
	if t1, err := v.Mtime(TEST_HASH); err != nil {
		t.Error(err)
	} else if t1.Before(threshold) {
		t.Errorf("t1 %v should be >= threshold %v after v.Put ", t1, threshold)
	}
}

// Serialization tests: launch a bunch of concurrent
//
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
//
func TestGetSerialized(t *testing.T) {
	// Create a volume with I/O serialization enabled.
	v := NewTestableUnixVolume(t, true, false)
	defer v.Teardown()

	v.Put(TEST_HASH, TEST_BLOCK)
	v.Put(TEST_HASH_2, TEST_BLOCK_2)
	v.Put(TEST_HASH_3, TEST_BLOCK_3)

	sem := make(chan int)
	go func(sem chan int) {
		buf, err := v.Get(TEST_HASH)
		if err != nil {
			t.Errorf("err1: %v", err)
		}
		if bytes.Compare(buf, TEST_BLOCK) != 0 {
			t.Errorf("buf should be %s, is %s", string(TEST_BLOCK), string(buf))
		}
		sem <- 1
	}(sem)

	go func(sem chan int) {
		buf, err := v.Get(TEST_HASH_2)
		if err != nil {
			t.Errorf("err2: %v", err)
		}
		if bytes.Compare(buf, TEST_BLOCK_2) != 0 {
			t.Errorf("buf should be %s, is %s", string(TEST_BLOCK_2), string(buf))
		}
		sem <- 1
	}(sem)

	go func(sem chan int) {
		buf, err := v.Get(TEST_HASH_3)
		if err != nil {
			t.Errorf("err3: %v", err)
		}
		if bytes.Compare(buf, TEST_BLOCK_3) != 0 {
			t.Errorf("buf should be %s, is %s", string(TEST_BLOCK_3), string(buf))
		}
		sem <- 1
	}(sem)

	// Wait for all goroutines to finish
	for done := 0; done < 3; {
		done += <-sem
	}
}

func TestPutSerialized(t *testing.T) {
	// Create a volume with I/O serialization enabled.
	v := NewTestableUnixVolume(t, true, false)
	defer v.Teardown()

	sem := make(chan int)
	go func(sem chan int) {
		err := v.Put(TEST_HASH, TEST_BLOCK)
		if err != nil {
			t.Errorf("err1: %v", err)
		}
		sem <- 1
	}(sem)

	go func(sem chan int) {
		err := v.Put(TEST_HASH_2, TEST_BLOCK_2)
		if err != nil {
			t.Errorf("err2: %v", err)
		}
		sem <- 1
	}(sem)

	go func(sem chan int) {
		err := v.Put(TEST_HASH_3, TEST_BLOCK_3)
		if err != nil {
			t.Errorf("err3: %v", err)
		}
		sem <- 1
	}(sem)

	// Wait for all goroutines to finish
	for done := 0; done < 3; {
		done += <-sem
	}

	// Double check that we actually wrote the blocks we expected to write.
	buf, err := v.Get(TEST_HASH)
	if err != nil {
		t.Errorf("Get #1: %v", err)
	}
	if bytes.Compare(buf, TEST_BLOCK) != 0 {
		t.Errorf("Get #1: expected %s, got %s", string(TEST_BLOCK), string(buf))
	}

	buf, err = v.Get(TEST_HASH_2)
	if err != nil {
		t.Errorf("Get #2: %v", err)
	}
	if bytes.Compare(buf, TEST_BLOCK_2) != 0 {
		t.Errorf("Get #2: expected %s, got %s", string(TEST_BLOCK_2), string(buf))
	}

	buf, err = v.Get(TEST_HASH_3)
	if err != nil {
		t.Errorf("Get #3: %v", err)
	}
	if bytes.Compare(buf, TEST_BLOCK_3) != 0 {
		t.Errorf("Get #3: expected %s, got %s", string(TEST_BLOCK_3), string(buf))
	}
}

func TestIsFull(t *testing.T) {
	v := NewTestableUnixVolume(t, false, false)
	defer v.Teardown()

	full_path := v.root + "/full"
	now := fmt.Sprintf("%d", time.Now().Unix())
	os.Symlink(now, full_path)
	if !v.IsFull() {
		t.Errorf("%s: claims not to be full", v)
	}
	os.Remove(full_path)

	// Test with an expired /full link.
	expired := fmt.Sprintf("%d", time.Now().Unix()-3605)
	os.Symlink(expired, full_path)
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
