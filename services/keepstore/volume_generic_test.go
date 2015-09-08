package main

import (
	"bytes"
	"os"
	"testing"
	"time"
)

type TestableVolumeFactory func(t *testing.T) TestableVolume

func DoGenericVolumeTests(t *testing.T, factory TestableVolumeFactory) {
	testDeleteNewBlock(t, factory)
	testDeleteOldBlock(t, factory)
}

func testDeleteNewBlock(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()
	v.Put(TEST_HASH, TEST_BLOCK)

	if err := v.Delete(TEST_HASH); err != nil {
		t.Error(err)
	}
	// This isn't reported as an error, but the block should not
	// have been deleted: it's newer than blob_signature_ttl.
	if data, err := v.Get(TEST_HASH); err != nil {
		t.Error(err)
	} else if bytes.Compare(data, TEST_BLOCK) != 0 {
		t.Error("Block still present, but content is incorrect: %+v != %+v", data, TEST_BLOCK)
	}
}

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
