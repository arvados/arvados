// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

type s3fakeClock struct {
	now *time.Time
}

func (c *s3fakeClock) Now() time.Time {
	if c.now == nil {
		return time.Now().UTC()
	}
	return c.now.UTC()
}

func (c *s3fakeClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

var _ = check.Suite(&stubbedS3Suite{})

var srv httptest.Server

type stubbedS3Suite struct {
	s3server    *httptest.Server
	s3fakeClock *s3fakeClock
	metadata    *httptest.Server
	cluster     *arvados.Cluster
	volumes     []*testableS3Volume
}

func (s *stubbedS3Suite) SetUpTest(c *check.C) {
	s.s3server = nil
	s.s3fakeClock = &s3fakeClock{}
	s.metadata = nil
	s.cluster = testCluster(c)
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Driver: "S3"},
		"zzzzz-nyw5e-111111111111111": {Driver: "S3"},
	}
}

func (s *stubbedS3Suite) TearDownTest(c *check.C) {
	if s.s3server != nil {
		s.s3server.Close()
	}
}

func (s *stubbedS3Suite) TestGeneric(c *check.C) {
	DoGenericVolumeTests(c, false, func(t TB, params newVolumeParams) TestableVolume {
		// Use a negative raceWindow so s3test's 1-second
		// timestamp precision doesn't confuse fixRace.
		return s.newTestableVolume(c, params, -2*time.Second)
	})
}

func (s *stubbedS3Suite) TestGenericReadOnly(c *check.C) {
	DoGenericVolumeTests(c, true, func(t TB, params newVolumeParams) TestableVolume {
		return s.newTestableVolume(c, params, -2*time.Second)
	})
}

func (s *stubbedS3Suite) TestGenericWithPrefix(c *check.C) {
	DoGenericVolumeTests(c, false, func(t TB, params newVolumeParams) TestableVolume {
		v := s.newTestableVolume(c, params, -2*time.Second)
		v.PrefixLength = 3
		return v
	})
}

func (s *stubbedS3Suite) TestIndex(c *check.C) {
	v := s.newTestableVolume(c, newVolumeParams{
		Cluster:      s.cluster,
		ConfigVolume: arvados.Volume{Replication: 2},
		MetricsVecs:  newVolumeMetricsVecs(prometheus.NewRegistry()),
		BufferPool:   newBufferPool(ctxlog.TestLogger(c), 8, prometheus.NewRegistry()),
	}, 0)
	v.IndexPageSize = 3
	for i := 0; i < 256; i++ {
		err := v.blockWriteWithoutMD5Check(fmt.Sprintf("%02x%030x", i, i), []byte{102, 111, 111})
		c.Assert(err, check.IsNil)
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
		err := v.Index(context.Background(), spec.prefix, buf)
		c.Check(err, check.IsNil)

		idx := bytes.SplitAfter(buf.Bytes(), []byte{10})
		c.Check(len(idx), check.Equals, spec.expectMatch+1)
		c.Check(len(idx[len(idx)-1]), check.Equals, 0)
	}
}

func (s *stubbedS3Suite) TestSignature(c *check.C) {
	var header http.Header
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header = r.Header
	}))
	defer stub.Close()

	// The aws-sdk-go-v2 driver only supports S3 V4 signatures. S3 v2 signatures are being phased out
	// as of June 24, 2020. Cf. https://forums.aws.amazon.com/ann.jspa?annID=5816
	vol := s3Volume{
		S3VolumeDriverParameters: arvados.S3VolumeDriverParameters{
			AccessKeyID:     "xxx",
			SecretAccessKey: "xxx",
			Endpoint:        stub.URL,
			Region:          "test-region-1",
			Bucket:          "test-bucket-name",
			UsePathStyle:    true,
		},
		cluster: s.cluster,
		logger:  ctxlog.TestLogger(c),
		metrics: newVolumeMetricsVecs(prometheus.NewRegistry()),
	}
	err := vol.check("")

	c.Check(err, check.IsNil)
	err = vol.BlockWrite(context.Background(), "acbd18db4cc2f85cedef654fccc4a4d8", []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(header.Get("Authorization"), check.Matches, `AWS4-HMAC-SHA256 .*`)
}

func (s *stubbedS3Suite) TestIAMRoleCredentials(c *check.C) {
	var reqHeader http.Header
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqHeader = r.Header
	}))
	defer stub.Close()

	retrievedMetadata := false
	s.metadata = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retrievedMetadata = true
		upd := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
		exp := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
		c.Logf("metadata stub received request: %s %s", r.Method, r.URL.Path)
		switch {
		case r.URL.Path == "/latest/meta-data/iam/security-credentials/":
			io.WriteString(w, "testcredential\n")
		case r.URL.Path == "/latest/api/token",
			r.URL.Path == "/latest/meta-data/iam/security-credentials/testcredential":
			// Literal example from
			// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#instance-metadata-security-credentials
			// but with updated timestamps
			io.WriteString(w, `{"Code":"Success","LastUpdated":"`+upd+`","Type":"AWS-HMAC","AccessKeyId":"ASIAIOSFODNN7EXAMPLE","SecretAccessKey":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY","Token":"token","Expiration":"`+exp+`"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.metadata.Close()

	v := &s3Volume{
		S3VolumeDriverParameters: arvados.S3VolumeDriverParameters{
			Endpoint: stub.URL,
			Region:   "test-region-1",
			Bucket:   "test-bucket-name",
		},
		cluster: s.cluster,
		logger:  ctxlog.TestLogger(c),
		metrics: newVolumeMetricsVecs(prometheus.NewRegistry()),
	}
	err := v.check(s.metadata.URL + "/latest")
	c.Check(err, check.IsNil)
	resp, err := v.bucket.svc.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	c.Check(err, check.IsNil)
	c.Check(resp.Buckets, check.HasLen, 0)
	c.Check(retrievedMetadata, check.Equals, true)
	c.Check(reqHeader.Get("Authorization"), check.Matches, `AWS4-HMAC-SHA256 Credential=ASIAIOSFODNN7EXAMPLE/\d+/test-region-1/s3/aws4_request, SignedHeaders=.*`)

	retrievedMetadata = false
	s.metadata = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retrievedMetadata = true
		c.Logf("metadata stub received request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	deadv := &s3Volume{
		S3VolumeDriverParameters: arvados.S3VolumeDriverParameters{
			Endpoint: "http://localhost:9",
			Region:   "test-region-1",
			Bucket:   "test-bucket-name",
		},
		cluster: s.cluster,
		logger:  ctxlog.TestLogger(c),
		metrics: newVolumeMetricsVecs(prometheus.NewRegistry()),
	}
	err = deadv.check(s.metadata.URL + "/latest")
	c.Check(err, check.IsNil)
	_, err = deadv.bucket.svc.ListBuckets(context.Background(), &s3.ListBucketsInput{})
	c.Check(err, check.ErrorMatches, `(?s).*failed to refresh cached credentials, no EC2 IMDS role found.*`)
	c.Check(err, check.ErrorMatches, `(?s).*404.*`)
	c.Check(retrievedMetadata, check.Equals, true)
}

func (s *stubbedS3Suite) TestStats(c *check.C) {
	v := s.newTestableVolume(c, newVolumeParams{
		Cluster:      s.cluster,
		ConfigVolume: arvados.Volume{Replication: 2},
		MetricsVecs:  newVolumeMetricsVecs(prometheus.NewRegistry()),
		BufferPool:   newBufferPool(ctxlog.TestLogger(c), 8, prometheus.NewRegistry()),
	}, 5*time.Minute)
	stats := func() string {
		buf, err := json.Marshal(v.InternalStats())
		c.Check(err, check.IsNil)
		return string(buf)
	}

	c.Check(stats(), check.Matches, `.*"Ops":0,.*`)

	loc := "acbd18db4cc2f85cedef654fccc4a4d8"
	err := v.BlockRead(context.Background(), loc, brdiscard)
	c.Check(err, check.NotNil)
	c.Check(stats(), check.Matches, `.*"Ops":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"\*smithy.OperationError 404 NoSuchKey":[^0].*`)
	c.Check(stats(), check.Matches, `.*"InBytes":0,.*`)

	err = v.BlockWrite(context.Background(), loc, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"OutBytes":3,.*`)
	c.Check(stats(), check.Matches, `.*"PutOps":2,.*`)

	err = v.BlockRead(context.Background(), loc, brdiscard)
	c.Check(err, check.IsNil)
	err = v.BlockRead(context.Background(), loc, brdiscard)
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"InBytes":6,.*`)
}

type s3AWSBlockingHandler struct {
	requested chan *http.Request
	unblock   chan struct{}
}

func (h *s3AWSBlockingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" && !strings.Contains(strings.Trim(r.URL.Path, "/"), "/") {
		// Accept PutBucket ("PUT /bucketname/"), called by
		// newTestableVolume
		return
	}
	if h.requested != nil {
		h.requested <- r
	}
	if h.unblock != nil {
		<-h.unblock
	}
	http.Error(w, "nothing here", http.StatusNotFound)
}

func (s *stubbedS3Suite) TestGetContextCancel(c *check.C) {
	s.testContextCancel(c, func(ctx context.Context, v *testableS3Volume) error {
		return v.BlockRead(ctx, fooHash, brdiscard)
	})
}

func (s *stubbedS3Suite) TestPutContextCancel(c *check.C) {
	s.testContextCancel(c, func(ctx context.Context, v *testableS3Volume) error {
		return v.BlockWrite(ctx, fooHash, []byte("foo"))
	})
}

func (s *stubbedS3Suite) testContextCancel(c *check.C, testFunc func(context.Context, *testableS3Volume) error) {
	handler := &s3AWSBlockingHandler{}
	s.s3server = httptest.NewServer(handler)
	defer s.s3server.Close()

	v := s.newTestableVolume(c, newVolumeParams{
		Cluster:      s.cluster,
		ConfigVolume: arvados.Volume{Replication: 2},
		MetricsVecs:  newVolumeMetricsVecs(prometheus.NewRegistry()),
		BufferPool:   newBufferPool(ctxlog.TestLogger(c), 8, prometheus.NewRegistry()),
	}, 5*time.Minute)

	ctx, cancel := context.WithCancel(context.Background())

	handler.requested = make(chan *http.Request)
	handler.unblock = make(chan struct{})
	defer close(handler.unblock)

	doneFunc := make(chan struct{})
	go func() {
		err := testFunc(ctx, v)
		c.Check(err, check.Equals, context.Canceled)
		close(doneFunc)
	}()

	timeout := time.After(10 * time.Second)

	// Wait for the stub server to receive a request, meaning
	// Get() is waiting for an s3 operation.
	select {
	case <-timeout:
		c.Fatal("timed out waiting for test func to call our handler")
	case <-doneFunc:
		c.Fatal("test func finished without even calling our handler!")
	case <-handler.requested:
	}

	cancel()

	select {
	case <-timeout:
		c.Fatal("timed out")
	case <-doneFunc:
	}
}

func (s *stubbedS3Suite) TestBackendStates(c *check.C) {
	s.cluster.Collections.BlobTrashLifetime.Set("1h")
	s.cluster.Collections.BlobSigningTTL.Set("1h")

	v := s.newTestableVolume(c, newVolumeParams{
		Cluster:      s.cluster,
		ConfigVolume: arvados.Volume{Replication: 2},
		Logger:       ctxlog.TestLogger(c),
		MetricsVecs:  newVolumeMetricsVecs(prometheus.NewRegistry()),
		BufferPool:   newBufferPool(ctxlog.TestLogger(c), 8, prometheus.NewRegistry()),
	}, 5*time.Minute)
	var none time.Time

	putS3Obj := func(t time.Time, key string, data []byte) {
		if t == none {
			return
		}
		s.s3fakeClock.now = &t
		uploader := manager.NewUploader(v.bucket.svc)
		_, err := uploader.Upload(context.Background(), &s3.PutObjectInput{
			Bucket: aws.String(v.bucket.bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(data),
		})
		if err != nil {
			panic(err)
		}
		s.s3fakeClock.now = nil
		_, err = v.head(key)
		if err != nil {
			panic(err)
		}
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
			"Erroneously trashed during a race, detected before BlobTrashLifetime",
			none, t0.Add(-30 * time.Minute), t0.Add(-29 * time.Minute),
			true, false, true, true, true, false,
		},
		{
			"Erroneously trashed during a race, rescue during EmptyTrash despite reaching BlobTrashLifetime",
			none, t0.Add(-90 * time.Minute), t0.Add(-89 * time.Minute),
			true, false, true, true, true, false,
		},
		{
			"Trashed copy exists with no recent/* marker (cause unknown); repair by untrashing",
			none, none, t0.Add(-time.Minute),
			false, false, false, true, true, true,
		},
	} {
		for _, prefixLength := range []int{0, 3} {
			v.PrefixLength = prefixLength
			c.Logf("Scenario: %q (prefixLength=%d)", scenario.label, prefixLength)

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
				key := loc
				if prefixLength > 0 {
					key = loc[:prefixLength] + "/" + loc
				}
				c.Log("\t", loc, "\t", key)
				putS3Obj(scenario.dataT, key, blk)
				putS3Obj(scenario.recentT, "recent/"+key, nil)
				putS3Obj(scenario.trashT, "trash/"+key, blk)
				v.s3fakeClock.now = &t0
				return loc, blk
			}

			// Check canGet
			loc, blk := setupScenario()
			err := v.BlockRead(context.Background(), loc, brdiscard)
			c.Check(err == nil, check.Equals, scenario.canGet, check.Commentf("err was %+v", err))
			if err != nil {
				c.Check(os.IsNotExist(err), check.Equals, true)
			}

			// Call Trash, then check canTrash and canGetAfterTrash
			loc, _ = setupScenario()
			err = v.BlockTrash(loc)
			c.Check(err == nil, check.Equals, scenario.canTrash)
			err = v.BlockRead(context.Background(), loc, brdiscard)
			c.Check(err == nil, check.Equals, scenario.canGetAfterTrash)
			if err != nil {
				c.Check(os.IsNotExist(err), check.Equals, true)
			}

			// Call Untrash, then check canUntrash
			loc, _ = setupScenario()
			err = v.BlockUntrash(loc)
			c.Check(err == nil, check.Equals, scenario.canUntrash)
			if scenario.dataT != none || scenario.trashT != none {
				// In all scenarios where the data exists, we
				// should be able to Get after Untrash --
				// regardless of timestamps, errors, race
				// conditions, etc.
				err = v.BlockRead(context.Background(), loc, brdiscard)
				c.Check(err, check.IsNil)
			}

			// Call EmptyTrash, then check haveTrashAfterEmpty and
			// freshAfterEmpty
			loc, _ = setupScenario()
			v.EmptyTrash()
			_, err = v.head("trash/" + v.key(loc))
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
			err = v.BlockWrite(context.Background(), loc, blk)
			c.Check(err, check.IsNil)
			t, err := v.Mtime(loc)
			c.Check(err, check.IsNil)
			c.Check(t.After(t0.Add(-time.Second)), check.Equals, true)
		}
	}
}

type testableS3Volume struct {
	*s3Volume
	server      *httptest.Server
	c           *check.C
	s3fakeClock *s3fakeClock
}

type gofakes3logger struct {
	logrus.FieldLogger
}

func (l gofakes3logger) Print(level gofakes3.LogLevel, v ...interface{}) {
	switch level {
	case gofakes3.LogErr:
		l.Errorln(v...)
	case gofakes3.LogWarn:
		l.Warnln(v...)
	case gofakes3.LogInfo:
		l.Infoln(v...)
	default:
		panic("unknown level")
	}
}

var testBucketSerial atomic.Int64

func (s *stubbedS3Suite) newTestableVolume(c *check.C, params newVolumeParams, raceWindow time.Duration) *testableS3Volume {
	if params.Logger == nil {
		params.Logger = ctxlog.TestLogger(c)
	}
	if s.s3server == nil {
		backend := s3mem.New(s3mem.WithTimeSource(s.s3fakeClock))
		logger := ctxlog.TestLogger(c)
		faker := gofakes3.New(backend,
			gofakes3.WithTimeSource(s.s3fakeClock),
			gofakes3.WithLogger(gofakes3logger{FieldLogger: logger}),
			gofakes3.WithTimeSkewLimit(0))
		s.s3server = httptest.NewServer(faker.Server())
	}
	endpoint := s.s3server.URL
	bucketName := fmt.Sprintf("testbucket%d", testBucketSerial.Add(1))

	var metadataURL, accessKey, secretKey string
	if s.metadata != nil {
		metadataURL = s.metadata.URL
	} else {
		accessKey, secretKey = "xxx", "xxx"
	}

	v := &testableS3Volume{
		s3Volume: &s3Volume{
			S3VolumeDriverParameters: arvados.S3VolumeDriverParameters{
				AccessKeyID:        accessKey,
				SecretAccessKey:    secretKey,
				Bucket:             bucketName,
				Endpoint:           endpoint,
				Region:             "test-region-1",
				LocationConstraint: true,
				UnsafeDelete:       true,
				IndexPageSize:      1000,
				UsePathStyle:       true,
			},
			cluster:    params.Cluster,
			volume:     params.ConfigVolume,
			logger:     params.Logger,
			metrics:    params.MetricsVecs,
			bufferPool: params.BufferPool,
		},
		c:           c,
		s3fakeClock: s.s3fakeClock,
	}
	c.Assert(v.s3Volume.check(metadataURL), check.IsNil)
	// Create the testbucket
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	_, err := v.s3Volume.bucket.svc.CreateBucket(context.Background(), input)
	c.Assert(err, check.IsNil)
	// We couldn't set RaceWindow until now because check()
	// rejects negative values.
	v.s3Volume.RaceWindow = arvados.Duration(raceWindow)
	return v
}

func (v *testableS3Volume) blockWriteWithoutMD5Check(loc string, block []byte) error {
	key := v.key(loc)
	r := newCountingReader(bytes.NewReader(block), v.bucket.stats.TickOutBytes)

	uploader := manager.NewUploader(v.bucket.svc, func(u *manager.Uploader) {
		u.PartSize = 5 * 1024 * 1024
		u.Concurrency = 13
	})

	_, err := uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return err
	}

	empty := bytes.NewReader([]byte{})
	_, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String("recent/" + key),
		Body:   empty,
	})
	return err
}

// TouchWithDate turns back the clock while doing a Touch(). We assume
// there are no other operations happening on the same s3test server
// while we do this.
func (v *testableS3Volume) TouchWithDate(loc string, lastPut time.Time) {
	v.s3fakeClock.now = &lastPut

	uploader := manager.NewUploader(v.bucket.svc)
	empty := bytes.NewReader([]byte{})
	_, err := uploader.Upload(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String("recent/" + v.key(loc)),
		Body:   empty,
	})
	if err != nil {
		panic(err)
	}

	v.s3fakeClock.now = nil
}

func (v *testableS3Volume) Teardown() {
}

func (v *testableS3Volume) ReadWriteOperationLabelValues() (r, w string) {
	return "get", "put"
}
