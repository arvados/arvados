package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/s3"
	"github.com/AdRoll/goamz/s3/s3test"
	check "gopkg.in/check.v1"
)

type TestableS3Volume struct {
	*S3Volume
	server      *s3test.Server
	c           *check.C
	serverClock *fakeClock
}

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

func NewTestableS3Volume(c *check.C, readonly bool, replication int) *TestableS3Volume {
	clock := &fakeClock{}
	srv, err := s3test.NewServer(&s3test.Config{Clock: clock})
	c.Assert(err, check.IsNil)
	auth := aws.Auth{}
	region := aws.Region{
		Name:                 "test-region-1",
		S3Endpoint:           srv.URL(),
		S3LocationConstraint: true,
	}
	bucket := &s3.Bucket{
		S3:   s3.New(auth, region),
		Name: TestBucketName,
	}
	err = bucket.PutBucket(s3.ACL("private"))
	c.Assert(err, check.IsNil)

	return &TestableS3Volume{
		S3Volume:    NewS3Volume(auth, region, TestBucketName, readonly, replication),
		server:      srv,
		serverClock: clock,
	}
}

var _ = check.Suite(&StubbedS3Suite{})

type StubbedS3Suite struct {
	volumes []*TestableS3Volume
}

func (s *StubbedS3Suite) TestGeneric(c *check.C) {
	DoGenericVolumeTests(c, func(t TB) TestableVolume {
		return NewTestableS3Volume(c, false, 2)
	})
}

func (s *StubbedS3Suite) TestGenericReadOnly(c *check.C) {
	DoGenericVolumeTests(c, func(t TB) TestableVolume {
		return NewTestableS3Volume(c, true, 2)
	})
}

func (s *StubbedS3Suite) TestIndex(c *check.C) {
	v := NewTestableS3Volume(c, false, 2)
	v.indexPageSize = 3
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

// PutRaw skips the ContentMD5 test
func (v *TestableS3Volume) PutRaw(loc string, block []byte) {
	err := v.Bucket.Put(loc, block, "application/octet-stream", s3ACL, s3.Options{})
	if err != nil {
		log.Printf("PutRaw: %+v", err)
	}
}

// TouchWithDate turns back the clock while doing a Touch(). We assume
// there are no other operations happening on the same s3test server
// while we do this.
func (v *TestableS3Volume) TouchWithDate(locator string, lastPut time.Time) {
	v.serverClock.now = &lastPut
	err := v.Touch(locator)
	if err != nil && !strings.Contains(err.Error(), "PutCopy returned old LastModified") {
		log.Printf("Touch: %+v", err)
	}
	v.serverClock.now = nil
}

func (v *TestableS3Volume) Teardown() {
	v.server.Quit()
}
