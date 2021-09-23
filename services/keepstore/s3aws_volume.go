// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
	"github.com/aws/aws-sdk-go-v2/aws/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// S3AWSVolume implements Volume using an S3 bucket.
type S3AWSVolume struct {
	arvados.S3VolumeDriverParameters
	AuthToken      string    // populated automatically when IAMRole is used
	AuthExpiration time.Time // populated automatically when IAMRole is used

	cluster   *arvados.Cluster
	volume    arvados.Volume
	logger    logrus.FieldLogger
	metrics   *volumeMetricsVecs
	bucket    *s3AWSbucket
	region    string
	startOnce sync.Once
}

// s3bucket wraps s3.bucket and counts I/O and API usage stats. The
// wrapped bucket can be replaced atomically with SetBucket in order
// to update credentials.
type s3AWSbucket struct {
	bucket string
	svc    *s3.Client
	stats  s3awsbucketStats
	mu     sync.Mutex
}

// chooseS3VolumeDriver distinguishes between the old goamz driver and
// aws-sdk-go based on the UseAWSS3v2Driver feature flag
func chooseS3VolumeDriver(cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) (Volume, error) {
	v := &S3Volume{cluster: cluster, volume: volume, metrics: metrics}
	err := json.Unmarshal(volume.DriverParameters, v)
	if err != nil {
		return nil, err
	}
	if v.UseAWSS3v2Driver {
		logger.Debugln("Using AWS S3 v2 driver")
		return newS3AWSVolume(cluster, volume, logger, metrics)
	}
	logger.Debugln("Using goamz S3 driver")
	return newS3Volume(cluster, volume, logger, metrics)
}

const (
	PartSize         = 5 * 1024 * 1024
	ReadConcurrency  = 13
	WriteConcurrency = 5
)

var s3AWSKeepBlockRegexp = regexp.MustCompile(`^[0-9a-f]{32}$`)
var s3AWSZeroTime time.Time

func (v *S3AWSVolume) isKeepBlock(s string) (string, bool) {
	if v.PrefixLength > 0 && len(s) == v.PrefixLength+33 && s[:v.PrefixLength] == s[v.PrefixLength+1:v.PrefixLength*2+1] {
		s = s[v.PrefixLength+1:]
	}
	return s, s3AWSKeepBlockRegexp.MatchString(s)
}

// Return the key used for a given loc. If PrefixLength==0 then
// key("abcdef0123") is "abcdef0123", if PrefixLength==3 then key is
// "abc/abcdef0123", etc.
func (v *S3AWSVolume) key(loc string) string {
	if v.PrefixLength > 0 && v.PrefixLength < len(loc)-1 {
		return loc[:v.PrefixLength] + "/" + loc
	} else {
		return loc
	}
}

func newS3AWSVolume(cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) (Volume, error) {
	v := &S3AWSVolume{cluster: cluster, volume: volume, metrics: metrics}
	err := json.Unmarshal(volume.DriverParameters, v)
	if err != nil {
		return nil, err
	}
	v.logger = logger.WithField("Volume", v.String())
	return v, v.check("")
}

func (v *S3AWSVolume) translateError(err error) error {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case "NotFound":
			return os.ErrNotExist
		case "NoSuchKey":
			return os.ErrNotExist
		}
	}
	return err
}

// safeCopy calls CopyObjectRequest, and checks the response to make
// sure the copy succeeded and updated the timestamp on the
// destination object
//
// (If something goes wrong during the copy, the error will be
// embedded in the 200 OK response)
func (v *S3AWSVolume) safeCopy(dst, src string) error {
	input := &s3.CopyObjectInput{
		Bucket:      aws.String(v.bucket.bucket),
		ContentType: aws.String("application/octet-stream"),
		CopySource:  aws.String(v.bucket.bucket + "/" + src),
		Key:         aws.String(dst),
	}

	req := v.bucket.svc.CopyObjectRequest(input)
	resp, err := req.Send(context.Background())

	err = v.translateError(err)
	if os.IsNotExist(err) {
		return err
	} else if err != nil {
		return fmt.Errorf("PutCopy(%q ← %q): %s", dst, v.bucket.bucket+"/"+src, err)
	}

	if resp.CopyObjectResult.LastModified == nil {
		return fmt.Errorf("PutCopy succeeded but did not return a timestamp: %q: %s", resp.CopyObjectResult.LastModified, err)
	} else if time.Now().Sub(*resp.CopyObjectResult.LastModified) > maxClockSkew {
		return fmt.Errorf("PutCopy succeeded but returned an old timestamp: %q: %s", resp.CopyObjectResult.LastModified, resp.CopyObjectResult.LastModified)
	}
	return nil
}

func (v *S3AWSVolume) check(ec2metadataHostname string) error {
	if v.Bucket == "" {
		return errors.New("DriverParameters: Bucket must be provided")
	}
	if v.IndexPageSize == 0 {
		v.IndexPageSize = 1000
	}
	if v.RaceWindow < 0 {
		return errors.New("DriverParameters: RaceWindow must not be negative")
	}

	if v.V2Signature {
		return errors.New("DriverParameters: V2Signature is not supported")
	}

	defaultResolver := endpoints.NewDefaultResolver()

	cfg := defaults.Config()

	if v.Endpoint == "" && v.Region == "" {
		return fmt.Errorf("AWS region or endpoint must be specified")
	} else if v.Endpoint != "" || ec2metadataHostname != "" {
		myCustomResolver := func(service, region string) (aws.Endpoint, error) {
			if v.Endpoint != "" && service == "s3" {
				return aws.Endpoint{
					URL:           v.Endpoint,
					SigningRegion: v.Region,
				}, nil
			} else if service == "ec2metadata" && ec2metadataHostname != "" {
				return aws.Endpoint{
					URL: ec2metadataHostname,
				}, nil
			}

			return defaultResolver.ResolveEndpoint(service, region)
		}
		cfg.EndpointResolver = aws.EndpointResolverFunc(myCustomResolver)
	}

	cfg.Region = v.Region

	// Zero timeouts mean "wait forever", which is a bad
	// default. Default to long timeouts instead.
	if v.ConnectTimeout == 0 {
		v.ConnectTimeout = s3DefaultConnectTimeout
	}
	if v.ReadTimeout == 0 {
		v.ReadTimeout = s3DefaultReadTimeout
	}

	creds := aws.NewChainProvider(
		[]aws.CredentialsProvider{
			aws.NewStaticCredentialsProvider(v.AccessKeyID, v.SecretAccessKey, v.AuthToken),
			ec2rolecreds.New(ec2metadata.New(cfg)),
		})

	cfg.Credentials = creds

	v.bucket = &s3AWSbucket{
		bucket: v.Bucket,
		svc:    s3.New(cfg),
	}

	// Set up prometheus metrics
	lbls := prometheus.Labels{"device_id": v.GetDeviceID()}
	v.bucket.stats.opsCounters, v.bucket.stats.errCounters, v.bucket.stats.ioBytes = v.metrics.getCounterVecsFor(lbls)

	return nil
}

// String implements fmt.Stringer.
func (v *S3AWSVolume) String() string {
	return fmt.Sprintf("s3-bucket:%+q", v.Bucket)
}

// GetDeviceID returns a globally unique ID for the storage bucket.
func (v *S3AWSVolume) GetDeviceID() string {
	return "s3://" + v.Endpoint + "/" + v.Bucket
}

// Compare the given data with the stored data.
func (v *S3AWSVolume) Compare(ctx context.Context, loc string, expect []byte) error {
	key := v.key(loc)
	errChan := make(chan error, 1)
	go func() {
		_, err := v.head("recent/" + key)
		errChan <- err
	}()
	var err error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-errChan:
	}
	if err != nil {
		// Checking for the key itself here would interfere
		// with future GET requests.
		//
		// On AWS, if X doesn't exist, a HEAD or GET request
		// for X causes X's non-existence to be cached. Thus,
		// if we test for X, then create X and return a
		// signature to our client, the client might still get
		// 404 from all keepstores when trying to read it.
		//
		// To avoid this, we avoid doing HEAD X or GET X until
		// we know X has been written.
		//
		// Note that X might exist even though recent/X
		// doesn't: for example, the response to HEAD recent/X
		// might itself come from a stale cache. In such
		// cases, we will return a false negative and
		// PutHandler might needlessly create another replica
		// on a different volume. That's not ideal, but it's
		// better than passing the eventually-consistent
		// problem on to our clients.
		return v.translateError(err)
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
	}

	req := v.bucket.svc.GetObjectRequest(input)
	result, err := req.Send(ctx)
	if err != nil {
		return v.translateError(err)
	}
	return v.translateError(compareReaderWithBuf(ctx, result.Body, expect, loc[:32]))
}

// EmptyTrash looks for trashed blocks that exceeded BlobTrashLifetime
// and deletes them from the volume.
func (v *S3AWSVolume) EmptyTrash() {
	if v.cluster.Collections.BlobDeleteConcurrency < 1 {
		return
	}

	var bytesInTrash, blocksInTrash, bytesDeleted, blocksDeleted int64

	// Define "ready to delete" as "...when EmptyTrash started".
	startT := time.Now()

	emptyOneKey := func(trash *s3.Object) {
		key := strings.TrimPrefix(*trash.Key, "trash/")
		loc, isblk := v.isKeepBlock(key)
		if !isblk {
			return
		}
		atomic.AddInt64(&bytesInTrash, *trash.Size)
		atomic.AddInt64(&blocksInTrash, 1)

		trashT := *trash.LastModified
		recent, err := v.head("recent/" + key)
		if err != nil && os.IsNotExist(v.translateError(err)) {
			v.logger.Warnf("EmptyTrash: found trash marker %q but no %q (%s); calling Untrash", *trash.Key, "recent/"+key, err)
			err = v.Untrash(loc)
			if err != nil {
				v.logger.WithError(err).Errorf("EmptyTrash: Untrash(%q) failed", loc)
			}
			return
		} else if err != nil {
			v.logger.WithError(err).Warnf("EmptyTrash: HEAD %q failed", "recent/"+key)
			return
		}
		if trashT.Sub(*recent.LastModified) < v.cluster.Collections.BlobSigningTTL.Duration() {
			if age := startT.Sub(*recent.LastModified); age >= v.cluster.Collections.BlobSigningTTL.Duration()-time.Duration(v.RaceWindow) {
				// recent/key is too old to protect
				// loc from being Trashed again during
				// the raceWindow that starts if we
				// delete trash/X now.
				//
				// Note this means (TrashSweepInterval
				// < BlobSigningTTL - raceWindow) is
				// necessary to avoid starvation.
				v.logger.Infof("EmptyTrash: detected old race for %q, calling fixRace + Touch", loc)
				v.fixRace(key)
				v.Touch(loc)
				return
			}
			_, err := v.head(key)
			if os.IsNotExist(err) {
				v.logger.Infof("EmptyTrash: detected recent race for %q, calling fixRace", loc)
				v.fixRace(key)
				return
			} else if err != nil {
				v.logger.WithError(err).Warnf("EmptyTrash: HEAD %q failed", loc)
				return
			}
		}
		if startT.Sub(trashT) < v.cluster.Collections.BlobTrashLifetime.Duration() {
			return
		}
		err = v.bucket.Del(*trash.Key)
		if err != nil {
			v.logger.WithError(err).Errorf("EmptyTrash: error deleting %q", *trash.Key)
			return
		}
		atomic.AddInt64(&bytesDeleted, *trash.Size)
		atomic.AddInt64(&blocksDeleted, 1)

		_, err = v.head(*trash.Key)
		if err == nil {
			v.logger.Warnf("EmptyTrash: HEAD %q succeeded immediately after deleting %q", loc, loc)
			return
		}
		if !os.IsNotExist(v.translateError(err)) {
			v.logger.WithError(err).Warnf("EmptyTrash: HEAD %q failed", key)
			return
		}
		err = v.bucket.Del("recent/" + key)
		if err != nil {
			v.logger.WithError(err).Warnf("EmptyTrash: error deleting %q", "recent/"+key)
		}
	}

	var wg sync.WaitGroup
	todo := make(chan *s3.Object, v.cluster.Collections.BlobDeleteConcurrency)
	for i := 0; i < v.cluster.Collections.BlobDeleteConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for key := range todo {
				emptyOneKey(key)
			}
		}()
	}

	trashL := s3awsLister{
		Logger:   v.logger,
		Bucket:   v.bucket,
		Prefix:   "trash/",
		PageSize: v.IndexPageSize,
		Stats:    &v.bucket.stats,
	}
	for trash := trashL.First(); trash != nil; trash = trashL.Next() {
		todo <- trash
	}
	close(todo)
	wg.Wait()

	if err := trashL.Error(); err != nil {
		v.logger.WithError(err).Error("EmptyTrash: lister failed")
	}
	v.logger.Infof("EmptyTrash: stats for %v: Deleted %v bytes in %v blocks. Remaining in trash: %v bytes in %v blocks.", v.String(), bytesDeleted, blocksDeleted, bytesInTrash-bytesDeleted, blocksInTrash-blocksDeleted)
}

// fixRace(X) is called when "recent/X" exists but "X" doesn't
// exist. If the timestamps on "recent/X" and "trash/X" indicate there
// was a race between Put and Trash, fixRace recovers from the race by
// Untrashing the block.
func (v *S3AWSVolume) fixRace(key string) bool {
	trash, err := v.head("trash/" + key)
	if err != nil {
		if !os.IsNotExist(v.translateError(err)) {
			v.logger.WithError(err).Errorf("fixRace: HEAD %q failed", "trash/"+key)
		}
		return false
	}

	recent, err := v.head("recent/" + key)
	if err != nil {
		v.logger.WithError(err).Errorf("fixRace: HEAD %q failed", "recent/"+key)
		return false
	}

	recentTime := *recent.LastModified
	trashTime := *trash.LastModified
	ageWhenTrashed := trashTime.Sub(recentTime)
	if ageWhenTrashed >= v.cluster.Collections.BlobSigningTTL.Duration() {
		// No evidence of a race: block hasn't been written
		// since it became eligible for Trash. No fix needed.
		return false
	}

	v.logger.Infof("fixRace: %q: trashed at %s but touched at %s (age when trashed = %s < %s)", key, trashTime, recentTime, ageWhenTrashed, v.cluster.Collections.BlobSigningTTL)
	v.logger.Infof("fixRace: copying %q to %q to recover from race between Put/Touch and Trash", "recent/"+key, key)
	err = v.safeCopy(key, "trash/"+key)
	if err != nil {
		v.logger.WithError(err).Error("fixRace: copy failed")
		return false
	}
	return true
}

func (v *S3AWSVolume) head(key string) (result *s3.HeadObjectOutput, err error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
	}

	req := v.bucket.svc.HeadObjectRequest(input)
	res, err := req.Send(context.TODO())

	v.bucket.stats.TickOps("head")
	v.bucket.stats.Tick(&v.bucket.stats.Ops, &v.bucket.stats.HeadOps)
	v.bucket.stats.TickErr(err)

	if err != nil {
		return nil, v.translateError(err)
	}
	result = res.HeadObjectOutput
	return
}

// Get a block: copy the block data into buf, and return the number of
// bytes copied.
func (v *S3AWSVolume) Get(ctx context.Context, loc string, buf []byte) (int, error) {
	return getWithPipe(ctx, loc, buf, v)
}

func (v *S3AWSVolume) readWorker(ctx context.Context, key string) (rdr io.ReadCloser, err error) {
	buf := make([]byte, 0, 67108864)
	awsBuf := aws.NewWriteAtBuffer(buf)

	downloader := s3manager.NewDownloaderWithClient(v.bucket.svc, func(u *s3manager.Downloader) {
		u.PartSize = PartSize
		u.Concurrency = ReadConcurrency
	})

	v.logger.Debugf("Partsize: %d; Concurrency: %d\n", downloader.PartSize, downloader.Concurrency)

	_, err = downloader.DownloadWithContext(ctx, awsBuf, &s3.GetObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
	})
	v.bucket.stats.TickOps("get")
	v.bucket.stats.Tick(&v.bucket.stats.Ops, &v.bucket.stats.GetOps)
	v.bucket.stats.TickErr(err)
	if err != nil {
		return nil, v.translateError(err)
	}
	buf = awsBuf.Bytes()

	rdr = NewCountingReader(bytes.NewReader(buf), v.bucket.stats.TickInBytes)
	return
}

// ReadBlock implements BlockReader.
func (v *S3AWSVolume) ReadBlock(ctx context.Context, loc string, w io.Writer) error {
	key := v.key(loc)
	rdr, err := v.readWorker(ctx, key)

	if err == nil {
		_, err2 := io.Copy(w, rdr)
		if err2 != nil {
			return err2
		}
		return err
	}

	err = v.translateError(err)
	if !os.IsNotExist(err) {
		return err
	}

	_, err = v.head("recent/" + key)
	err = v.translateError(err)
	if err != nil {
		// If we can't read recent/X, there's no point in
		// trying fixRace. Give up.
		return err
	}
	if !v.fixRace(key) {
		err = os.ErrNotExist
		return err
	}

	rdr, err = v.readWorker(ctx, key)
	if err != nil {
		v.logger.Warnf("reading %s after successful fixRace: %s", loc, err)
		err = v.translateError(err)
		return err
	}

	_, err = io.Copy(w, rdr)

	return err
}

func (v *S3AWSVolume) writeObject(ctx context.Context, key string, r io.Reader) error {
	if r == nil {
		// r == nil leads to a memory violation in func readFillBuf in
		// aws-sdk-go-v2@v0.23.0/service/s3/s3manager/upload.go
		r = bytes.NewReader(nil)
	}

	uploadInput := s3manager.UploadInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
		Body:   r,
	}

	if loc, ok := v.isKeepBlock(key); ok {
		var contentMD5 string
		md5, err := hex.DecodeString(loc)
		if err != nil {
			return err
		}
		contentMD5 = base64.StdEncoding.EncodeToString(md5)
		uploadInput.ContentMD5 = &contentMD5
	}

	// Experimentation indicated that using concurrency 5 yields the best
	// throughput, better than higher concurrency (10 or 13) by ~5%.
	// Defining u.BufferProvider = s3manager.NewBufferedReadSeekerWriteToPool(64 * 1024 * 1024)
	// is detrimental to througput (minus ~15%).
	uploader := s3manager.NewUploaderWithClient(v.bucket.svc, func(u *s3manager.Uploader) {
		u.PartSize = PartSize
		u.Concurrency = WriteConcurrency
	})

	// Unlike the goamz S3 driver, we don't need to precompute ContentSHA256:
	// the aws-sdk-go v2 SDK uses a ReadSeeker to avoid having to copy the
	// block, so there is no extra memory use to be concerned about. See
	// makeSha256Reader in aws/signer/v4/v4.go. In fact, we explicitly disable
	// calculating the Sha-256 because we don't need it; we already use md5sum
	// hashes that match the name of the block.
	_, err := uploader.UploadWithContext(ctx, &uploadInput, s3manager.WithUploaderRequestOptions(func(r *aws.Request) {
		r.HTTPRequest.Header.Set("X-Amz-Content-Sha256", "UNSIGNED-PAYLOAD")
	}))

	v.bucket.stats.TickOps("put")
	v.bucket.stats.Tick(&v.bucket.stats.Ops, &v.bucket.stats.PutOps)
	v.bucket.stats.TickErr(err)

	return err
}

// Put writes a block.
func (v *S3AWSVolume) Put(ctx context.Context, loc string, block []byte) error {
	return putWithPipe(ctx, loc, block, v)
}

// WriteBlock implements BlockWriter.
func (v *S3AWSVolume) WriteBlock(ctx context.Context, loc string, rdr io.Reader) error {
	if v.volume.ReadOnly {
		return MethodDisabledError
	}

	r := NewCountingReader(rdr, v.bucket.stats.TickOutBytes)
	key := v.key(loc)
	err := v.writeObject(ctx, key, r)
	if err != nil {
		return err
	}
	return v.writeObject(ctx, "recent/"+key, nil)
}

type s3awsLister struct {
	Logger            logrus.FieldLogger
	Bucket            *s3AWSbucket
	Prefix            string
	PageSize          int
	Stats             *s3awsbucketStats
	ContinuationToken string
	buf               []s3.Object
	err               error
}

// First fetches the first page and returns the first item. It returns
// nil if the response is the empty set or an error occurs.
func (lister *s3awsLister) First() *s3.Object {
	lister.getPage()
	return lister.pop()
}

// Next returns the next item, fetching the next page if necessary. It
// returns nil if the last available item has already been fetched, or
// an error occurs.
func (lister *s3awsLister) Next() *s3.Object {
	if len(lister.buf) == 0 && lister.ContinuationToken != "" {
		lister.getPage()
	}
	return lister.pop()
}

// Return the most recent error encountered by First or Next.
func (lister *s3awsLister) Error() error {
	return lister.err
}

func (lister *s3awsLister) getPage() {
	lister.Stats.TickOps("list")
	lister.Stats.Tick(&lister.Stats.Ops, &lister.Stats.ListOps)

	var input *s3.ListObjectsV2Input
	if lister.ContinuationToken == "" {
		input = &s3.ListObjectsV2Input{
			Bucket:  aws.String(lister.Bucket.bucket),
			MaxKeys: aws.Int64(int64(lister.PageSize)),
			Prefix:  aws.String(lister.Prefix),
		}
	} else {
		input = &s3.ListObjectsV2Input{
			Bucket:            aws.String(lister.Bucket.bucket),
			MaxKeys:           aws.Int64(int64(lister.PageSize)),
			Prefix:            aws.String(lister.Prefix),
			ContinuationToken: &lister.ContinuationToken,
		}
	}

	req := lister.Bucket.svc.ListObjectsV2Request(input)
	resp, err := req.Send(context.Background())
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			lister.err = aerr
		} else {
			lister.err = err
		}
		return
	}

	if *resp.IsTruncated {
		lister.ContinuationToken = *resp.NextContinuationToken
	} else {
		lister.ContinuationToken = ""
	}
	lister.buf = make([]s3.Object, 0, len(resp.Contents))
	for _, key := range resp.Contents {
		if !strings.HasPrefix(*key.Key, lister.Prefix) {
			lister.Logger.Warnf("s3awsLister: S3 Bucket.List(prefix=%q) returned key %q", lister.Prefix, *key.Key)
			continue
		}
		lister.buf = append(lister.buf, key)
	}
}

func (lister *s3awsLister) pop() (k *s3.Object) {
	if len(lister.buf) > 0 {
		k = &lister.buf[0]
		lister.buf = lister.buf[1:]
	}
	return
}

// IndexTo writes a complete list of locators with the given prefix
// for which Get() can retrieve data.
func (v *S3AWSVolume) IndexTo(prefix string, writer io.Writer) error {
	prefix = v.key(prefix)
	// Use a merge sort to find matching sets of X and recent/X.
	dataL := s3awsLister{
		Logger:   v.logger,
		Bucket:   v.bucket,
		Prefix:   prefix,
		PageSize: v.IndexPageSize,
		Stats:    &v.bucket.stats,
	}
	recentL := s3awsLister{
		Logger:   v.logger,
		Bucket:   v.bucket,
		Prefix:   "recent/" + prefix,
		PageSize: v.IndexPageSize,
		Stats:    &v.bucket.stats,
	}
	for data, recent := dataL.First(), recentL.First(); data != nil && dataL.Error() == nil; data = dataL.Next() {
		if *data.Key >= "g" {
			// Conveniently, "recent/*" and "trash/*" are
			// lexically greater than all hex-encoded data
			// hashes, so stopping here avoids iterating
			// over all of them needlessly with dataL.
			break
		}
		loc, isblk := v.isKeepBlock(*data.Key)
		if !isblk {
			continue
		}

		// stamp is the list entry we should use to report the
		// last-modified time for this data block: it will be
		// the recent/X entry if one exists, otherwise the
		// entry for the data block itself.
		stamp := data

		// Advance to the corresponding recent/X marker, if any
		for recent != nil && recentL.Error() == nil {
			if cmp := strings.Compare((*recent.Key)[7:], *data.Key); cmp < 0 {
				recent = recentL.Next()
				continue
			} else if cmp == 0 {
				stamp = recent
				recent = recentL.Next()
				break
			} else {
				// recent/X marker is missing: we'll
				// use the timestamp on the data
				// object.
				break
			}
		}
		if err := recentL.Error(); err != nil {
			return err
		}
		// We truncate sub-second precision here. Otherwise
		// timestamps will never match the RFC1123-formatted
		// Last-Modified values parsed by Mtime().
		fmt.Fprintf(writer, "%s+%d %d\n", loc, *data.Size, stamp.LastModified.Unix()*1000000000)
	}
	return dataL.Error()
}

// Mtime returns the stored timestamp for the given locator.
func (v *S3AWSVolume) Mtime(loc string) (time.Time, error) {
	key := v.key(loc)
	_, err := v.head(key)
	if err != nil {
		return s3AWSZeroTime, v.translateError(err)
	}
	resp, err := v.head("recent/" + key)
	err = v.translateError(err)
	if os.IsNotExist(err) {
		// The data object X exists, but recent/X is missing.
		err = v.writeObject(context.Background(), "recent/"+key, nil)
		if err != nil {
			v.logger.WithError(err).Errorf("error creating %q", "recent/"+key)
			return s3AWSZeroTime, v.translateError(err)
		}
		v.logger.Infof("Mtime: created %q to migrate existing block to new storage scheme", "recent/"+key)
		resp, err = v.head("recent/" + key)
		if err != nil {
			v.logger.WithError(err).Errorf("HEAD failed after creating %q", "recent/"+key)
			return s3AWSZeroTime, v.translateError(err)
		}
	} else if err != nil {
		// HEAD recent/X failed for some other reason.
		return s3AWSZeroTime, err
	}
	return *resp.LastModified, err
}

// Status returns a *VolumeStatus representing the current in-use
// storage capacity and a fake available capacity that doesn't make
// the volume seem full or nearly-full.
func (v *S3AWSVolume) Status() *VolumeStatus {
	return &VolumeStatus{
		DeviceNum: 1,
		BytesFree: BlockSize * 1000,
		BytesUsed: 1,
	}
}

// InternalStats returns bucket I/O and API call counters.
func (v *S3AWSVolume) InternalStats() interface{} {
	return &v.bucket.stats
}

// Touch sets the timestamp for the given locator to the current time.
func (v *S3AWSVolume) Touch(loc string) error {
	if v.volume.ReadOnly {
		return MethodDisabledError
	}
	key := v.key(loc)
	_, err := v.head(key)
	err = v.translateError(err)
	if os.IsNotExist(err) && v.fixRace(key) {
		// The data object got trashed in a race, but fixRace
		// rescued it.
	} else if err != nil {
		return err
	}
	err = v.writeObject(context.Background(), "recent/"+key, nil)
	return v.translateError(err)
}

// checkRaceWindow returns a non-nil error if trash/key is, or might
// be, in the race window (i.e., it's not safe to trash key).
func (v *S3AWSVolume) checkRaceWindow(key string) error {
	resp, err := v.head("trash/" + key)
	err = v.translateError(err)
	if os.IsNotExist(err) {
		// OK, trash/X doesn't exist so we're not in the race
		// window
		return nil
	} else if err != nil {
		// Error looking up trash/X. We don't know whether
		// we're in the race window
		return err
	}
	t := resp.LastModified
	safeWindow := t.Add(v.cluster.Collections.BlobTrashLifetime.Duration()).Sub(time.Now().Add(time.Duration(v.RaceWindow)))
	if safeWindow <= 0 {
		// We can't count on "touch trash/X" to prolong
		// trash/X's lifetime. The new timestamp might not
		// become visible until now+raceWindow, and EmptyTrash
		// is allowed to delete trash/X before then.
		return fmt.Errorf("%s: same block is already in trash, and safe window ended %s ago", key, -safeWindow)
	}
	// trash/X exists, but it won't be eligible for deletion until
	// after now+raceWindow, so it's safe to overwrite it.
	return nil
}

func (b *s3AWSbucket) Del(path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	}
	req := b.svc.DeleteObjectRequest(input)
	_, err := req.Send(context.Background())
	b.stats.TickOps("delete")
	b.stats.Tick(&b.stats.Ops, &b.stats.DelOps)
	b.stats.TickErr(err)
	return err
}

// Trash a Keep block.
func (v *S3AWSVolume) Trash(loc string) error {
	if v.volume.ReadOnly {
		return MethodDisabledError
	}
	if t, err := v.Mtime(loc); err != nil {
		return err
	} else if time.Since(t) < v.cluster.Collections.BlobSigningTTL.Duration() {
		return nil
	}
	key := v.key(loc)
	if v.cluster.Collections.BlobTrashLifetime == 0 {
		if !v.UnsafeDelete {
			return ErrS3TrashDisabled
		}
		return v.translateError(v.bucket.Del(key))
	}
	err := v.checkRaceWindow(key)
	if err != nil {
		return err
	}
	err = v.safeCopy("trash/"+key, key)
	if err != nil {
		return err
	}
	return v.translateError(v.bucket.Del(key))
}

// Untrash moves block from trash back into store
func (v *S3AWSVolume) Untrash(loc string) error {
	key := v.key(loc)
	err := v.safeCopy(key, "trash/"+key)
	if err != nil {
		return err
	}
	err = v.writeObject(context.Background(), "recent/"+key, nil)
	return v.translateError(err)
}

type s3awsbucketStats struct {
	statsTicker
	Ops     uint64
	GetOps  uint64
	PutOps  uint64
	HeadOps uint64
	DelOps  uint64
	ListOps uint64
}

func (s *s3awsbucketStats) TickErr(err error) {
	if err == nil {
		return
	}
	errType := fmt.Sprintf("%T", err)
	if aerr, ok := err.(awserr.Error); ok {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			// A service error occurred
			errType = errType + fmt.Sprintf(" %d %s", reqErr.StatusCode(), aerr.Code())
		} else {
			errType = errType + fmt.Sprintf(" 000 %s", aerr.Code())
		}
	}
	s.statsTicker.TickErr(err, errType)
}
