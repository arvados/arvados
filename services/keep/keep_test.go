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

// Test simple block reads.
func TestGetBlockOK(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes and store a block in each of them.
	KeepVolumes = setup(t, 2)
	fmt.Println("KeepVolumes = ", KeepVolumes)

	for _, vol := range KeepVolumes {
		store(t, vol, TEST_HASH, TEST_BLOCK)
	}

	// Check that GetBlock returns success.
	result, err := GetBlock(TEST_HASH)
	if err != nil {
		t.Errorf("GetBlock error: %s", err)
	}
	if fmt.Sprint(result) != fmt.Sprint(TEST_BLOCK) {
		t.Errorf("expected %s, got %s", TEST_BLOCK, result)
	}
}

// Test block reads when one Keep volume is missing.
func TestGetBlockOneKeepOK(t *testing.T) {
	defer teardown()

	// Two test Keep volumes, only the second has a block.
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

// Test block read failure.
func TestGetBlockFail(t *testing.T) {
	defer teardown()

	// Create two empty test Keep volumes.
	KeepVolumes = setup(t, 2)

	// Check that GetBlock returns failure.
	result, err := GetBlock(TEST_HASH)
	if err == nil {
		t.Errorf("GetBlock incorrectly returned success: ", result)
	}
}

// Test reading a corrupt block.
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

// Test finding Keep volumes.
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

// Test that FindKeepVolumes returns an empty slice when no Keep volumes
// are present.
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
//
func store(t *testing.T, keepdir string, filename string, block []byte) error {
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

	return nil
}
