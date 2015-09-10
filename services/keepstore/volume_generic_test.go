package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// A TestableVolumeFactory returns a new TestableVolume. The factory
// function, and the TestableVolume it returns, can use t to write
// logs, fail the current test, etc.
type TestableVolumeFactory func(t *testing.T) TestableVolume

// DoGenericVolumeTests runs a set of tests that every TestableVolume
// is expected to pass. It calls factory to create a new
// TestableVolume for each test case, to avoid leaking state between
// tests.
func DoGenericVolumeTests(t *testing.T, factory TestableVolumeFactory) {
	testDeleteNewBlock(t, factory)
	testDeleteOldBlock(t, factory)
}

// Calling Delete() for a block immediately after writing it should
// neither delete the data nor return an error.
func testDeleteNewBlock(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()
	v.Put(TEST_HASH, TEST_BLOCK)

	if err := v.Delete(TEST_HASH); err != nil {
		t.Error(err)
	}
	if data, err := v.Get(TEST_HASH); err != nil {
		t.Error(err)
	} else if bytes.Compare(data, TEST_BLOCK) != 0 {
		t.Error("Block still present, but content is incorrect: %+v != %+v", data, TEST_BLOCK)
	}
}

// Calling Delete() for a block with a timestamp older than
// blob_signature_ttl seconds in the past should delete the data.
func testDeleteOldBlock(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()
	v.Put(TEST_HASH, TEST_BLOCK)
	v.TouchWithDate(TEST_HASH, time.Now().Add(-2*blob_signature_ttl*time.Second))

	if err := v.Delete(TEST_HASH); err != nil {
		t.Error(err)
	}
	if _, err := v.Get(TEST_HASH); err == nil || !os.IsNotExist(err) {
		t.Errorf("os.IsNotExist(%v) should have been true", err.Error())
	}
}
