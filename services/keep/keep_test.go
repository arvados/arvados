package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

var TEST_BLOCK = []byte("The quick brown fox jumps over the lazy dog.")
var TEST_HASH = "e4d909c290d0fb1ca068ffaddf22cbd0"
var BAD_BLOCK = []byte("The magic words are squeamish ossifrage.")

// ========================================
// GetBlock tests.
// ========================================

// TestGetBlock
//     Test that simple block reads succeed.
//
func TestGetBlock(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes. Our block is stored on the second volume.
	KeepVolumes = setup(t, 2)
	store(t, KeepVolumes[1], TEST_HASH, TEST_BLOCK)

	// Check that GetBlock returns success.
	result, err := GetBlock(TEST_HASH)
	if err != nil {
		t.Errorf("GetBlock error: %s", err)
	}
	if fmt.Sprint(result) != fmt.Sprint(TEST_BLOCK) {
		t.Errorf("expected %s, got %s", TEST_BLOCK, result)
	}
}

// TestGetBlockMissing
//     GetBlock must return an error when the block is not found.
//
func TestGetBlockMissing(t *testing.T) {
	defer teardown()

	// Create two empty test Keep volumes.
	KeepVolumes = setup(t, 2)

	// Check that GetBlock returns failure.
	result, err := GetBlock(TEST_HASH)
	if err == nil {
		t.Errorf("GetBlock incorrectly returned success: ", result)
	}
}

// TestGetBlockCorrupt
//     GetBlock must return an error when a corrupted block is requested
//     (the contents of the file do not checksum to its hash).
//
func TestGetBlockCorrupt(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes and store a block in each of them,
	// but the hash of the block does not match the filename.
	KeepVolumes = setup(t, 2)
	for _, vol := range KeepVolumes {
		store(t, vol, TEST_HASH, BAD_BLOCK)
	}

	// Check that GetBlock returns failure.
	result, err := GetBlock(TEST_HASH)
	if err == nil {
		t.Errorf("GetBlock incorrectly returned success: %s", result)
	}
}

// ========================================
// PutBlock tests
// ========================================

// TestPutBlockOK
//     PutBlock can perform a simple block write and returns success.
//
func TestPutBlockOK(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes.
	KeepVolumes = setup(t, 2)

	// Check that PutBlock stores the data as expected.
	if err := PutBlock(TEST_BLOCK, TEST_HASH); err != nil {
		t.Fatalf("PutBlock: %v", err)
	}

	result, err := GetBlock(TEST_HASH)
	if err != nil {
		t.Fatalf("GetBlock: %s", err.Error())
	}
	if string(result) != string(TEST_BLOCK) {
		t.Error("PutBlock/GetBlock mismatch")
		t.Fatalf("PutBlock stored '%s', GetBlock retrieved '%s'",
			string(TEST_BLOCK), string(result))
	}
}

// TestPutBlockOneVol
//     PutBlock still returns success even when only one of the known
//     volumes is online.
//
func TestPutBlockOneVol(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes, but cripple one of them.
	KeepVolumes = setup(t, 2)
	os.Chmod(KeepVolumes[0], 000)

	// Check that PutBlock stores the data as expected.
	if err := PutBlock(TEST_BLOCK, TEST_HASH); err != nil {
		t.Fatalf("PutBlock: %v", err)
	}

	result, err := GetBlock(TEST_HASH)
	if err != nil {
		t.Fatalf("GetBlock: %s", err.Error())
	}
	if string(result) != string(TEST_BLOCK) {
		t.Error("PutBlock/GetBlock mismatch")
		t.Fatalf("PutBlock stored '%s', GetBlock retrieved '%s'",
			string(TEST_BLOCK), string(result))
	}
}

// TestPutBlockMD5Fail
//     Check that PutBlock returns an error if passed a block and hash that
//     do not match.
//
func TestPutBlockMD5Fail(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes.
	KeepVolumes = setup(t, 2)

	// Check that PutBlock returns the expected error when the hash does
	// not match the block.
	if err := PutBlock(BAD_BLOCK, TEST_HASH); err == nil {
		t.Error("PutBlock succeeded despite a block mismatch")
	} else {
		ke := err.(*KeepError)
		if ke.HTTPCode != ErrMD5Fail {
			t.Errorf("PutBlock returned the wrong error (%v)", ke)
		}
	}

	// Confirm that GetBlock fails to return anything.
	if result, err := GetBlock(TEST_HASH); err == nil {
		t.Errorf("GetBlock succeded after a corrupt block store, returned '%s'",
			string(result))
	}
}

// TestPutBlockCollision
//     PutBlock must report a 400 Collision error when asked to store a block
//     when a different block exists on disk under the same identifier.
//
func TestPutBlockCollision(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes.
	KeepVolumes = setup(t, 2)

	// Store a corrupted block under TEST_HASH.
	store(t, KeepVolumes[0], TEST_HASH, BAD_BLOCK)

	// Attempting to put TEST_BLOCK should produce a 400 Collision error.
	if err := PutBlock(TEST_BLOCK, TEST_HASH); err == nil {
		t.Error("Expected PutBlock error, but no error returned")
	} else {
		ke := err.(*KeepError)
		if ke.HTTPCode != ErrCollision {
			t.Errorf("Expected 400 Collision error, got %v", ke)
		}
	}

	KeepVolumes = nil
}

// TestFindKeepVolumes
//     Confirms that FindKeepVolumes finds tmpfs volumes with "/keep"
//     directories at the top level.
//
func TestFindKeepVolumes(t *testing.T) {
	defer teardown()

	// Initialize two keep volumes.
	var tempVols []string = setup(t, 2)

	// Set up a bogus PROC_MOUNTS file.
	if f, err := ioutil.TempFile("", "keeptest"); err == nil {
		for _, vol := range tempVols {
			fmt.Fprintf(f, "tmpfs %s tmpfs opts\n", path.Dir(vol))
		}
		f.Close()
		PROC_MOUNTS = f.Name()

		// Check that FindKeepVolumes finds the temp volumes.
		resultVols := FindKeepVolumes()
		if len(tempVols) != len(resultVols) {
			t.Fatalf("set up %d volumes, FindKeepVolumes found %d\n",
				len(tempVols), len(resultVols))
		}
		for i := range tempVols {
			if tempVols[i] != resultVols[i] {
				t.Errorf("FindKeepVolumes returned %s, expected %s\n",
					resultVols[i], tempVols[i])
			}
		}

		os.Remove(f.Name())
	}
}

// TestFindKeepVolumesFail
//     When no Keep volumes are present, FindKeepVolumes returns an empty slice.
//
func TestFindKeepVolumesFail(t *testing.T) {
	defer teardown()

	// Set up a bogus PROC_MOUNTS file with no Keep vols.
	if f, err := ioutil.TempFile("", "keeptest"); err == nil {
		fmt.Fprintln(f, "rootfs / rootfs opts 0 0")
		fmt.Fprintln(f, "sysfs /sys sysfs opts 0 0")
		fmt.Fprintln(f, "proc /proc proc opts 0 0")
		fmt.Fprintln(f, "udev /dev devtmpfs opts 0 0")
		fmt.Fprintln(f, "devpts /dev/pts devpts opts 0 0")
		f.Close()
		PROC_MOUNTS = f.Name()

		// Check that FindKeepVolumes returns an empty array.
		resultVols := FindKeepVolumes()
		if len(resultVols) != 0 {
			t.Fatalf("FindKeepVolumes returned %v", resultVols)
		}

		os.Remove(PROC_MOUNTS)
	}
}

// ========================================
// Helper functions for unit tests.
// ========================================

// setup
//     Create KeepVolumes for testing.
//     Returns a slice of pathnames to temporary Keep volumes.
//
func setup(t *testing.T, num_volumes int) []string {
	vols := make([]string, num_volumes)
	for i := range vols {
		if dir, err := ioutil.TempDir(os.TempDir(), "keeptest"); err == nil {
			vols[i] = dir + "/keep"
			os.Mkdir(vols[i], 0755)
		} else {
			t.Fatal(err)
		}
	}
	return vols
}

// teardown
//     Cleanup to perform after each test.
//
func teardown() {
	for _, vol := range KeepVolumes {
		os.RemoveAll(path.Dir(vol))
	}
}

// store
//     Low-level code to write Keep blocks directly to disk for testing.
//
func store(t *testing.T, keepdir string, filename string, block []byte) {
	blockdir := fmt.Sprintf("%s/%s", keepdir, filename[:3])
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
