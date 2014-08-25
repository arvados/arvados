package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var TEST_BLOCK = []byte("The quick brown fox jumps over the lazy dog.")
var TEST_HASH = "e4d909c290d0fb1ca068ffaddf22cbd0"
var TEST_HASH_PUT_RESPONSE = "e4d909c290d0fb1ca068ffaddf22cbd0+44\n"

var TEST_BLOCK_2 = []byte("Pack my box with five dozen liquor jugs.")
var TEST_HASH_2 = "f15ac516f788aec4f30932ffb6395c39"

var TEST_BLOCK_3 = []byte("Now is the time for all good men to come to the aid of their country.")
var TEST_HASH_3 = "eed29bbffbc2dbe5e5ee0bb71888e61f"

// BAD_BLOCK is used to test collisions and corruption.
// It must not match any test hashes.
var BAD_BLOCK = []byte("The magic words are squeamish ossifrage.")

// TODO(twp): Tests still to be written
//
//   * TestPutBlockFull
//       - test that PutBlock returns 503 Full if the filesystem is full.
//         (must mock FreeDiskSpace or Statfs? use a tmpfs?)
//
//   * TestPutBlockWriteErr
//       - test the behavior when Write returns an error.
//           - Possible solutions: use a small tmpfs and a high
//             MIN_FREE_KILOBYTES to trick PutBlock into attempting
//             to write a block larger than the amount of space left
//           - use an interface to mock ioutil.TempFile with a File
//             object that always returns an error on write
//
// ========================================
// GetBlock tests.
// ========================================

// TestGetBlock
//     Test that simple block reads succeed.
//
func TestGetBlock(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes. Our block is stored on the second volume.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	if err := vols[1].Put(TEST_HASH, TEST_BLOCK); err != nil {
		t.Error(err)
	}

	// Check that GetBlock returns success.
	result, err := GetBlock(TEST_HASH, false)
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
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	// Check that GetBlock returns failure.
	result, err := GetBlock(TEST_HASH, false)
	if err != NotFoundError {
		t.Errorf("Expected NotFoundError, got %v", result)
	}
}

// TestGetBlockCorrupt
//     GetBlock must return an error when a corrupted block is requested
//     (the contents of the file do not checksum to its hash).
//
func TestGetBlockCorrupt(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes and store a corrupt block in one.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, BAD_BLOCK)

	// Check that GetBlock returns failure.
	result, err := GetBlock(TEST_HASH, false)
	if err != DiskHashError {
		t.Errorf("Expected DiskHashError, got %v (buf: %v)", err, result)
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
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	// Check that PutBlock stores the data as expected.
	if err := PutBlock(TEST_BLOCK, TEST_HASH); err != nil {
		t.Fatalf("PutBlock: %v", err)
	}

	vols := KeepVM.Volumes()
	result, err := vols[0].Get(TEST_HASH)
	if err != nil {
		t.Fatalf("Volume #0 Get returned error: %v", err)
	}
	if string(result) != string(TEST_BLOCK) {
		t.Fatalf("PutBlock stored '%s', Get retrieved '%s'",
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
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	vols[0].(*MockVolume).Bad = true

	// Check that PutBlock stores the data as expected.
	if err := PutBlock(TEST_BLOCK, TEST_HASH); err != nil {
		t.Fatalf("PutBlock: %v", err)
	}

	result, err := GetBlock(TEST_HASH, false)
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
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
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	// Check that PutBlock returns the expected error when the hash does
	// not match the block.
	if err := PutBlock(BAD_BLOCK, TEST_HASH); err != RequestHashError {
		t.Error("Expected RequestHashError, got %v", err)
	}

	// Confirm that GetBlock fails to return anything.
	if result, err := GetBlock(TEST_HASH, false); err != NotFoundError {
		t.Errorf("GetBlock succeeded after a corrupt block store (result = %s, err = %v)",
			string(result), err)
	}
}

// TestPutBlockCorrupt
//     PutBlock should overwrite corrupt blocks on disk when given
//     a PUT request with a good block.
//
func TestPutBlockCorrupt(t *testing.T) {
	defer teardown()

	// Create two test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	// Store a corrupted block under TEST_HASH.
	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, BAD_BLOCK)
	if err := PutBlock(TEST_BLOCK, TEST_HASH); err != nil {
		t.Errorf("PutBlock: %v", err)
	}

	// The block on disk should now match TEST_BLOCK.
	if block, err := GetBlock(TEST_HASH, false); err != nil {
		t.Errorf("GetBlock: %v", err)
	} else if bytes.Compare(block, TEST_BLOCK) != 0 {
		t.Errorf("GetBlock returned: '%s'", string(block))
	}
}

// TestPutBlockCollision
//     PutBlock returns a 400 Collision error when attempting to
//     store a block that collides with another block on disk.
//
func TestPutBlockCollision(t *testing.T) {
	defer teardown()

	// These blocks both hash to the MD5 digest cee9a457e790cf20d4bdaa6d69f01e41.
	var b1 = []byte("\x0e0eaU\x9a\xa7\x87\xd0\x0b\xc6\xf7\x0b\xbd\xfe4\x04\xcf\x03e\x9epO\x854\xc0\x0f\xfbe\x9cL\x87@\xcc\x94/\xeb-\xa1\x15\xa3\xf4\x15\\\xbb\x86\x07Is\x86em}\x1f4\xa4 Y\xd7\x8fZ\x8d\xd1\xef")
	var b2 = []byte("\x0e0eaU\x9a\xa7\x87\xd0\x0b\xc6\xf7\x0b\xbd\xfe4\x04\xcf\x03e\x9etO\x854\xc0\x0f\xfbe\x9cL\x87@\xcc\x94/\xeb-\xa1\x15\xa3\xf4\x15\xdc\xbb\x86\x07Is\x86em}\x1f4\xa4 Y\xd7\x8fZ\x8d\xd1\xef")
	var locator = "cee9a457e790cf20d4bdaa6d69f01e41"

	// Prepare two test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	// Store one block, then attempt to store the other. Confirm that
	// PutBlock reported a CollisionError.
	if err := PutBlock(b1, locator); err != nil {
		t.Error(err)
	}
	if err := PutBlock(b2, locator); err == nil {
		t.Error("PutBlock did not report a collision")
	} else if err != CollisionError {
		t.Errorf("PutBlock returned %v", err)
	}
}

// TestPutBlockTouchFails
//     When PutBlock is asked to PUT an existing block, but cannot
//     modify the timestamp, it should write a second block.
//
func TestPutBlockTouchFails(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()
	vols := KeepVM.Volumes()

	// Store a block and then make the underlying volume bad,
	// so a subsequent attempt to update the file timestamp
	// will fail.
	vols[0].Put(TEST_HASH, BAD_BLOCK)
	old_mtime, err := vols[0].Mtime(TEST_HASH)
	if err != nil {
		t.Fatalf("vols[0].Mtime(%s): %s\n", TEST_HASH, err)
	}

	// vols[0].Touch will fail on the next call, so the volume
	// manager will store a copy on vols[1] instead.
	vols[0].(*MockVolume).Touchable = false
	if err := PutBlock(TEST_BLOCK, TEST_HASH); err != nil {
		t.Fatalf("PutBlock: %v", err)
	}
	vols[0].(*MockVolume).Touchable = true

	// Now the mtime on the block on vols[0] should be unchanged, and
	// there should be a copy of the block on vols[1].
	new_mtime, err := vols[0].Mtime(TEST_HASH)
	if err != nil {
		t.Fatalf("vols[0].Mtime(%s): %s\n", TEST_HASH, err)
	}
	if !new_mtime.Equal(old_mtime) {
		t.Errorf("mtime was changed on vols[0]:\nold_mtime = %v\nnew_mtime = %v\n",
			old_mtime, new_mtime)
	}
	result, err := vols[1].Get(TEST_HASH)
	if err != nil {
		t.Fatalf("vols[1]: %v", err)
	}
	if bytes.Compare(result, TEST_BLOCK) != 0 {
		t.Errorf("new block does not match test block\nnew block = %v\n", result)
	}
}

// ========================================
// FindKeepVolumes tests.
// ========================================

// TestFindKeepVolumes
//     Confirms that FindKeepVolumes finds tmpfs volumes with "/keep"
//     directories at the top level.
//
func TestFindKeepVolumes(t *testing.T) {
	var tempVols [2]string
	var err error

	defer func() {
		for _, path := range tempVols {
			os.RemoveAll(path)
		}
	}()

	// Create two directories suitable for using as keep volumes.
	for i := range tempVols {
		if tempVols[i], err = ioutil.TempDir("", "findvol"); err != nil {
			t.Fatal(err)
		}
		tempVols[i] = tempVols[i] + "/keep"
		if err = os.Mkdir(tempVols[i], 0755); err != nil {
			t.Fatal(err)
		}
	}

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

// TestIndex
//     Test an /index request.
func TestIndex(t *testing.T) {
	defer teardown()

	// Set up Keep volumes and populate them.
	// Include multiple blocks on different volumes, and
	// some metadata files.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, TEST_BLOCK)
	vols[1].Put(TEST_HASH_2, TEST_BLOCK_2)
	vols[0].Put(TEST_HASH_3, TEST_BLOCK_3)
	vols[0].Put(TEST_HASH+".meta", []byte("metadata"))
	vols[1].Put(TEST_HASH_2+".meta", []byte("metadata"))

	index := vols[0].Index("") + vols[1].Index("")
	index_rows := strings.Split(index, "\n")
	sort.Strings(index_rows)
	sorted_index := strings.Join(index_rows, "\n")
	expected := `^\n` + TEST_HASH + `\+\d+ \d+\n` +
		TEST_HASH_3 + `\+\d+ \d+\n` +
		TEST_HASH_2 + `\+\d+ \d+$`

	match, err := regexp.MatchString(expected, sorted_index)
	if err == nil {
		if !match {
			t.Errorf("IndexLocators returned:\n%s", index)
		}
	} else {
		t.Errorf("regexp.MatchString: %s", err)
	}
}

// TestNodeStatus
//     Test that GetNodeStatus returns valid info about available volumes.
//
//     TODO(twp): set up appropriate interfaces to permit more rigorous
//     testing.
//
func TestNodeStatus(t *testing.T) {
	defer teardown()

	// Set up test Keep volumes with some blocks.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, TEST_BLOCK)
	vols[1].Put(TEST_HASH_2, TEST_BLOCK_2)

	// Get node status and make a basic sanity check.
	st := GetNodeStatus()
	for i := range vols {
		volinfo := st.Volumes[i]
		mtp := volinfo.MountPoint
		if mtp != "/bogo" {
			t.Errorf("GetNodeStatus mount_point %s, expected /bogo", mtp)
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
}

// ========================================
// Helper functions for unit tests.
// ========================================

// MakeTestVolumeManager
//     Creates and returns a RRVolumeManager with the specified number
//     of MockVolumes.
//
func MakeTestVolumeManager(num_volumes int) VolumeManager {
	vols := make([]Volume, num_volumes)
	for i := range vols {
		vols[i] = CreateMockVolume()
	}
	return MakeRRVolumeManager(vols)
}

// teardown
//     Cleanup to perform after each test.
//
func teardown() {
	data_manager_token = ""
	enforce_permissions = false
	PermissionSecret = nil
	KeepVM = nil
}
