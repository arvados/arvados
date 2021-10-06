// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"container/list"
	"context"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

type TrashWorkerTestData struct {
	Locator1    string
	Block1      []byte
	BlockMtime1 int64

	Locator2    string
	Block2      []byte
	BlockMtime2 int64

	CreateData      bool
	CreateInVolume1 bool

	UseTrashLifeTime bool
	DifferentMtimes  bool

	DeleteLocator    string
	SpecifyMountUUID bool

	ExpectLocator1 bool
	ExpectLocator2 bool
}

/* Delete block that does not exist in any of the keep volumes.
   Expect no errors.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_GetNonExistingLocator(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: "5d41402abc4b2a76b9719d911017c592",
		Block1:   []byte("hello"),

		Locator2: "5d41402abc4b2a76b9719d911017c592",
		Block2:   []byte("hello"),

		CreateData: false,

		DeleteLocator: "5d41402abc4b2a76b9719d911017c592",

		ExpectLocator1: false,
		ExpectLocator2: false,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Delete a block that exists on volume 1 of the keep servers.
   Expect the second locator in volume 2 to be unaffected.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_LocatorInVolume1(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash2,
		Block2:   TestBlock2,

		CreateData: true,

		DeleteLocator: TestHash, // first locator

		ExpectLocator1: false,
		ExpectLocator2: true,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Delete a block that exists on volume 2 of the keep servers.
   Expect the first locator in volume 1 to be unaffected.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_LocatorInVolume2(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash2,
		Block2:   TestBlock2,

		CreateData: true,

		DeleteLocator: TestHash2, // locator 2

		ExpectLocator1: true,
		ExpectLocator2: false,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Delete a block with matching mtime for locator in both volumes.
   Expect locator to be deleted from both volumes.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_LocatorInBothVolumes(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash,
		Block2:   TestBlock,

		CreateData: true,

		DeleteLocator: TestHash,

		ExpectLocator1: false,
		ExpectLocator2: false,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Same locator with different Mtimes exists in both volumes.
   Delete the second and expect the first to be still around.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_MtimeMatchesForLocator1ButNotForLocator2(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash,
		Block2:   TestBlock,

		CreateData:      true,
		DifferentMtimes: true,

		DeleteLocator: TestHash,

		ExpectLocator1: true,
		ExpectLocator2: false,
	}
	s.performTrashWorkerTest(c, testData)
}

// Delete a block that exists on both volumes with matching mtimes,
// but specify a MountUUID in the request so it only gets deleted from
// the first volume.
func (s *HandlerSuite) TestTrashWorkerIntegration_SpecifyMountUUID(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash,
		Block2:   TestBlock,

		CreateData: true,

		DeleteLocator:    TestHash,
		SpecifyMountUUID: true,

		ExpectLocator1: true,
		ExpectLocator2: true,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Two different locators in volume 1.
   Delete one of them.
   Expect the other unaffected.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_TwoDifferentLocatorsInVolume1(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash2,
		Block2:   TestBlock2,

		CreateData:      true,
		CreateInVolume1: true,

		DeleteLocator: TestHash, // locator 1

		ExpectLocator1: false,
		ExpectLocator2: true,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Allow default Trash Life time to be used. Thus, the newly created block
   will not be deleted because its Mtime is within the trash life time.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_SameLocatorInTwoVolumesWithDefaultTrashLifeTime(c *check.C) {
	s.cluster.Collections.BlobTrash = true
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash2,
		Block2:   TestBlock2,

		CreateData:      true,
		CreateInVolume1: true,

		UseTrashLifeTime: true,

		DeleteLocator: TestHash, // locator 1

		// Since trash life time is in effect, block won't be deleted.
		ExpectLocator1: true,
		ExpectLocator2: true,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Delete a block with matching mtime for locator in both volumes, but EnableDelete is false,
   so block won't be deleted.
*/
func (s *HandlerSuite) TestTrashWorkerIntegration_DisabledDelete(c *check.C) {
	s.cluster.Collections.BlobTrash = false
	testData := TrashWorkerTestData{
		Locator1: TestHash,
		Block1:   TestBlock,

		Locator2: TestHash,
		Block2:   TestBlock,

		CreateData: true,

		DeleteLocator: TestHash,

		ExpectLocator1: true,
		ExpectLocator2: true,
	}
	s.performTrashWorkerTest(c, testData)
}

/* Perform the test */
func (s *HandlerSuite) performTrashWorkerTest(c *check.C, testData TrashWorkerTestData) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	// Replace the router's trashq -- which the worker goroutines
	// started by setup() are now receiving from -- with a new
	// one, so we can see what the handler sends to it.
	trashq := NewWorkQueue()
	s.handler.Handler.(*router).trashq = trashq

	// Put test content
	mounts := s.handler.volmgr.AllWritable()
	if testData.CreateData {
		mounts[0].Put(context.Background(), testData.Locator1, testData.Block1)
		mounts[0].Put(context.Background(), testData.Locator1+".meta", []byte("metadata"))

		if testData.CreateInVolume1 {
			mounts[0].Put(context.Background(), testData.Locator2, testData.Block2)
			mounts[0].Put(context.Background(), testData.Locator2+".meta", []byte("metadata"))
		} else {
			mounts[1].Put(context.Background(), testData.Locator2, testData.Block2)
			mounts[1].Put(context.Background(), testData.Locator2+".meta", []byte("metadata"))
		}
	}

	oldBlockTime := time.Now().Add(-s.cluster.Collections.BlobSigningTTL.Duration() - time.Minute)

	// Create TrashRequest for the test
	trashRequest := TrashRequest{
		Locator:    testData.DeleteLocator,
		BlockMtime: oldBlockTime.UnixNano(),
	}
	if testData.SpecifyMountUUID {
		trashRequest.MountUUID = s.handler.volmgr.Mounts()[0].UUID
	}

	// Run trash worker and put the trashRequest on trashq
	trashList := list.New()
	trashList.PushBack(trashRequest)

	if !testData.UseTrashLifeTime {
		// Trash worker would not delete block if its Mtime is
		// within trash life time. Back-date the block to
		// allow the deletion to succeed.
		for _, mnt := range mounts {
			mnt.Volume.(*MockVolume).Timestamps[testData.DeleteLocator] = oldBlockTime
			if testData.DifferentMtimes {
				oldBlockTime = oldBlockTime.Add(time.Second)
			}
		}
	}
	go RunTrashWorker(s.handler.volmgr, ctxlog.TestLogger(c), s.cluster, trashq)

	// Install gate so all local operations block until we say go
	gate := make(chan struct{})
	for _, mnt := range mounts {
		mnt.Volume.(*MockVolume).Gate = gate
	}

	assertStatusItem := func(k string, expect float64) {
		if v := getStatusItem(s.handler, "TrashQueue", k); v != expect {
			c.Errorf("Got %s %v, expected %v", k, v, expect)
		}
	}

	assertStatusItem("InProgress", 0)
	assertStatusItem("Queued", 0)

	listLen := trashList.Len()
	trashq.ReplaceQueue(trashList)

	// Wait for worker to take request(s)
	expectEqualWithin(c, time.Second, listLen, func() interface{} { return trashq.Status().InProgress })

	// Ensure status.json also reports work is happening
	assertStatusItem("InProgress", float64(1))
	assertStatusItem("Queued", float64(listLen-1))

	// Let worker proceed
	close(gate)

	// Wait for worker to finish
	expectEqualWithin(c, time.Second, 0, func() interface{} { return trashq.Status().InProgress })

	// Verify Locator1 to be un/deleted as expected
	buf := make([]byte, BlockSize)
	size, err := GetBlock(context.Background(), s.handler.volmgr, testData.Locator1, buf, nil)
	if testData.ExpectLocator1 {
		if size == 0 || err != nil {
			c.Errorf("Expected Locator1 to be still present: %s", testData.Locator1)
		}
	} else {
		if size > 0 || err == nil {
			c.Errorf("Expected Locator1 to be deleted: %s", testData.Locator1)
		}
	}

	// Verify Locator2 to be un/deleted as expected
	if testData.Locator1 != testData.Locator2 {
		size, err = GetBlock(context.Background(), s.handler.volmgr, testData.Locator2, buf, nil)
		if testData.ExpectLocator2 {
			if size == 0 || err != nil {
				c.Errorf("Expected Locator2 to be still present: %s", testData.Locator2)
			}
		} else {
			if size > 0 || err == nil {
				c.Errorf("Expected Locator2 to be deleted: %s", testData.Locator2)
			}
		}
	}

	// The DifferentMtimes test puts the same locator in two
	// different volumes, but only one copy has an Mtime matching
	// the trash request.
	if testData.DifferentMtimes {
		locatorFoundIn := 0
		for _, volume := range s.handler.volmgr.AllReadable() {
			buf := make([]byte, BlockSize)
			if _, err := volume.Get(context.Background(), testData.Locator1, buf); err == nil {
				locatorFoundIn = locatorFoundIn + 1
			}
		}
		c.Check(locatorFoundIn, check.Equals, 1)
	}
}
