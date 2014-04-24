package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TempUnixVolume(t *testing.T) UnixVolume {
	d, err := ioutil.TempDir("", "volume_test")
	if err != nil {
		t.Fatal(err)
	}
	return UnixVolume{d}
}

func _teardown(v UnixVolume) {
	os.RemoveAll(v.root)
}

// store writes a Keep block directly into a UnixVolume, for testing
// UnixVolume methods.
//
func _store(t *testing.T, vol UnixVolume, filename string, block []byte) {
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

func TestRead(t *testing.T) {
	v := TempUnixVolume(t)
	defer _teardown(v)
	_store(t, v, TEST_HASH, TEST_BLOCK)

	buf, err := v.Read(TEST_HASH)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(buf, TEST_BLOCK) != 0 {
		t.Errorf("expected %s, got %s", string(TEST_BLOCK), string(buf))
	}
}

func TestReadNotFound(t *testing.T) {
	v := TempUnixVolume(t)
	defer _teardown(v)
	_store(t, v, TEST_HASH, TEST_BLOCK)

	buf, err := v.Read(TEST_HASH_2)
	switch {
	case os.IsNotExist(err):
		break
	case err == nil:
		t.Errorf("Read should have failed, returned %s", string(buf))
	default:
		t.Errorf("Read expected ErrNotExist, got: %s", err)
	}
}

func TestWrite(t *testing.T) {
	v := TempUnixVolume(t)
	defer _teardown(v)

	err := v.Write(TEST_HASH, TEST_BLOCK)
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

func TestWriteBadVolume(t *testing.T) {
	v := TempUnixVolume(t)
	defer _teardown(v)

	os.Chmod(v.root, 000)
	err := v.Write(TEST_HASH, TEST_BLOCK)
	if err == nil {
		t.Error("Write should have failed")
	}
}

func TestIsFull(t *testing.T) {
	v := TempUnixVolume(t)
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
