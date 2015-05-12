package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"
)

func TempUnixVolume(t *testing.T, serialize bool, readonly bool) *UnixVolume {
	d, err := ioutil.TempDir("", "volume_test")
	if err != nil {
		t.Fatal(err)
	}
	return &UnixVolume{
		root:      d,
		serialize: serialize,
		readonly:  readonly,
	}
}

func _teardown(v *UnixVolume) {
	os.RemoveAll(v.root)
}

// _store writes a Keep block directly into a UnixVolume, bypassing
// the overhead and safeguards of Put(). Useful for storing bogus data
// and isolating unit tests from Put() behavior.
func _store(t *testing.T, vol *UnixVolume, filename string, block []byte) {
	blockdir := fmt.Sprintf("%s/%s", vol.root, filename[:3])
	if err := os.MkdirAll(blockdir, 0755); err != nil {
		t.Fatal(err)
	}

	blockpath := fmt.Sprintf("%s/%s", blockdir, filename)
	if f, err := os.Create(blockpath); err == nil {
		f.Write(block)
		f.Close()
	} else {
		t.Fatal(err)
	}
}

func TestGet(t *testing.T) {
	v := TempUnixVolume(t, false, false)
	defer _teardown(v)
	_store(t, v, TEST_HASH, TEST_BLOCK)

	buf, err := v.Get(TEST_HASH)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(buf, TEST_BLOCK) != 0 {
		t.Errorf("expected %s, got %s", string(TEST_BLOCK), string(buf))
	}
}

func TestGetNotFound(t *testing.T) {
	v := TempUnixVolume(t, false, false)
	defer _teardown(v)
	_store(t, v, TEST_HASH, TEST_BLOCK)

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

func TestPut(t *testing.T) {
	v := TempUnixVolume(t, false, false)
	defer _teardown(v)

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
	v := TempUnixVolume(t, false, false)
	defer _teardown(v)

	os.Chmod(v.root, 000)
	err := v.Put(TEST_HASH, TEST_BLOCK)
	if err == nil {
		t.Error("Write should have failed")
	}
}

func TestUnixVolumeReadonly(t *testing.T) {
	v := TempUnixVolume(t, false, false)
	defer _teardown(v)

	// First write something before marking readonly
	err := v.Put(TEST_HASH, TEST_BLOCK)
	if err != nil {
		t.Error("got err %v, expected nil", err)
	}

	v.readonly = true

	_, err = v.Get(TEST_HASH)
	if err != nil {
		t.Error("got err %v, expected nil", err)
	}

	err = v.Put(TEST_HASH, TEST_BLOCK)
	if err != MethodDisabledError {
		t.Error("got err %v, expected MethodDisabledError", err)
	}

	err = v.Touch(TEST_HASH)
	if err != MethodDisabledError {
		t.Error("got err %v, expected MethodDisabledError", err)
	}

	err = v.Delete(TEST_HASH)
	if err != MethodDisabledError {
		t.Error("got err %v, expected MethodDisabledError", err)
	}
}

// TestPutTouch
//     Test that when applying PUT to a block that already exists,
//     the block's modification time is updated.
func TestPutTouch(t *testing.T) {
	v := TempUnixVolume(t, false, false)
	defer _teardown(v)

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
	{
		oldtime := time.Now().Add(-20 * time.Second).Unix()
		if err := syscall.Utime(v.blockPath(TEST_HASH),
			&syscall.Utimbuf{oldtime, oldtime}); err != nil {
			t.Error(err)
		}

		// Make sure v.Mtime() agrees the above Utime really worked.
		if t0, err := v.Mtime(TEST_HASH); err != nil || t0.IsZero() || !t0.Before(threshold) {
			t.Errorf("Setting mtime failed: %v, %v", t0, err)
		}
	}

	// Write the same block again.
	if err := v.Put(TEST_HASH, TEST_BLOCK); err != nil {
		t.Error(err)
	}

	// Verify threshold < t1
	t1, err := v.Mtime(TEST_HASH)
	if err != nil {
		t.Error(err)
	}
	if t1.Before(threshold) {
		t.Errorf("t1 %v must be >= threshold %v after v.Put ",
			t1, threshold)
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
	v := TempUnixVolume(t, true, false)
	defer _teardown(v)

	_store(t, v, TEST_HASH, TEST_BLOCK)
	_store(t, v, TEST_HASH_2, TEST_BLOCK_2)
	_store(t, v, TEST_HASH_3, TEST_BLOCK_3)

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
	v := TempUnixVolume(t, true, false)
	defer _teardown(v)

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
	for done := 0; done < 2; {
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
	v := TempUnixVolume(t, false, false)
	defer _teardown(v)

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
