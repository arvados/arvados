// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

type TB interface {
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

// A TestableVolumeFactory returns a new TestableVolume. The factory
// function, and the TestableVolume it returns, can use "t" to write
// logs, fail the current test, etc.
type TestableVolumeFactory func(t TB, cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) TestableVolume

// DoGenericVolumeTests runs a set of tests that every TestableVolume
// is expected to pass. It calls factory to create a new TestableVolume
// for each test case, to avoid leaking state between tests.
func DoGenericVolumeTests(t TB, readonly bool, factory TestableVolumeFactory) {
	var s genericVolumeSuite
	s.volume.ReadOnly = readonly

	s.testGet(t, factory)
	s.testGetNoSuchBlock(t, factory)

	s.testCompareNonexistent(t, factory)
	s.testCompareSameContent(t, factory, TestHash, TestBlock)
	s.testCompareSameContent(t, factory, EmptyHash, EmptyBlock)
	s.testCompareWithCollision(t, factory, TestHash, TestBlock, []byte("baddata"))
	s.testCompareWithCollision(t, factory, TestHash, TestBlock, EmptyBlock)
	s.testCompareWithCollision(t, factory, EmptyHash, EmptyBlock, TestBlock)
	s.testCompareWithCorruptStoredData(t, factory, TestHash, TestBlock, []byte("baddata"))
	s.testCompareWithCorruptStoredData(t, factory, TestHash, TestBlock, EmptyBlock)
	s.testCompareWithCorruptStoredData(t, factory, EmptyHash, EmptyBlock, []byte("baddata"))

	if !readonly {
		s.testPutBlockWithSameContent(t, factory, TestHash, TestBlock)
		s.testPutBlockWithSameContent(t, factory, EmptyHash, EmptyBlock)
		s.testPutBlockWithDifferentContent(t, factory, arvadostest.MD5CollisionMD5, arvadostest.MD5CollisionData[0], arvadostest.MD5CollisionData[1])
		s.testPutBlockWithDifferentContent(t, factory, arvadostest.MD5CollisionMD5, EmptyBlock, arvadostest.MD5CollisionData[0])
		s.testPutBlockWithDifferentContent(t, factory, arvadostest.MD5CollisionMD5, arvadostest.MD5CollisionData[0], EmptyBlock)
		s.testPutBlockWithDifferentContent(t, factory, EmptyHash, EmptyBlock, arvadostest.MD5CollisionData[0])
		s.testPutMultipleBlocks(t, factory)

		s.testPutAndTouch(t, factory)
	}
	s.testTouchNoSuchBlock(t, factory)

	s.testMtimeNoSuchBlock(t, factory)

	s.testIndexTo(t, factory)

	if !readonly {
		s.testDeleteNewBlock(t, factory)
		s.testDeleteOldBlock(t, factory)
	}
	s.testDeleteNoSuchBlock(t, factory)

	s.testStatus(t, factory)

	s.testMetrics(t, readonly, factory)

	s.testString(t, factory)

	if readonly {
		s.testUpdateReadOnly(t, factory)
	}

	s.testGetConcurrent(t, factory)
	if !readonly {
		s.testPutConcurrent(t, factory)

		s.testPutFullBlock(t, factory)
	}

	s.testTrashUntrash(t, readonly, factory)
	s.testTrashEmptyTrashUntrash(t, factory)
}

type genericVolumeSuite struct {
	cluster  *arvados.Cluster
	volume   arvados.Volume
	logger   logrus.FieldLogger
	metrics  *volumeMetricsVecs
	registry *prometheus.Registry
}

func (s *genericVolumeSuite) setup(t TB) {
	s.cluster = testCluster(t)
	s.logger = ctxlog.TestLogger(t)
	s.registry = prometheus.NewRegistry()
	s.metrics = newVolumeMetricsVecs(s.registry)
}

func (s *genericVolumeSuite) newVolume(t TB, factory TestableVolumeFactory) TestableVolume {
	return factory(t, s.cluster, s.volume, s.logger, s.metrics)
}

// Put a test block, get it and verify content
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testGet(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.PutRaw(TestHash, TestBlock)

	buf := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(buf[:n], TestBlock) != 0 {
		t.Errorf("expected %s, got %s", string(TestBlock), string(buf))
	}
}

// Invoke get on a block that does not exist in volume; should result in error
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testGetNoSuchBlock(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	buf := make([]byte, BlockSize)
	if _, err := v.Get(context.Background(), TestHash2, buf); err == nil {
		t.Errorf("Expected error while getting non-existing block %v", TestHash2)
	}
}

// Compare() should return os.ErrNotExist if the block does not exist.
// Otherwise, writing new data causes CompareAndTouch() to generate
// error logs even though everything is working fine.
func (s *genericVolumeSuite) testCompareNonexistent(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	err := v.Compare(context.Background(), TestHash, TestBlock)
	if err != os.ErrNotExist {
		t.Errorf("Got err %T %q, expected os.ErrNotExist", err, err)
	}
}

// Put a test block and compare the locator with same content
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testCompareSameContent(t TB, factory TestableVolumeFactory, testHash string, testData []byte) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.PutRaw(testHash, testData)

	// Compare the block locator with same content
	err := v.Compare(context.Background(), testHash, testData)
	if err != nil {
		t.Errorf("Got err %q, expected nil", err)
	}
}

// Test behavior of Compare() when stored data matches expected
// checksum but differs from new data we need to store. Requires
// testHash = md5(testDataA).
//
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testCompareWithCollision(t TB, factory TestableVolumeFactory, testHash string, testDataA, testDataB []byte) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.PutRaw(testHash, testDataA)

	// Compare the block locator with different content; collision
	err := v.Compare(context.Background(), TestHash, testDataB)
	if err == nil {
		t.Errorf("Got err nil, expected error due to collision")
	}
}

// Test behavior of Compare() when stored data has become
// corrupted. Requires testHash = md5(testDataA) != md5(testDataB).
//
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testCompareWithCorruptStoredData(t TB, factory TestableVolumeFactory, testHash string, testDataA, testDataB []byte) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.PutRaw(TestHash, testDataB)

	err := v.Compare(context.Background(), testHash, testDataA)
	if err == nil || err == CollisionError {
		t.Errorf("Got err %+v, expected non-collision error", err)
	}
}

// Put a block and put again with same content
// Test is intended for only writable volumes
func (s *genericVolumeSuite) testPutBlockWithSameContent(t TB, factory TestableVolumeFactory, testHash string, testData []byte) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	err := v.Put(context.Background(), testHash, testData)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TestBlock, err)
	}

	err = v.Put(context.Background(), testHash, testData)
	if err != nil {
		t.Errorf("Got err putting block second time %q: %q, expected nil", TestBlock, err)
	}
}

// Put a block and put again with different content
// Test is intended for only writable volumes
func (s *genericVolumeSuite) testPutBlockWithDifferentContent(t TB, factory TestableVolumeFactory, testHash string, testDataA, testDataB []byte) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.PutRaw(testHash, testDataA)

	putErr := v.Put(context.Background(), testHash, testDataB)
	buf := make([]byte, BlockSize)
	n, getErr := v.Get(context.Background(), testHash, buf)
	if putErr == nil {
		// Put must not return a nil error unless it has
		// overwritten the existing data.
		if bytes.Compare(buf[:n], testDataB) != 0 {
			t.Errorf("Put succeeded but Get returned %+q, expected %+q", buf[:n], testDataB)
		}
	} else {
		// It is permissible for Put to fail, but it must
		// leave us with either the original data, the new
		// data, or nothing at all.
		if getErr == nil && bytes.Compare(buf[:n], testDataA) != 0 && bytes.Compare(buf[:n], testDataB) != 0 {
			t.Errorf("Put failed but Get returned %+q, which is neither %+q nor %+q", buf[:n], testDataA, testDataB)
		}
	}
}

// Put and get multiple blocks
// Test is intended for only writable volumes
func (s *genericVolumeSuite) testPutMultipleBlocks(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	err := v.Put(context.Background(), TestHash, TestBlock)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TestBlock, err)
	}

	err = v.Put(context.Background(), TestHash2, TestBlock2)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TestBlock2, err)
	}

	err = v.Put(context.Background(), TestHash3, TestBlock3)
	if err != nil {
		t.Errorf("Got err putting block %q: %q, expected nil", TestBlock3, err)
	}

	data := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), TestHash, data)
	if err != nil {
		t.Error(err)
	} else {
		if bytes.Compare(data[:n], TestBlock) != 0 {
			t.Errorf("Block present, but got %+q, expected %+q", data[:n], TestBlock)
		}
	}

	n, err = v.Get(context.Background(), TestHash2, data)
	if err != nil {
		t.Error(err)
	} else {
		if bytes.Compare(data[:n], TestBlock2) != 0 {
			t.Errorf("Block present, but got %+q, expected %+q", data[:n], TestBlock2)
		}
	}

	n, err = v.Get(context.Background(), TestHash3, data)
	if err != nil {
		t.Error(err)
	} else {
		if bytes.Compare(data[:n], TestBlock3) != 0 {
			t.Errorf("Block present, but to %+q, expected %+q", data[:n], TestBlock3)
		}
	}
}

// testPutAndTouch
//   Test that when applying PUT to a block that already exists,
//   the block's modification time is updated.
// Test is intended for only writable volumes
func (s *genericVolumeSuite) testPutAndTouch(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	if err := v.Put(context.Background(), TestHash, TestBlock); err != nil {
		t.Error(err)
	}

	// We'll verify { t0 < threshold < t1 }, where t0 is the
	// existing block's timestamp on disk before Put() and t1 is
	// its timestamp after Put().
	threshold := time.Now().Add(-time.Second)

	// Set the stored block's mtime far enough in the past that we
	// can see the difference between "timestamp didn't change"
	// and "timestamp granularity is too low".
	v.TouchWithDate(TestHash, time.Now().Add(-20*time.Second))

	// Make sure v.Mtime() agrees the above Utime really worked.
	if t0, err := v.Mtime(TestHash); err != nil || t0.IsZero() || !t0.Before(threshold) {
		t.Errorf("Setting mtime failed: %v, %v", t0, err)
	}

	// Write the same block again.
	if err := v.Put(context.Background(), TestHash, TestBlock); err != nil {
		t.Error(err)
	}

	// Verify threshold < t1
	if t1, err := v.Mtime(TestHash); err != nil {
		t.Error(err)
	} else if t1.Before(threshold) {
		t.Errorf("t1 %v should be >= threshold %v after v.Put ", t1, threshold)
	}
}

// Touching a non-existing block should result in error.
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testTouchNoSuchBlock(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	if err := v.Touch(TestHash); err == nil {
		t.Error("Expected error when attempted to touch a non-existing block")
	}
}

// Invoking Mtime on a non-existing block should result in error.
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testMtimeNoSuchBlock(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	if _, err := v.Mtime("12345678901234567890123456789012"); err == nil {
		t.Error("Expected error when updating Mtime on a non-existing block")
	}
}

// Put a few blocks and invoke IndexTo with:
// * no prefix
// * with a prefix
// * with no such prefix
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testIndexTo(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	// minMtime and maxMtime are the minimum and maximum
	// acceptable values the index can report for our test
	// blocks. 1-second precision is acceptable.
	minMtime := time.Now().UTC().UnixNano()
	minMtime -= minMtime % 1e9

	v.PutRaw(TestHash, TestBlock)
	v.PutRaw(TestHash2, TestBlock2)
	v.PutRaw(TestHash3, TestBlock3)

	maxMtime := time.Now().UTC().UnixNano()
	if maxMtime%1e9 > 0 {
		maxMtime -= maxMtime % 1e9
		maxMtime += 1e9
	}

	// Blocks whose names aren't Keep hashes should be omitted from
	// index
	v.PutRaw("fffffffffnotreallyahashfffffffff", nil)
	v.PutRaw("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", nil)
	v.PutRaw("f0000000000000000000000000000000f", nil)
	v.PutRaw("f00", nil)

	buf := new(bytes.Buffer)
	v.IndexTo("", buf)
	indexRows := strings.Split(string(buf.Bytes()), "\n")
	sort.Strings(indexRows)
	sortedIndex := strings.Join(indexRows, "\n")
	m := regexp.MustCompile(
		`^\n` + TestHash + `\+\d+ (\d+)\n` +
			TestHash3 + `\+\d+ \d+\n` +
			TestHash2 + `\+\d+ \d+$`,
	).FindStringSubmatch(sortedIndex)
	if m == nil {
		t.Errorf("Got index %q for empty prefix", sortedIndex)
	} else {
		mtime, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			t.Error(err)
		} else if mtime < minMtime || mtime > maxMtime {
			t.Errorf("got %d for TestHash timestamp, expected %d <= t <= %d",
				mtime, minMtime, maxMtime)
		}
	}

	for _, prefix := range []string{"f", "f15", "f15ac"} {
		buf = new(bytes.Buffer)
		v.IndexTo(prefix, buf)

		m, err := regexp.MatchString(`^`+TestHash2+`\+\d+ \d+\n$`, string(buf.Bytes()))
		if err != nil {
			t.Error(err)
		} else if !m {
			t.Errorf("Got index %q for prefix %s", string(buf.Bytes()), prefix)
		}
	}

	for _, prefix := range []string{"zero", "zip", "zilch"} {
		buf = new(bytes.Buffer)
		err := v.IndexTo(prefix, buf)
		if err != nil {
			t.Errorf("Got error on IndexTo with no such prefix %v", err.Error())
		} else if buf.Len() != 0 {
			t.Errorf("Expected empty list for IndexTo with no such prefix %s", prefix)
		}
	}
}

// Calling Delete() for a block immediately after writing it (not old enough)
// should neither delete the data nor return an error.
// Test is intended for only writable volumes
func (s *genericVolumeSuite) testDeleteNewBlock(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	s.cluster.Collections.BlobSigningTTL.Set("5m")
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.Put(context.Background(), TestHash, TestBlock)

	if err := v.Trash(TestHash); err != nil {
		t.Error(err)
	}
	data := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), TestHash, data)
	if err != nil {
		t.Error(err)
	} else if bytes.Compare(data[:n], TestBlock) != 0 {
		t.Errorf("Got data %+q, expected %+q", data[:n], TestBlock)
	}
}

// Calling Delete() for a block with a timestamp older than
// BlobSigningTTL seconds in the past should delete the data.  Test is
// intended for only writable volumes
func (s *genericVolumeSuite) testDeleteOldBlock(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	s.cluster.Collections.BlobSigningTTL.Set("5m")
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.Put(context.Background(), TestHash, TestBlock)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	if err := v.Trash(TestHash); err != nil {
		t.Error(err)
	}
	data := make([]byte, BlockSize)
	if _, err := v.Get(context.Background(), TestHash, data); err == nil || !os.IsNotExist(err) {
		t.Errorf("os.IsNotExist(%v) should have been true", err)
	}

	_, err := v.Mtime(TestHash)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	err = v.Compare(context.Background(), TestHash, TestBlock)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	indexBuf := new(bytes.Buffer)
	v.IndexTo("", indexBuf)
	if strings.Contains(string(indexBuf.Bytes()), TestHash) {
		t.Fatalf("Found trashed block in IndexTo")
	}

	err = v.Touch(TestHash)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}
}

// Calling Delete() for a block that does not exist should result in error.
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testDeleteNoSuchBlock(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	if err := v.Trash(TestHash2); err == nil {
		t.Errorf("Expected error when attempting to delete a non-existing block")
	}
}

// Invoke Status and verify that VolumeStatus is returned
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testStatus(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
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

func getValueFrom(cv *prometheus.CounterVec, lbls prometheus.Labels) float64 {
	c, _ := cv.GetMetricWith(lbls)
	pb := &dto.Metric{}
	c.Write(pb)
	return pb.GetCounter().GetValue()
}

func (s *genericVolumeSuite) testMetrics(t TB, readonly bool, factory TestableVolumeFactory) {
	var err error

	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	opsC, _, ioC := s.metrics.getCounterVecsFor(prometheus.Labels{"device_id": v.GetDeviceID()})

	if ioC == nil {
		t.Error("ioBytes CounterVec is nil")
		return
	}

	if getValueFrom(ioC, prometheus.Labels{"direction": "out"})+
		getValueFrom(ioC, prometheus.Labels{"direction": "in"}) > 0 {
		t.Error("ioBytes counter should be zero")
	}

	if opsC == nil {
		t.Error("opsCounter CounterVec is nil")
		return
	}

	var c, writeOpCounter, readOpCounter float64

	readOpType, writeOpType := v.ReadWriteOperationLabelValues()
	writeOpCounter = getValueFrom(opsC, prometheus.Labels{"operation": writeOpType})
	readOpCounter = getValueFrom(opsC, prometheus.Labels{"operation": readOpType})

	// Test Put if volume is writable
	if !readonly {
		err = v.Put(context.Background(), TestHash, TestBlock)
		if err != nil {
			t.Errorf("Got err putting block %q: %q, expected nil", TestBlock, err)
		}
		// Check that the write operations counter increased
		c = getValueFrom(opsC, prometheus.Labels{"operation": writeOpType})
		if c <= writeOpCounter {
			t.Error("Operation(s) not counted on Put")
		}
		// Check that bytes counter is > 0
		if getValueFrom(ioC, prometheus.Labels{"direction": "out"}) == 0 {
			t.Error("ioBytes{direction=out} counter shouldn't be zero")
		}
	} else {
		v.PutRaw(TestHash, TestBlock)
	}

	buf := make([]byte, BlockSize)
	_, err = v.Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the operations counter increased
	c = getValueFrom(opsC, prometheus.Labels{"operation": readOpType})
	if c <= readOpCounter {
		t.Error("Operation(s) not counted on Get")
	}
	// Check that the bytes "in" counter is > 0
	if getValueFrom(ioC, prometheus.Labels{"direction": "in"}) == 0 {
		t.Error("ioBytes{direction=in} counter shouldn't be zero")
	}
}

// Invoke String for the volume; expect non-empty result
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testString(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	if id := v.String(); len(id) == 0 {
		t.Error("Got empty string for v.String()")
	}
}

// Putting, updating, touching, and deleting blocks from a read-only volume result in error.
// Test is intended for only read-only volumes
func (s *genericVolumeSuite) testUpdateReadOnly(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.PutRaw(TestHash, TestBlock)
	buf := make([]byte, BlockSize)

	// Get from read-only volume should succeed
	_, err := v.Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Errorf("got err %v, expected nil", err)
	}

	// Put a new block to read-only volume should result in error
	err = v.Put(context.Background(), TestHash2, TestBlock2)
	if err == nil {
		t.Errorf("Expected error when putting block in a read-only volume")
	}
	_, err = v.Get(context.Background(), TestHash2, buf)
	if err == nil {
		t.Errorf("Expected error when getting block whose put in read-only volume failed")
	}

	// Touch a block in read-only volume should result in error
	err = v.Touch(TestHash)
	if err == nil {
		t.Errorf("Expected error when touching block in a read-only volume")
	}

	// Delete a block from a read-only volume should result in error
	err = v.Trash(TestHash)
	if err == nil {
		t.Errorf("Expected error when deleting block from a read-only volume")
	}

	// Overwriting an existing block in read-only volume should result in error
	err = v.Put(context.Background(), TestHash, TestBlock)
	if err == nil {
		t.Errorf("Expected error when putting block in a read-only volume")
	}
}

// Launch concurrent Gets
// Test should pass for both writable and read-only volumes
func (s *genericVolumeSuite) testGetConcurrent(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	v.PutRaw(TestHash, TestBlock)
	v.PutRaw(TestHash2, TestBlock2)
	v.PutRaw(TestHash3, TestBlock3)

	sem := make(chan int)
	go func() {
		buf := make([]byte, BlockSize)
		n, err := v.Get(context.Background(), TestHash, buf)
		if err != nil {
			t.Errorf("err1: %v", err)
		}
		if bytes.Compare(buf[:n], TestBlock) != 0 {
			t.Errorf("buf should be %s, is %s", string(TestBlock), string(buf[:n]))
		}
		sem <- 1
	}()

	go func() {
		buf := make([]byte, BlockSize)
		n, err := v.Get(context.Background(), TestHash2, buf)
		if err != nil {
			t.Errorf("err2: %v", err)
		}
		if bytes.Compare(buf[:n], TestBlock2) != 0 {
			t.Errorf("buf should be %s, is %s", string(TestBlock2), string(buf[:n]))
		}
		sem <- 1
	}()

	go func() {
		buf := make([]byte, BlockSize)
		n, err := v.Get(context.Background(), TestHash3, buf)
		if err != nil {
			t.Errorf("err3: %v", err)
		}
		if bytes.Compare(buf[:n], TestBlock3) != 0 {
			t.Errorf("buf should be %s, is %s", string(TestBlock3), string(buf[:n]))
		}
		sem <- 1
	}()

	// Wait for all goroutines to finish
	for done := 0; done < 3; done++ {
		<-sem
	}
}

// Launch concurrent Puts
// Test is intended for only writable volumes
func (s *genericVolumeSuite) testPutConcurrent(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	sem := make(chan int)
	go func(sem chan int) {
		err := v.Put(context.Background(), TestHash, TestBlock)
		if err != nil {
			t.Errorf("err1: %v", err)
		}
		sem <- 1
	}(sem)

	go func(sem chan int) {
		err := v.Put(context.Background(), TestHash2, TestBlock2)
		if err != nil {
			t.Errorf("err2: %v", err)
		}
		sem <- 1
	}(sem)

	go func(sem chan int) {
		err := v.Put(context.Background(), TestHash3, TestBlock3)
		if err != nil {
			t.Errorf("err3: %v", err)
		}
		sem <- 1
	}(sem)

	// Wait for all goroutines to finish
	for done := 0; done < 3; done++ {
		<-sem
	}

	// Double check that we actually wrote the blocks we expected to write.
	buf := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Errorf("Get #1: %v", err)
	}
	if bytes.Compare(buf[:n], TestBlock) != 0 {
		t.Errorf("Get #1: expected %s, got %s", string(TestBlock), string(buf[:n]))
	}

	n, err = v.Get(context.Background(), TestHash2, buf)
	if err != nil {
		t.Errorf("Get #2: %v", err)
	}
	if bytes.Compare(buf[:n], TestBlock2) != 0 {
		t.Errorf("Get #2: expected %s, got %s", string(TestBlock2), string(buf[:n]))
	}

	n, err = v.Get(context.Background(), TestHash3, buf)
	if err != nil {
		t.Errorf("Get #3: %v", err)
	}
	if bytes.Compare(buf[:n], TestBlock3) != 0 {
		t.Errorf("Get #3: expected %s, got %s", string(TestBlock3), string(buf[:n]))
	}
}

// Write and read back a full size block
func (s *genericVolumeSuite) testPutFullBlock(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	wdata := make([]byte, BlockSize)
	wdata[0] = 'a'
	wdata[BlockSize-1] = 'z'
	hash := fmt.Sprintf("%x", md5.Sum(wdata))
	err := v.Put(context.Background(), hash, wdata)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), hash, buf)
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(buf[:n], wdata) != 0 {
		t.Error("buf %+q != wdata %+q", buf[:n], wdata)
	}
}

// With BlobTrashLifetime != 0, perform:
// Trash an old block - which either raises ErrNotImplemented or succeeds
// Untrash -  which either raises ErrNotImplemented or succeeds
// Get - which must succeed
func (s *genericVolumeSuite) testTrashUntrash(t TB, readonly bool, factory TestableVolumeFactory) {
	s.setup(t)
	s.cluster.Collections.BlobTrashLifetime.Set("1h")
	v := s.newVolume(t, factory)
	defer v.Teardown()

	// put block and backdate it
	v.PutRaw(TestHash, TestBlock)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	buf := make([]byte, BlockSize)
	n, err := v.Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf[:n], TestBlock) != 0 {
		t.Errorf("Got data %+q, expected %+q", buf[:n], TestBlock)
	}

	// Trash
	err = v.Trash(TestHash)
	if readonly {
		if err != MethodDisabledError {
			t.Fatal(err)
		}
	} else if err != nil {
		if err != ErrNotImplemented {
			t.Fatal(err)
		}
	} else {
		_, err = v.Get(context.Background(), TestHash, buf)
		if err == nil || !os.IsNotExist(err) {
			t.Errorf("os.IsNotExist(%v) should have been true", err)
		}

		// Untrash
		err = v.Untrash(TestHash)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Get the block - after trash and untrash sequence
	n, err = v.Get(context.Background(), TestHash, buf)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(buf[:n], TestBlock) != 0 {
		t.Errorf("Got data %+q, expected %+q", buf[:n], TestBlock)
	}
}

func (s *genericVolumeSuite) testTrashEmptyTrashUntrash(t TB, factory TestableVolumeFactory) {
	s.setup(t)
	v := s.newVolume(t, factory)
	defer v.Teardown()

	checkGet := func() error {
		buf := make([]byte, BlockSize)
		n, err := v.Get(context.Background(), TestHash, buf)
		if err != nil {
			return err
		}
		if bytes.Compare(buf[:n], TestBlock) != 0 {
			t.Fatalf("Got data %+q, expected %+q", buf[:n], TestBlock)
		}

		_, err = v.Mtime(TestHash)
		if err != nil {
			return err
		}

		err = v.Compare(context.Background(), TestHash, TestBlock)
		if err != nil {
			return err
		}

		indexBuf := new(bytes.Buffer)
		v.IndexTo("", indexBuf)
		if !strings.Contains(string(indexBuf.Bytes()), TestHash) {
			return os.ErrNotExist
		}

		return nil
	}

	// First set: EmptyTrash before reaching the trash deadline.

	s.cluster.Collections.BlobTrashLifetime.Set("1h")

	v.PutRaw(TestHash, TestBlock)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	err := checkGet()
	if err != nil {
		t.Fatal(err)
	}

	// Trash the block
	err = v.Trash(TestHash)
	if err == MethodDisabledError || err == ErrNotImplemented {
		// Skip the trash tests for read-only volumes, and
		// volume types that don't support
		// BlobTrashLifetime>0.
		return
	}

	err = checkGet()
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	err = v.Touch(TestHash)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	v.EmptyTrash()

	// Even after emptying the trash, we can untrash our block
	// because the deadline hasn't been reached.
	err = v.Untrash(TestHash)
	if err != nil {
		t.Fatal(err)
	}

	err = checkGet()
	if err != nil {
		t.Fatal(err)
	}

	err = v.Touch(TestHash)
	if err != nil {
		t.Fatal(err)
	}

	// Because we Touch'ed, need to backdate again for next set of tests
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	// If the only block in the trash has already been untrashed,
	// most volumes will fail a subsequent Untrash with a 404, but
	// it's also acceptable for Untrash to succeed.
	err = v.Untrash(TestHash)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Expected success or os.IsNotExist(), but got: %v", err)
	}

	// The additional Untrash should not interfere with our
	// already-untrashed copy.
	err = checkGet()
	if err != nil {
		t.Fatal(err)
	}

	// Untrash might have updated the timestamp, so backdate again
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	// Second set: EmptyTrash after the trash deadline has passed.

	s.cluster.Collections.BlobTrashLifetime.Set("1ns")

	err = v.Trash(TestHash)
	if err != nil {
		t.Fatal(err)
	}
	err = checkGet()
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	// Even though 1ns has passed, we can untrash because we
	// haven't called EmptyTrash yet.
	err = v.Untrash(TestHash)
	if err != nil {
		t.Fatal(err)
	}
	err = checkGet()
	if err != nil {
		t.Fatal(err)
	}

	// Trash it again, and this time call EmptyTrash so it really
	// goes away.
	// (In Azure volumes, un/trash changes Mtime, so first backdate again)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))
	_ = v.Trash(TestHash)
	err = checkGet()
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}
	v.EmptyTrash()

	// Untrash won't find it
	err = v.Untrash(TestHash)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	// Get block won't find it
	err = checkGet()
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	// Third set: If the same data block gets written again after
	// being trashed, and then the trash gets emptied, the newer
	// un-trashed copy doesn't get deleted along with it.

	v.PutRaw(TestHash, TestBlock)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	s.cluster.Collections.BlobTrashLifetime.Set("1ns")
	err = v.Trash(TestHash)
	if err != nil {
		t.Fatal(err)
	}
	err = checkGet()
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("os.IsNotExist(%v) should have been true", err)
	}

	v.PutRaw(TestHash, TestBlock)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	// EmptyTrash should not delete the untrashed copy.
	v.EmptyTrash()
	err = checkGet()
	if err != nil {
		t.Fatal(err)
	}

	// Fourth set: If the same data block gets trashed twice with
	// different deadlines A and C, and then the trash is emptied
	// at intermediate time B (A < B < C), it is still possible to
	// untrash the block whose deadline is "C".

	v.PutRaw(TestHash, TestBlock)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	s.cluster.Collections.BlobTrashLifetime.Set("1ns")
	err = v.Trash(TestHash)
	if err != nil {
		t.Fatal(err)
	}

	v.PutRaw(TestHash, TestBlock)
	v.TouchWithDate(TestHash, time.Now().Add(-2*s.cluster.Collections.BlobSigningTTL.Duration()))

	s.cluster.Collections.BlobTrashLifetime.Set("1h")
	err = v.Trash(TestHash)
	if err != nil {
		t.Fatal(err)
	}

	// EmptyTrash should not prevent us from recovering the
	// time.Hour ("C") trash
	v.EmptyTrash()
	err = v.Untrash(TestHash)
	if err != nil {
		t.Fatal(err)
	}
	err = checkGet()
	if err != nil {
		t.Fatal(err)
	}
}
