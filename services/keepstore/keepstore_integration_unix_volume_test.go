package main

import (
	"bytes"
	"os"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
)

func SetupKeepStoreUnixVolumeIntegrationTest(t *testing.T) {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	// Set up Keep unix volumes
	KeepVM = MakeTestUnixVolumeManager(t, 2)
	defer KeepVM.Close()

	// Start api and keep servers
	arvadostest.StartAPI()
	arvadostest.StartKeep()
}

// MakeTestUnixVolumeManager returns a RRVolumeManager
// with the specified number of UnixVolumes.
var testableUnixVols []*TestableUnixVolume

func MakeTestUnixVolumeManager(t *testing.T, numVolumes int) VolumeManager {
	vols := make([]Volume, numVolumes)
	testableUnixVols = make([]*TestableUnixVolume, numVolumes)

	for i := range vols {
		v := NewTestableUnixVolume(t, false, false)
		vols[i] = v
		testableUnixVols[i] = v
	}
	return MakeRRVolumeManager(vols)
}

// Put TestBlock and Get it
func TestPutTestBlock(t *testing.T) {
	SetupKeepStoreUnixVolumeIntegrationTest(t)

	// Check that PutBlock succeeds
	if err := PutBlock(TestBlock, TestHash); err != nil {
		t.Fatalf("Error during PutBlock: %s", err)
	}

	// Check that PutBlock succeeds again even after CompareAndTouch
	if err := PutBlock(TestBlock, TestHash); err != nil {
		t.Fatalf("Error during PutBlock: %s", err)
	}

	// Check that PutBlock stored the data as expected
	buf, err := GetBlock(TestHash)
	if err != nil {
		t.Fatalf("Error during GetBlock for %q: %s", TestHash, err)
	} else if bytes.Compare(buf, TestBlock) != 0 {
		t.Errorf("Get response incorrect. Expected %q; found %q", TestBlock, buf)
	}
}

// UnixVolume -> Compare is falling in infinite loop since EOF is not being
// returned by reader.Read() for empty block resulting in issue #7329.
// Hence invoke PutBlock twice to test that path involving CompareAndTouch
func TestPutEmptyBlock(t *testing.T) {
	SetupKeepStoreUnixVolumeIntegrationTest(t)

	// Check that PutBlock succeeds
	if err := PutBlock(EmptyBlock, EmptyHash); err != nil {
		t.Fatalf("Error during PutBlock: %s", err)
	}

	// Check that PutBlock succeeds again even after CompareAndTouch
	// With #7329 unresovled, this falls in infinite loop in UnixVolume -> Compare method
	if err := PutBlock(EmptyBlock, EmptyHash); err != nil {
		t.Fatalf("Error during PutBlock: %s", err)
	}

	// Check that PutBlock stored the data as expected
	buf, err := GetBlock(EmptyHash)
	if err != nil {
		t.Fatalf("Error during GetBlock for %q: %s", EmptyHash, err)
	} else if bytes.Compare(buf, EmptyBlock) != 0 {
		t.Errorf("Get response incorrect. Expected %q; found %q", EmptyBlock, buf)
	}
}

// PutRaw EmptyHash with bad data (which bypasses hash check)
// and then invoke PutBlock with the correct EmptyBlock.
// Put should succeed and next Get should return EmptyBlock
func TestPutEmptyBlockDiskHashError(t *testing.T) {
	SetupKeepStoreUnixVolumeIntegrationTest(t)

	badEmptyBlock := []byte("verybaddata")

	// Put bad data for EmptyHash in both volumes
	testableUnixVols[0].PutRaw(EmptyHash, badEmptyBlock)
	testableUnixVols[1].PutRaw(EmptyHash, badEmptyBlock)

	// Check that PutBlock with good data succeeds
	if err := PutBlock(EmptyBlock, EmptyHash); err != nil {
		t.Fatalf("Error during PutBlock for %q: %s", EmptyHash, err)
	}

	// Put succeeded and overwrote the badEmptyBlock in one volume,
	// and Get should return the EmptyBlock now, ignoring the bad data.
	buf, err := GetBlock(EmptyHash)
	if err != nil {
		t.Fatalf("Error during GetBlock for %q: %s", EmptyHash, err)
	} else if bytes.Compare(buf, EmptyBlock) != 0 {
		t.Errorf("Get response incorrect. Expected %q; found %q", TestBlock, buf)
	}
}
