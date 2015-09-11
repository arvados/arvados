package main

import (
	"bytes"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// A TestableVolumeFactory returns a new TestableVolume. The factory
// function, and the TestableVolume it returns, can use t to write
// logs, fail the current test, etc.
type TestableVolumeFactory func(t *testing.T) TestableVolume

// DoGenericVolumeTests runs a set of tests that every TestableVolume
// is expected to pass. It calls factory to create a new writable
// TestableVolume for each test case, to avoid leaking state between
// tests.
func DoGenericVolumeTests(t *testing.T, factory TestableVolumeFactory) {
	testGet(t, factory)
	testGetNoSuchBlock(t, factory)

	testCompareSameContent(t, factory)
	testCompareWithDifferentContent(t, factory)
	testCompareWithBadData(t, factory)

	testPutBlockWithSameContent(t, factory)
	testPutBlockWithDifferentContent(t, factory)
	testPutMultipleBlocks(t, factory)

	testPutAndTouch(t, factory)
	testTouchNoSuchBlock(t, factory)

	testMtimeNoSuchBlock(t, factory)

	testIndexTo(t, factory)

	testDeleteNewBlock(t, factory)
	testDeleteOldBlock(t, factory)
	testDeleteNoSuchBlock(t, factory)

	testStatus(t, factory)

	testString(t, factory)

	testWritableTrue(t, factory)

	testGetSerialized(t, factory)
	testPutSerialized(t, factory)
}

// DoGenericReadOnlyVolumeTests runs a set of tests that every
// read-only TestableVolume is expected to pass. It calls factory
// to create a new read-only TestableVolume for each test case,
// to avoid leaking state between tests.
func DoGenericReadOnlyVolumeTests(t *testing.T, factory TestableVolumeFactory) {
	testWritableFalse(t, factory)
	testUpdateReadOnly(t, factory)
}

// Put a test block, get it and verify content
func testGet(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
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

// Invoke get on a block that does not exist in volume; should result in error
func testGetNoSuchBlock(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()
	v.Put(TEST_HASH, TEST_BLOCK)

	if _, err := v.Get(TEST_HASH_2); err == nil {
		t.Errorf("Expected error while getting non-existing block %v", TEST_HASH_2)
	}
}

// Put a test block and compare the locator with same content
func testCompareSameContent(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	v.Put(TEST_HASH, TEST_BLOCK)

	// Compare the block locator with same content
	err := v.Compare(TEST_HASH, TEST_BLOCK)
	if err != nil {
		t.Errorf("Got err %q, expected nil", err)
	}
}

// Put a test block and compare the locator with a different content
// Expect error due to collision
func testCompareWithDifferentContent(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	v.Put(TEST_HASH, TEST_BLOCK)

	// Compare the block locator with different content; collision
	err := v.Compare(TEST_HASH, []byte("baddata"))
	if err == nil {
		t.Errorf("Expected error due to collision")
	}
}

// Put a test block with bad data (hash does not match, but Put does not verify)
// Compare the locator with good data whose has matches with locator
// Expect error due to corruption.
func testCompareWithBadData(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	v.Put(TEST_HASH, []byte("baddata"))

	err := v.Compare(TEST_HASH, TEST_BLOCK)
	if err == nil {
		t.Errorf("Expected error due to corruption")
	}
}

// Put a block and put again with same content
func testPutBlockWithSameContent(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	err := v.Put(TEST_HASH, TEST_BLOCK)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TEST_BLOCK, err)
	}

	err = v.Put(TEST_HASH, TEST_BLOCK)
	if err != nil {
		t.Errorf("Got err putting block second time %q: %q, expected nil", TEST_BLOCK, err)
	}
}

// Put a block and put again with different content
func testPutBlockWithDifferentContent(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	err := v.Put(TEST_HASH, TEST_BLOCK)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TEST_BLOCK, err)
	}

	// Whether Put with the same loc with different content fails or succeeds
	// is implementation dependent. So, just check loc exists after overwriting.
	// We also do not want to see if loc has block1 or block2, for the same reason.
	if err = v.Put(TEST_HASH, TEST_BLOCK_2); err != nil {
		t.Errorf("Got err putting block with different content %q: %q, expected nil", TEST_BLOCK, err)
	}
	if _, err := v.Get(TEST_HASH); err != nil {
		t.Errorf("Got err getting block %q: %q, expected nil", TEST_BLOCK, err)
	}
}

// Put and get multiple blocks
func testPutMultipleBlocks(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	err := v.Put(TEST_HASH, TEST_BLOCK)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TEST_BLOCK, err)
	}

	err = v.Put(TEST_HASH_2, TEST_BLOCK_2)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TEST_BLOCK_2, err)
	}

	err = v.Put(TEST_HASH_3, TEST_BLOCK_3)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TEST_BLOCK_3, err)
	}

	if data, err := v.Get(TEST_HASH); err != nil {
		t.Error(err)
	} else if bytes.Compare(data, TEST_BLOCK) != 0 {
		t.Errorf("Block present, but content is incorrect: Expected: %v  Found: %v", data, TEST_BLOCK)
	}

	if data, err := v.Get(TEST_HASH_2); err != nil {
		t.Error(err)
	} else if bytes.Compare(data, TEST_BLOCK_2) != 0 {
		t.Errorf("Block present, but content is incorrect: Expected: %v  Found: %v", data, TEST_BLOCK_2)
	}

	if data, err := v.Get(TEST_HASH_3); err != nil {
		t.Error(err)
	} else if bytes.Compare(data, TEST_BLOCK_3) != 0 {
		t.Errorf("Block present, but content is incorrect: Expected: %v  Found: %v", data, TEST_BLOCK_3)
	}
}

// testPutAndTouch
//   Test that when applying PUT to a block that already exists,
//   the block's modification time is updated.
func testPutAndTouch(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
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

// Touching a non-existing block should result in error.
func testTouchNoSuchBlock(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	if err := v.Put(TEST_HASH, TEST_BLOCK); err != nil {
		t.Error(err)
	}

	if err := v.Touch(TEST_HASH); err != nil {
		t.Error("Expected error when attempted to touch a non-existing block")
	}
}

// Invoking Mtime on a non-existing block should result in error.
func testMtimeNoSuchBlock(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	if _, err := v.Mtime("12345678901234567890123456789012"); err == nil {
		t.Error("Expected error when updating Mtime on a non-existing block")
	}
}

// Put a few blocks and invoke IndexTo with:
// * no prefix
// * with a prefix
// * with no such prefix
func testIndexTo(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
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
			t.Errorf("Got index %q for prefix %s", string(buf.Bytes()), prefix)
		}
	}

	for _, prefix := range []string{"zero", "zip", "zilch"} {
		buf = new(bytes.Buffer)
		v.IndexTo(prefix, buf)
		if err != nil {
			t.Errorf("Got error on IndexTo with no such prefix %v", err.Error())
		} else if buf.Len() != 0 {
			t.Errorf("Expected empty list for IndexTo with no such prefix %s", prefix)
		}
	}
}

// Calling Delete() for a block immediately after writing it (not old enough)
// should neither delete the data nor return an error.
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

// Calling Delete() for a block that does not exist should result in error.
func testDeleteNoSuchBlock(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()
	v.Put(TEST_HASH, TEST_BLOCK)

	if err := v.Delete(TEST_HASH_2); err == nil {
		t.Errorf("Expected error when attempting to delete a non-existing block")
	}
}

// Invoke Status and verify that VolumeStatus is returned
func testStatus(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	// Get node status and make a basic sanity check.
	status := v.Status()
	if status.DeviceNum == 0 {
		t.Errorf("uninitialized device_num in %v", status)
	}

	if status.BytesFree == 0 {
		t.Errorf("uninitialized bytes_free in %v", status)
	}

	if status.BytesUsed == 0 {
		t.Errorf("uninitialized bytes_used in %v", status)
	}
}

// Invoke String for the volume; expect non-empty result
func testString(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	if id := v.String(); len(id) == 0 {
		t.Error("Got empty string for v.String()")
	}
}

// Verify Writable is true on a writable volume
func testWritableTrue(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	if v.Writable() == false {
		t.Errorf("Expected writable to be true on a writable volume")
	}
}

// Verify Writable is false on a read-only volume
func testWritableFalse(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	if v.Writable() != false {
		t.Errorf("Expected writable to be false on a read-only volume")
	}
}

// Updating, touching, and deleting blocks from a read-only volume result in error.
func testUpdateReadOnly(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
	defer v.Teardown()

	v.PutRaw(TEST_HASH, TEST_BLOCK)

	_, err := v.Get(TEST_HASH)
	if err != nil {
		t.Errorf("got err %v, expected nil", err)
	}

	err = v.Put(TEST_HASH, TEST_BLOCK)
	if err == nil {
		t.Errorf("Expected error when putting block in a read-only volume")
	}

	err = v.Touch(TEST_HASH)
	if err == nil {
		t.Errorf("Expected error when touching block in a read-only volume")
	}

	err = v.Delete(TEST_HASH)
	if err == nil {
		t.Errorf("Expected error when deleting block from a read-only volume")
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

func testGetSerialized(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
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

func testPutSerialized(t *testing.T, factory TestableVolumeFactory) {
	v := factory(t)
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
