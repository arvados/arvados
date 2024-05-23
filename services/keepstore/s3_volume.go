// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func init() {
	driver["S3"] = news3Volume
}

const (
	s3DefaultReadTimeout        = arvados.Duration(10 * time.Minute)
	s3DefaultConnectTimeout     = arvados.Duration(time.Minute)
	maxClockSkew                = 600 * time.Second
	nearlyRFC1123               = "Mon, 2 Jan 2006 15:04:05 GMT"
	s3downloaderPartSize        = 6 * 1024 * 1024
	s3downloaderReadConcurrency = 11
	s3uploaderPartSize          = 5 * 1024 * 1024
	s3uploaderWriteConcurrency  = 5
)

var (
	errS3TrashDisabled        = fmt.Errorf("trash function is disabled because Collections.BlobTrashLifetime=0 and DriverParameters.UnsafeDelete=false")
	s3AWSKeepBlockRegexp      = regexp.MustCompile(`^[0-9a-f]{32}$`)
	s3AWSZeroTime             time.Time
	defaultEndpointResolverV2 = s3.NewDefaultEndpointResolverV2()

	// Returned by an aws.EndpointResolverWithOptions to indicate
	// that the default resolver should be used.
	errEndpointNotOverridden = &aws.EndpointNotFoundError{Err: errors.New("endpoint not overridden")}
)

// s3Volume implements Volume using an S3 bucket.
type s3Volume struct {
	arvados.S3VolumeDriverParameters
	AuthToken      string    // populated automatically when IAMRole is used
	AuthExpiration time.Time // populated automatically when IAMRole is used

	cluster    *arvados.Cluster
	volume     arvados.Volume
	logger     logrus.FieldLogger
	metrics    *volumeMetricsVecs
	bufferPool *bufferPool
	bucket     *s3Bucket
	region     string
	startOnce  sync.Once

	overrideEndpoint *aws.Endpoint
	usePathStyle     bool // used by test suite
}

// s3bucket wraps s3.bucket and counts I/O and API usage stats. The
// wrapped bucket can be replaced atomically with SetBucket in order
// to update credentials.
type s3Bucket struct {
	bucket string
	svc    *s3.Client
	stats  s3awsbucketStats
	mu     sync.Mutex
}

func (v *s3Volume) isKeepBlock(s string) (string, bool) {
	if v.PrefixLength > 0 && len(s) == v.PrefixLength+33 && s[:v.PrefixLength] == s[v.PrefixLength+1:v.PrefixLength*2+1] {
		s = s[v.PrefixLength+1:]
	}
	return s, s3AWSKeepBlockRegexp.MatchString(s)
}

// Return the key used for a given loc. If PrefixLength==0 then
// key("abcdef0123") is "abcdef0123", if PrefixLength==3 then key is
// "abc/abcdef0123", etc.
func (v *s3Volume) key(loc string) string {
	if v.PrefixLength > 0 && v.PrefixLength < len(loc)-1 {
		return loc[:v.PrefixLength] + "/" + loc
	} else {
		return loc
	}
}

func news3Volume(params newVolumeParams) (volume, error) {
	v := &s3Volume{
		cluster:    params.Cluster,
		volume:     params.ConfigVolume,
		metrics:    params.MetricsVecs,
		bufferPool: params.BufferPool,
	}
	err := json.Unmarshal(params.ConfigVolume.DriverParameters, v)
	if err != nil {
		return nil, err
	}
	v.logger = params.Logger.WithField("Volume", v.DeviceID())
	return v, v.check("")
}

func (v *s3Volume) translateError(err error) error {
	if cerr := (interface{ CanceledError() bool })(nil); errors.As(err, &cerr) && cerr.CanceledError() {
		// *aws.RequestCanceledError and *smithy.CanceledError
		// implement this interface.
		return context.Canceled
	}
	var aerr smithy.APIError
	if errors.As(err, &aerr) {
		switch aerr.ErrorCode() {
		case "NotFound", "NoSuchKey":
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
func (v *s3Volume) safeCopy(dst, src string) error {
	input := &s3.CopyObjectInput{
		Bucket:      aws.String(v.bucket.bucket),
		ContentType: aws.String("application/octet-stream"),
		CopySource:  aws.String(v.bucket.bucket + "/" + src),
		Key:         aws.String(dst),
	}

	resp, err := v.bucket.svc.CopyObject(context.Background(), input)

	err = v.translateError(err)
	if os.IsNotExist(err) {
		return err
	} else if err != nil {
		return fmt.Errorf("PutCopy(%q ← %q): %s", dst, v.bucket.bucket+"/"+src, err)
	} else if resp.CopyObjectResult.LastModified == nil {
		return fmt.Errorf("PutCopy(%q ← %q): succeeded but did not return a timestamp", dst, v.bucket.bucket+"/"+src)
	} else if skew := time.Now().UTC().Sub(*resp.CopyObjectResult.LastModified); skew > maxClockSkew {
		return fmt.Errorf("PutCopy succeeded but returned old timestamp %s (skew %v > max %v, now %s)", resp.CopyObjectResult.LastModified, skew, maxClockSkew, time.Now())
	}
	return nil
}

func (v *s3Volume) check(ec2metadataHostname string) error {
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

	if v.Endpoint == "" && v.Region == "" {
		return fmt.Errorf("AWS region or endpoint must be specified")
	} else if v.Endpoint != "" {
		_, err := url.Parse(v.Endpoint)
		if err != nil {
			return fmt.Errorf("error parsing custom S3 endpoint %q: %w", v.Endpoint, err)
		}
		v.overrideEndpoint = &aws.Endpoint{
			URL:               v.Endpoint,
			HostnameImmutable: true,
			Source:            aws.EndpointSourceCustom,
		}
	}
	if v.Region == "" {
		// Endpoint is already specified (otherwise we would
		// have errored out above), but Region is also
		// required by the aws sdk, in order to determine
		// SignatureVersions.
		v.Region = "us-east-1"
	}

	// Zero timeouts mean "wait forever", which is a bad
	// default. Default to long timeouts instead.
	if v.ConnectTimeout == 0 {
		v.ConnectTimeout = s3DefaultConnectTimeout
	}
	if v.ReadTimeout == 0 {
		v.ReadTimeout = s3DefaultReadTimeout
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(v.Region),
		config.WithCredentialsCacheOptions(func(o *aws.CredentialsCacheOptions) {
			// (from aws-sdk-go-v2 comments) "allow the
			// credentials to trigger refreshing prior to
			// the credentials actually expiring. This is
			// beneficial so race conditions with expiring
			// credentials do not cause request to fail
			// unexpectedly due to ExpiredTokenException
			// exceptions."
			//
			// (from
			// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html)
			// "We make new credentials available at least
			// five minutes before the expiration of the
			// old credentials."
			o.ExpiryWindow = 5 * time.Minute
		}),
		func(o *config.LoadOptions) error {
			if v.AccessKeyID == "" && v.SecretAccessKey == "" {
				// Use default sdk behavior (IAM / IMDS)
				return nil
			}
			v.logger.Debug("using static credentials")
			o.Credentials = credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     v.AccessKeyID,
					SecretAccessKey: v.SecretAccessKey,
					Source:          "Arvados configuration",
				},
			}
			return nil
		},
		func(o *config.LoadOptions) error {
			if ec2metadataHostname != "" {
				o.EC2IMDSEndpoint = ec2metadataHostname
			}
			if v.overrideEndpoint != nil {
				o.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					if service == "S3" {
						return *v.overrideEndpoint, nil
					}
					return aws.Endpoint{}, errEndpointNotOverridden // use default resolver
				})
			}
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("error loading aws client config: %w", err)
	}

	v.bucket = &s3Bucket{
		bucket: v.Bucket,
		svc: s3.NewFromConfig(cfg, func(o *s3.Options) {
			if v.usePathStyle {
				o.UsePathStyle = true
			}
		}),
	}

	// Set up prometheus metrics
	lbls := prometheus.Labels{"device_id": v.DeviceID()}
	v.bucket.stats.opsCounters, v.bucket.stats.errCounters, v.bucket.stats.ioBytes = v.metrics.getCounterVecsFor(lbls)

	return nil
}

// DeviceID returns a globally unique ID for the storage bucket.
func (v *s3Volume) DeviceID() string {
	return "s3://" + v.Endpoint + "/" + v.Bucket
}

// EmptyTrash looks for trashed blocks that exceeded BlobTrashLifetime
// and deletes them from the volume.
func (v *s3Volume) EmptyTrash() {
	var bytesInTrash, blocksInTrash, bytesDeleted, blocksDeleted int64

	// Define "ready to delete" as "...when EmptyTrash started".
	startT := time.Now()

	emptyOneKey := func(trash *types.Object) {
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
			err = v.BlockUntrash(loc)
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
				v.BlockTouch(loc)
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
	todo := make(chan *types.Object, v.cluster.Collections.BlobDeleteConcurrency)
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
	v.logger.Infof("EmptyTrash: stats for %v: Deleted %v bytes in %v blocks. Remaining in trash: %v bytes in %v blocks.", v.DeviceID(), bytesDeleted, blocksDeleted, bytesInTrash-bytesDeleted, blocksInTrash-blocksDeleted)
}

// fixRace(X) is called when "recent/X" exists but "X" doesn't
// exist. If the timestamps on "recent/X" and "trash/X" indicate there
// was a race between Put and Trash, fixRace recovers from the race by
// Untrashing the block.
func (v *s3Volume) fixRace(key string) bool {
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

func (v *s3Volume) head(key string) (result *s3.HeadObjectOutput, err error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
	}

	res, err := v.bucket.svc.HeadObject(context.Background(), input)

	v.bucket.stats.TickOps("head")
	v.bucket.stats.Tick(&v.bucket.stats.Ops, &v.bucket.stats.HeadOps)
	v.bucket.stats.TickErr(err)

	if err != nil {
		return nil, v.translateError(err)
	}
	return res, nil
}

// BlockRead reads a Keep block that has been stored as a block blob
// in the S3 bucket.
func (v *s3Volume) BlockRead(ctx context.Context, hash string, w io.WriterAt) error {
	key := v.key(hash)
	err := v.readWorker(ctx, key, w)
	if err != nil {
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

		err = v.readWorker(ctx, key, w)
		if err != nil {
			v.logger.Warnf("reading %s after successful fixRace: %s", hash, err)
			err = v.translateError(err)
			return err
		}
	}
	return nil
}

func (v *s3Volume) readWorker(ctx context.Context, key string, dst io.WriterAt) error {
	downloader := manager.NewDownloader(v.bucket.svc, func(u *manager.Downloader) {
		u.PartSize = s3downloaderPartSize
		u.Concurrency = s3downloaderReadConcurrency
	})
	count, err := downloader.Download(ctx, dst, &s3.GetObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
	})
	v.bucket.stats.TickOps("get")
	v.bucket.stats.Tick(&v.bucket.stats.Ops, &v.bucket.stats.GetOps)
	v.bucket.stats.TickErr(err)
	v.bucket.stats.TickInBytes(uint64(count))
	return v.translateError(err)
}

func (v *s3Volume) writeObject(ctx context.Context, key string, r io.Reader) error {
	if r == nil {
		// r == nil leads to a memory violation in func readFillBuf in
		// aws-sdk-go-v2@v0.23.0/service/s3/s3manager/upload.go
		r = bytes.NewReader(nil)
	}

	uploadInput := s3.PutObjectInput{
		Bucket: aws.String(v.bucket.bucket),
		Key:    aws.String(key),
		Body:   r,
	}

	if loc, ok := v.isKeepBlock(key); ok {
		var contentMD5 string
		md5, err := hex.DecodeString(loc)
		if err != nil {
			return v.translateError(err)
		}
		contentMD5 = base64.StdEncoding.EncodeToString(md5)
		uploadInput.ContentMD5 = &contentMD5
	}

	// Experimentation indicated that using concurrency 5 yields the best
	// throughput, better than higher concurrency (10 or 13) by ~5%.
	// Defining u.BufferProvider = s3manager.NewBufferedReadSeekerWriteToPool(64 * 1024 * 1024)
	// is detrimental to throughput (minus ~15%).
	uploader := manager.NewUploader(v.bucket.svc, func(u *manager.Uploader) {
		u.PartSize = s3uploaderPartSize
		u.Concurrency = s3uploaderWriteConcurrency
	})

	_, err := uploader.Upload(ctx, &uploadInput,
		// Avoid precomputing SHA256 before sending.
		manager.WithUploaderRequestOptions(s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware)),
	)

	v.bucket.stats.TickOps("put")
	v.bucket.stats.Tick(&v.bucket.stats.Ops, &v.bucket.stats.PutOps)
	v.bucket.stats.TickErr(err)

	return v.translateError(err)
}

// Put writes a block.
func (v *s3Volume) BlockWrite(ctx context.Context, hash string, data []byte) error {
	// Do not use putWithPipe here; we want to pass an io.ReadSeeker to the S3
	// sdk to avoid memory allocation there. See #17339 for more information.
	rdr := bytes.NewReader(data)
	r := newCountingReaderAtSeeker(rdr, v.bucket.stats.TickOutBytes)
	key := v.key(hash)
	err := v.writeObject(ctx, key, r)
	if err != nil {
		return err
	}
	return v.writeObject(ctx, "recent/"+key, nil)
}

type s3awsLister struct {
	Logger            logrus.FieldLogger
	Bucket            *s3Bucket
	Prefix            string
	PageSize          int
	Stats             *s3awsbucketStats
	ContinuationToken string
	buf               []types.Object
	err               error
}

// First fetches the first page and returns the first item. It returns
// nil if the response is the empty set or an error occurs.
func (lister *s3awsLister) First() *types.Object {
	lister.getPage()
	return lister.pop()
}

// Next returns the next item, fetching the next page if necessary. It
// returns nil if the last available item has already been fetched, or
// an error occurs.
func (lister *s3awsLister) Next() *types.Object {
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
			MaxKeys: aws.Int32(int32(lister.PageSize)),
			Prefix:  aws.String(lister.Prefix),
		}
	} else {
		input = &s3.ListObjectsV2Input{
			Bucket:            aws.String(lister.Bucket.bucket),
			MaxKeys:           aws.Int32(int32(lister.PageSize)),
			Prefix:            aws.String(lister.Prefix),
			ContinuationToken: &lister.ContinuationToken,
		}
	}

	resp, err := lister.Bucket.svc.ListObjectsV2(context.Background(), input)
	if err != nil {
		var aerr smithy.APIError
		if errors.As(err, &aerr) {
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
	lister.buf = make([]types.Object, 0, len(resp.Contents))
	for _, key := range resp.Contents {
		if !strings.HasPrefix(*key.Key, lister.Prefix) {
			lister.Logger.Warnf("s3awsLister: S3 Bucket.List(prefix=%q) returned key %q", lister.Prefix, *key.Key)
			continue
		}
		lister.buf = append(lister.buf, key)
	}
}

func (lister *s3awsLister) pop() (k *types.Object) {
	if len(lister.buf) > 0 {
		k = &lister.buf[0]
		lister.buf = lister.buf[1:]
	}
	return
}

// Index writes a complete list of locators with the given prefix
// for which Get() can retrieve data.
func (v *s3Volume) Index(ctx context.Context, prefix string, writer io.Writer) error {
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
		if ctx.Err() != nil {
			return ctx.Err()
		}
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
func (v *s3Volume) Mtime(loc string) (time.Time, error) {
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

// InternalStats returns bucket I/O and API call counters.
func (v *s3Volume) InternalStats() interface{} {
	return &v.bucket.stats
}

// BlockTouch sets the timestamp for the given locator to the current time.
func (v *s3Volume) BlockTouch(hash string) error {
	key := v.key(hash)
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
func (v *s3Volume) checkRaceWindow(key string) error {
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

func (b *s3Bucket) Del(path string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(path),
	}
	_, err := b.svc.DeleteObject(context.Background(), input)
	b.stats.TickOps("delete")
	b.stats.Tick(&b.stats.Ops, &b.stats.DelOps)
	b.stats.TickErr(err)
	return err
}

// Trash a Keep block.
func (v *s3Volume) BlockTrash(loc string) error {
	if t, err := v.Mtime(loc); err != nil {
		return err
	} else if time.Since(t) < v.cluster.Collections.BlobSigningTTL.Duration() {
		return nil
	}
	key := v.key(loc)
	if v.cluster.Collections.BlobTrashLifetime == 0 {
		if !v.UnsafeDelete {
			return errS3TrashDisabled
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

// BlockUntrash moves block from trash back into store
func (v *s3Volume) BlockUntrash(hash string) error {
	key := v.key(hash)
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
	if aerr := smithy.APIError(nil); errors.As(err, &aerr) {
		if rerr := interface{ HTTPStatusCode() int }(nil); errors.As(err, &rerr) {
			errType = errType + fmt.Sprintf(" %d %s", rerr.HTTPStatusCode(), aerr.ErrorCode())
		} else {
			errType = errType + fmt.Sprintf(" 000 %s", aerr.ErrorCode())
		}
	}
	s.statsTicker.TickErr(err, errType)
}
