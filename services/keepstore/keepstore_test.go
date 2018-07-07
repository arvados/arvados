// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
)

var TestBlock = []byte("The quick brown fox jumps over the lazy dog.")
var TestHash = "e4d909c290d0fb1ca068ffaddf22cbd0"
var TestHashPutResp = "e4d909c290d0fb1ca068ffaddf22cbd0+44\n"

var TestBlock2 = []byte("Pack my box with five dozen liquor jugs.")
var TestHash2 = "f15ac516f788aec4f30932ffb6395c39"

var TestBlock3 = []byte("Now is the time for all good men to come to the aid of their country.")
var TestHash3 = "eed29bbffbc2dbe5e5ee0bb71888e61f"

// BadBlock is used to test collisions and corruption.
// It must not match any test hashes.
var BadBlock = []byte("The magic words are squeamish ossifrage.")

// Empty block
var EmptyHash = "d41d8cd98f00b204e9800998ecf8427e"
var EmptyBlock = []byte("")

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
	defer KeepVM.Close()

	vols := KeepVM.AllReadable()
	if err := vols[1].Put(context.Background(), TestHash, TestBlock); err != nil {
		t.Error(err)
	}

	// Check that GetBlock returns success.
	buf := make([]byte, BlockSize)
	size, err := GetBlock(context.Background(), TestHash, buf, nil)
	if err != nil {
		t.Errorf("GetBlock error: %s", err)
	}
	if bytes.Compare(buf[:size], TestBlock) != 0 {
		t.Errorf("got %v, expected %v", buf[:size], TestBlock)
	}
}

// TestGetBlockMissing
//     GetBlock must return an error when the block is not found.
//
func TestGetBlockMissing(t *testing.T) {
	defer teardown()

	// Create two empty test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	// Check that GetBlock returns failure.
	buf := make([]byte, BlockSize)
	size, err := GetBlock(context.Background(), TestHash, buf, nil)
	if err != NotFoundError {
		t.Errorf("Expected NotFoundError, got %v, err %v", buf[:size], err)
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
	defer KeepVM.Close()

	vols := KeepVM.AllReadable()
	vols[0].Put(context.Background(), TestHash, BadBlock)

	// Check that GetBlock returns failure.
	buf := make([]byte, BlockSize)
	size, err := GetBlock(context.Background(), TestHash, buf, nil)
	if err != DiskHashError {
		t.Errorf("Expected DiskHashError, got %v (buf: %v)", err, buf[:size])
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
	defer KeepVM.Close()

	// Check that PutBlock stores the data as expected.
	if n, err := PutBlock(context.Background(), TestBlock, TestHash); err != nil || n < 1 {
		t.Fatalf("PutBlock: n %d err %v", n, err)
	}

	vols := KeepVM.AllReadable()
	buf := make([]byte, BlockSize)
	n, err := vols[1].Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Fatalf("Volume #0 Get returned error: %v", err)
	}
	if string(buf[:n]) != string(TestBlock) {
		t.Fatalf("PutBlock stored '%s', Get retrieved '%s'",
			string(TestBlock), string(buf[:n]))
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
	defer KeepVM.Close()

	vols := KeepVM.AllWritable()
	vols[0].(*MockVolume).Bad = true

	// Check that PutBlock stores the data as expected.
	if n, err := PutBlock(context.Background(), TestBlock, TestHash); err != nil || n < 1 {
		t.Fatalf("PutBlock: n %d err %v", n, err)
	}

	buf := make([]byte, BlockSize)
	size, err := GetBlock(context.Background(), TestHash, buf, nil)
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
	}
	if bytes.Compare(buf[:size], TestBlock) != 0 {
		t.Fatalf("PutBlock stored %+q, GetBlock retrieved %+q",
			TestBlock, buf[:size])
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
	defer KeepVM.Close()

	// Check that PutBlock returns the expected error when the hash does
	// not match the block.
	if _, err := PutBlock(context.Background(), BadBlock, TestHash); err != RequestHashError {
		t.Errorf("Expected RequestHashError, got %v", err)
	}

	// Confirm that GetBlock fails to return anything.
	if result, err := GetBlock(context.Background(), TestHash, make([]byte, BlockSize), nil); err != NotFoundError {
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
	defer KeepVM.Close()

	// Store a corrupted block under TestHash.
	vols := KeepVM.AllWritable()
	vols[0].Put(context.Background(), TestHash, BadBlock)
	if n, err := PutBlock(context.Background(), TestBlock, TestHash); err != nil || n < 1 {
		t.Errorf("PutBlock: n %d err %v", n, err)
	}

	// The block on disk should now match TestBlock.
	buf := make([]byte, BlockSize)
	if size, err := GetBlock(context.Background(), TestHash, buf, nil); err != nil {
		t.Errorf("GetBlock: %v", err)
	} else if bytes.Compare(buf[:size], TestBlock) != 0 {
		t.Errorf("Got %+q, expected %+q", buf[:size], TestBlock)
	}
}

// TestPutBlockCollision
//     PutBlock returns a 400 Collision error when attempting to
//     store a block that collides with another block on disk.
//
func TestPutBlockCollision(t *testing.T) {
	defer teardown()

	// These blocks both hash to the MD5 digest cee9a457e790cf20d4bdaa6d69f01e41.
	b1 := arvadostest.MD5CollisionData[0]
	b2 := arvadostest.MD5CollisionData[1]
	locator := arvadostest.MD5CollisionMD5

	// Prepare two test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	// Store one block, then attempt to store the other. Confirm that
	// PutBlock reported a CollisionError.
	if _, err := PutBlock(context.Background(), b1, locator); err != nil {
		t.Error(err)
	}
	if _, err := PutBlock(context.Background(), b2, locator); err == nil {
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
	defer KeepVM.Close()
	vols := KeepVM.AllWritable()

	// Store a block and then make the underlying volume bad,
	// so a subsequent attempt to update the file timestamp
	// will fail.
	vols[0].Put(context.Background(), TestHash, BadBlock)
	oldMtime, err := vols[0].Mtime(TestHash)
	if err != nil {
		t.Fatalf("vols[0].Mtime(%s): %s\n", TestHash, err)
	}

	// vols[0].Touch will fail on the next call, so the volume
	// manager will store a copy on vols[1] instead.
	vols[0].(*MockVolume).Touchable = false
	if n, err := PutBlock(context.Background(), TestBlock, TestHash); err != nil || n < 1 {
		t.Fatalf("PutBlock: n %d err %v", n, err)
	}
	vols[0].(*MockVolume).Touchable = true

	// Now the mtime on the block on vols[0] should be unchanged, and
	// there should be a copy of the block on vols[1].
	newMtime, err := vols[0].Mtime(TestHash)
	if err != nil {
		t.Fatalf("vols[0].Mtime(%s): %s\n", TestHash, err)
	}
	if !newMtime.Equal(oldMtime) {
		t.Errorf("mtime was changed on vols[0]:\noldMtime = %v\nnewMtime = %v\n",
			oldMtime, newMtime)
	}
	buf := make([]byte, BlockSize)
	n, err := vols[1].Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Fatalf("vols[1]: %v", err)
	}
	if bytes.Compare(buf[:n], TestBlock) != 0 {
		t.Errorf("new block does not match test block\nnew block = %v\n", buf[:n])
	}
}

func TestDiscoverTmpfs(t *testing.T) {
	var tempVols [4]string
	var err error

	// Create some directories suitable for using as keep volumes.
	for i := range tempVols {
		if tempVols[i], err = ioutil.TempDir("", "findvol"); err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempVols[i])
		tempVols[i] = tempVols[i] + "/keep"
		if err = os.Mkdir(tempVols[i], 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Set up a bogus ProcMounts file.
	f, err := ioutil.TempFile("", "keeptest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	for i, vol := range tempVols {
		// Add readonly mount points at odd indexes.
		var opts string
		switch i % 2 {
		case 0:
			opts = "rw,nosuid,nodev,noexec"
		case 1:
			opts = "nosuid,nodev,noexec,ro"
		}
		fmt.Fprintf(f, "tmpfs %s tmpfs %s 0 0\n", path.Dir(vol), opts)
	}
	f.Close()
	ProcMounts = f.Name()

	cfg := &Config{}
	added := (&unixVolumeAdder{cfg}).Discover()

	if added != len(cfg.Volumes) {
		t.Errorf("Discover returned %d, but added %d volumes",
			added, len(cfg.Volumes))
	}
	if added != len(tempVols) {
		t.Errorf("Discover returned %d but we set up %d volumes",
			added, len(tempVols))
	}
	for i, tmpdir := range tempVols {
		if tmpdir != cfg.Volumes[i].(*UnixVolume).Root {
			t.Errorf("Discover returned %s, expected %s\n",
				cfg.Volumes[i].(*UnixVolume).Root, tmpdir)
		}
		if expectReadonly := i%2 == 1; expectReadonly != cfg.Volumes[i].(*UnixVolume).ReadOnly {
			t.Errorf("Discover added %s with readonly=%v, should be %v",
				tmpdir, !expectReadonly, expectReadonly)
		}
	}
}

func TestDiscoverNone(t *testing.T) {
	defer teardown()

	// Set up a bogus ProcMounts file with no Keep vols.
	f, err := ioutil.TempFile("", "keeptest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	fmt.Fprintln(f, "rootfs / rootfs opts 0 0")
	fmt.Fprintln(f, "sysfs /sys sysfs opts 0 0")
	fmt.Fprintln(f, "proc /proc proc opts 0 0")
	fmt.Fprintln(f, "udev /dev devtmpfs opts 0 0")
	fmt.Fprintln(f, "devpts /dev/pts devpts opts 0 0")
	f.Close()
	ProcMounts = f.Name()

	cfg := &Config{}
	added := (&unixVolumeAdder{cfg}).Discover()
	if added != 0 || len(cfg.Volumes) != 0 {
		t.Fatalf("got %d, %v; expected 0, []", added, cfg.Volumes)
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
	defer KeepVM.Close()

	vols := KeepVM.AllReadable()
	vols[0].Put(context.Background(), TestHash, TestBlock)
	vols[1].Put(context.Background(), TestHash2, TestBlock2)
	vols[0].Put(context.Background(), TestHash3, TestBlock3)
	vols[0].Put(context.Background(), TestHash+".meta", []byte("metadata"))
	vols[1].Put(context.Background(), TestHash2+".meta", []byte("metadata"))

	buf := new(bytes.Buffer)
	vols[0].IndexTo("", buf)
	vols[1].IndexTo("", buf)
	indexRows := strings.Split(string(buf.Bytes()), "\n")
	sort.Strings(indexRows)
	sortedIndex := strings.Join(indexRows, "\n")
	expected := `^\n` + TestHash + `\+\d+ \d+\n` +
		TestHash3 + `\+\d+ \d+\n` +
		TestHash2 + `\+\d+ \d+$`

	match, err := regexp.MatchString(expected, sortedIndex)
	if err == nil {
		if !match {
			t.Errorf("IndexLocators returned:\n%s", string(buf.Bytes()))
		}
	} else {
		t.Errorf("regexp.MatchString: %s", err)
	}
}

// ========================================
// Helper functions for unit tests.
// ========================================

// MakeTestVolumeManager returns a RRVolumeManager with the specified
// number of MockVolumes.
func MakeTestVolumeManager(numVolumes int) VolumeManager {
	vols := make([]Volume, numVolumes)
	for i := range vols {
		vols[i] = CreateMockVolume()
	}
	return MakeRRVolumeManager(vols)
}

// teardown cleans up after each test.
func teardown() {
	theConfig.systemAuthToken = ""
	theConfig.RequireSignatures = false
	theConfig.blobSigningKey = nil
	KeepVM = nil
}
