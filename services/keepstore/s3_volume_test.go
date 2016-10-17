package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/AdRoll/goamz/s3"
	"github.com/AdRoll/goamz/s3/s3test"
	check "gopkg.in/check.v1"
)

const (
	TestBucketName = "testbucket"
)

type fakeClock struct {
	now *time.Time
}

func (c *fakeClock) Now() time.Time {
	if c.now == nil {
		return time.Now()
	}
	return *c.now
}

func init() {
	// Deleting isn't safe from races, but if it's turned on
	// anyway we do expect it to pass the generic volume tests.
	s3UnsafeDelete = true
}

var _ = check.Suite(&StubbedS3Suite{})

type StubbedS3Suite struct {
	volumes []*TestableS3Volume
}

func (s *StubbedS3Suite) TestGeneric(c *check.C) {
	DoGenericVolumeTests(c, func(t TB) TestableVolume {
		// Use a negative raceWindow so s3test's 1-second
		// timestamp precision doesn't confuse fixRace.
		return s.newTestableVolume(c, -2*time.Second, false, 2)
	})
}

func (s *StubbedS3Suite) TestGenericReadOnly(c *check.C) {
	DoGenericVolumeTests(c, func(t TB) TestableVolume {
		return s.newTestableVolume(c, -2*time.Second, true, 2)
	})
}

func (s *StubbedS3Suite) TestIndex(c *check.C) {
	v := s.newTestableVolume(c, 0, false, 2)
	v.IndexPageSize = 3
	for i := 0; i < 256; i++ {
		v.PutRaw(fmt.Sprintf("%02x%030x", i, i), []byte{102, 111, 111})
	}
	for _, spec := range []struct {
		prefix      string
		expectMatch int
	}{
		{"", 256},
		{"c", 16},
		{"bc", 1},
		{"abc", 0},
	} {
		buf := new(bytes.Buffer)
		err := v.IndexTo(spec.prefix, buf)
		c.Check(err, check.IsNil)

		idx := bytes.SplitAfter(buf.Bytes(), []byte{10})
		c.Check(len(idx), check.Equals, spec.expectMatch+1)
		c.Check(len(idx[len(idx)-1]), check.Equals, 0)
	}
}

func (s *StubbedS3Suite) TestBackendStates(c *check.C) {
	defer func(tl, bs arvados.Duration) {
		theConfig.TrashLifetime = tl
		theConfig.BlobSignatureTTL = bs
	}(theConfig.TrashLifetime, theConfig.BlobSignatureTTL)
	theConfig.TrashLifetime.Set("1h")
	theConfig.BlobSignatureTTL.Set("1h")

	v := s.newTestableVolume(c, 5*time.Minute, false, 2)
	var none time.Time

	putS3Obj := func(t time.Time, key string, data []byte) {
		if t == none {
			return
		}
		v.serverClock.now = &t
		v.bucket.Put(key, data, "application/octet-stream", s3ACL, s3.Options{})
	}

	t0 := time.Now()
	nextKey := 0
	for _, scenario := range []struct {
		label               string
		dataT               time.Time
		recentT             time.Time
		trashT              time.Time
		canGet              bool
		canTrash            bool
		canGetAfterTrash    bool
		canUntrash          bool
		haveTrashAfterEmpty bool
		freshAfterEmpty     bool
	}{
		{
			"No related objects",
			none, none, none,
			false, false, false, false, false, false,
		},
		{
			// Stored by older version, or there was a
			// race between EmptyTrash and Put: Trash is a
			// no-op even though the data object is very
			// old
			"No recent/X",
			t0.Add(-48 * time.Hour), none, none,
			true, true, true, false, false, false,
		},
		{
			"Not trash, but old enough to be eligible for trash",
			t0.Add(-24 * time.Hour), t0.Add(-2 * time.Hour), none,
			true, true, false, false, false, false,
		},
		{
			"Not trash, and not old enough to be eligible for trash",
			t0.Add(-24 * time.Hour), t0.Add(-30 * time.Minute), none,
			true, true, true, false, false, false,
		},
		{
			"Trashed + untrashed copies exist, due to recent race between Trash and Put",
			t0.Add(-24 * time.Hour), t0.Add(-3 * time.Minute), t0.Add(-2 * time.Minute),
			true, true, true, true, true, false,
		},
		{
			"Trashed + untrashed copies exist, trash nearly eligible for deletion: prone to Trash race",
			t0.Add(-24 * time.Hour), t0.Add(-12 * time.Hour), t0.Add(-59 * time.Minute),
			true, false, true, true, true, false,
		},
		{
			"Trashed + untrashed copies exist, trash is eligible for deletion: prone to Trash race",
			t0.Add(-24 * time.Hour), t0.Add(-12 * time.Hour), t0.Add(-61 * time.Minute),
			true, false, true, true, false, false,
		},
		{
			"Trashed + untrashed copies exist, due to old race between Put and unfinished Trash: emptying trash is unsafe",
			t0.Add(-24 * time.Hour), t0.Add(-12 * time.Hour), t0.Add(-12 * time.Hour),
			true, false, true, true, true, true,
		},
		{
			"Trashed + untrashed copies exist, used to be unsafe to empty, but since made safe by fixRace+Touch",
			t0.Add(-time.Second), t0.Add(-time.Second), t0.Add(-12 * time.Hour),
			true, true, true, true, false, false,
		},
		{
			"Trashed + untrashed copies exist because Trash operation was interrupted (no race)",
			t0.Add(-24 * time.Hour), t0.Add(-24 * time.Hour), t0.Add(-12 * time.Hour),
			true, false, true, true, false, false,
		},
		{
			"Trash, not yet eligible for deletion",
			none, t0.Add(-12 * time.Hour), t0.Add(-time.Minute),
			false, false, false, true, true, false,
		},
		{
			"Trash, not yet eligible for deletion, prone to races",
			none, t0.Add(-12 * time.Hour), t0.Add(-59 * time.Minute),
			false, false, false, true, true, false,
		},
		{
			"Trash, eligible for deletion",
			none, t0.Add(-12 * time.Hour), t0.Add(-2 * time.Hour),
			false, false, false, true, false, false,
		},
		{
			"Erroneously trashed during a race, detected before TrashLifetime",
			none, t0.Add(-30 * time.Minute), t0.Add(-29 * time.Minute),
			true, false, true, true, true, false,
		},
		{
			"Erroneously trashed during a race, rescue during EmptyTrash despite reaching TrashLifetime",
			none, t0.Add(-90 * time.Minute), t0.Add(-89 * time.Minute),
			true, false, true, true, true, false,
		},
		{
			"Trashed copy exists with no recent/* marker (cause unknown); repair by untrashing",
			none, none, t0.Add(-time.Minute),
			false, false, false, true, true, true,
		},
	} {
		c.Log("Scenario: ", scenario.label)

		// We have a few tests to run for each scenario, and
		// the tests are expected to change state. By calling
		// this setup func between tests, we (re)create the
		// scenario as specified, using a new unique block
		// locator to prevent interference from previous
		// tests.

		setupScenario := func() (string, []byte) {
			nextKey++
			blk := []byte(fmt.Sprintf("%d", nextKey))
			loc := fmt.Sprintf("%x", md5.Sum(blk))
			c.Log("\t", loc)
			putS3Obj(scenario.dataT, loc, blk)
			putS3Obj(scenario.recentT, "recent/"+loc, nil)
			putS3Obj(scenario.trashT, "trash/"+loc, blk)
			v.serverClock.now = &t0
			return loc, blk
		}

		// Check canGet
		loc, blk := setupScenario()
		buf := make([]byte, len(blk))
		_, err := v.Get(loc, buf)
		c.Check(err == nil, check.Equals, scenario.canGet)
		if err != nil {
			c.Check(os.IsNotExist(err), check.Equals, true)
		}

		// Call Trash, then check canTrash and canGetAfterTrash
		loc, blk = setupScenario()
		err = v.Trash(loc)
		c.Check(err == nil, check.Equals, scenario.canTrash)
		_, err = v.Get(loc, buf)
		c.Check(err == nil, check.Equals, scenario.canGetAfterTrash)
		if err != nil {
			c.Check(os.IsNotExist(err), check.Equals, true)
		}

		// Call Untrash, then check canUntrash
		loc, blk = setupScenario()
		err = v.Untrash(loc)
		c.Check(err == nil, check.Equals, scenario.canUntrash)
		if scenario.dataT != none || scenario.trashT != none {
			// In all scenarios where the data exists, we
			// should be able to Get after Untrash --
			// regardless of timestamps, errors, race
			// conditions, etc.
			_, err = v.Get(loc, buf)
			c.Check(err, check.IsNil)
		}

		// Call EmptyTrash, then check haveTrashAfterEmpty and
		// freshAfterEmpty
		loc, blk = setupScenario()
		v.EmptyTrash()
		_, err = v.bucket.Head("trash/"+loc, nil)
		c.Check(err == nil, check.Equals, scenario.haveTrashAfterEmpty)
		if scenario.freshAfterEmpty {
			t, err := v.Mtime(loc)
			c.Check(err, check.IsNil)
			// new mtime must be current (with an
			// allowance for 1s timestamp precision)
			c.Check(t.After(t0.Add(-time.Second)), check.Equals, true)
		}

		// Check for current Mtime after Put (applies to all
		// scenarios)
		loc, blk = setupScenario()
		err = v.Put(loc, blk)
		c.Check(err, check.IsNil)
		t, err := v.Mtime(loc)
		c.Check(err, check.IsNil)
		c.Check(t.After(t0.Add(-time.Second)), check.Equals, true)
	}
}

type TestableS3Volume struct {
	*S3Volume
	server      *s3test.Server
	c           *check.C
	serverClock *fakeClock
}

func (s *StubbedS3Suite) newTestableVolume(c *check.C, raceWindow time.Duration, readonly bool, replication int) *TestableS3Volume {
	clock := &fakeClock{}
	srv, err := s3test.NewServer(&s3test.Config{Clock: clock})
	c.Assert(err, check.IsNil)

	tmp, err := ioutil.TempFile("", "keepstore")
	c.Assert(err, check.IsNil)
	defer os.Remove(tmp.Name())
	_, err = tmp.Write([]byte("xxx\n"))
	c.Assert(err, check.IsNil)
	c.Assert(tmp.Close(), check.IsNil)

	v := &TestableS3Volume{
		S3Volume: &S3Volume{
			Bucket:             TestBucketName,
			AccessKeyFile:      tmp.Name(),
			SecretKeyFile:      tmp.Name(),
			Endpoint:           srv.URL(),
			Region:             "test-region-1",
			LocationConstraint: true,
			RaceWindow:         arvados.Duration(raceWindow),
			S3Replication:      replication,
			UnsafeDelete:       s3UnsafeDelete,
			ReadOnly:           readonly,
			IndexPageSize:      1000,
		},
		server:      srv,
		serverClock: clock,
	}
	c.Assert(v.Start(), check.IsNil)
	err = v.bucket.PutBucket(s3.ACL("private"))
	c.Assert(err, check.IsNil)
	return v
}

// PutRaw skips the ContentMD5 test
func (v *TestableS3Volume) PutRaw(loc string, block []byte) {
	err := v.bucket.Put(loc, block, "application/octet-stream", s3ACL, s3.Options{})
	if err != nil {
		log.Printf("PutRaw: %+v", err)
	}
}

// TouchWithDate turns back the clock while doing a Touch(). We assume
// there are no other operations happening on the same s3test server
// while we do this.
func (v *TestableS3Volume) TouchWithDate(locator string, lastPut time.Time) {
	v.serverClock.now = &lastPut
	err := v.bucket.Put("recent/"+locator, nil, "application/octet-stream", s3ACL, s3.Options{})
	if err != nil {
		panic(err)
	}
	v.serverClock.now = nil
}

func (v *TestableS3Volume) Teardown() {
	v.server.Quit()
}
