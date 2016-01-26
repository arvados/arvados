package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/s3"
)

var (
	ErrS3DeleteNotAvailable = fmt.Errorf("delete without -s3-unsafe-delete is not implemented")

	s3AccessKeyFile string
	s3SecretKeyFile string
	s3RegionName    string
	s3Endpoint      string
	s3Replication   int
	s3UnsafeDelete  bool

	s3ACL = s3.Private
)

const (
	maxClockSkew  = 600 * time.Second
	nearlyRFC1123 = "Mon, 2 Jan 2006 15:04:05 GMT"
)

type s3VolumeAdder struct {
	*volumeSet
}

func (s *s3VolumeAdder) Set(bucketName string) error {
	if trashLifetime != 0 {
		return ErrNotImplemented
	}
	if bucketName == "" {
		return fmt.Errorf("no container name given")
	}
	if s3AccessKeyFile == "" || s3SecretKeyFile == "" {
		return fmt.Errorf("-s3-access-key-file and -s3-secret-key-file arguments must given before -s3-bucket-volume")
	}
	region, ok := aws.Regions[s3RegionName]
	if s3Endpoint == "" {
		if !ok {
			return fmt.Errorf("unrecognized region %+q; try specifying -s3-endpoint instead", s3RegionName)
		}
	} else {
		if ok {
			return fmt.Errorf("refusing to use AWS region name %+q with endpoint %+q; "+
				"specify empty endpoint (\"-s3-endpoint=\") or use a different region name", s3RegionName, s3Endpoint)
		}
		region = aws.Region{
			Name:       s3RegionName,
			S3Endpoint: s3Endpoint,
		}
	}
	var err error
	var auth aws.Auth
	auth.AccessKey, err = readKeyFromFile(s3AccessKeyFile)
	if err != nil {
		return err
	}
	auth.SecretKey, err = readKeyFromFile(s3SecretKeyFile)
	if err != nil {
		return err
	}
	if flagSerializeIO {
		log.Print("Notice: -serialize is not supported by s3-bucket volumes.")
	}
	v := NewS3Volume(auth, region, bucketName, flagReadonly, s3Replication)
	if err := v.Check(); err != nil {
		return err
	}
	*s.volumeSet = append(*s.volumeSet, v)
	return nil
}

func s3regions() (okList []string) {
	for r, _ := range aws.Regions {
		okList = append(okList, r)
	}
	return
}

func init() {
	flag.Var(&s3VolumeAdder{&volumes},
		"s3-bucket-volume",
		"Use the given bucket as a storage volume. Can be given multiple times.")
	flag.StringVar(
		&s3RegionName,
		"s3-region",
		"",
		fmt.Sprintf("AWS region used for subsequent -s3-bucket-volume arguments. Allowed values are %+q.", s3regions()))
	flag.StringVar(
		&s3Endpoint,
		"s3-endpoint",
		"",
		"Endpoint URL used for subsequent -s3-bucket-volume arguments. If blank, use the AWS endpoint corresponding to the -s3-region argument. For Google Storage, use \"https://storage.googleapis.com\".")
	flag.StringVar(
		&s3AccessKeyFile,
		"s3-access-key-file",
		"",
		"File containing the access key used for subsequent -s3-bucket-volume arguments.")
	flag.StringVar(
		&s3SecretKeyFile,
		"s3-secret-key-file",
		"",
		"File containing the secret key used for subsequent -s3-bucket-volume arguments.")
	flag.IntVar(
		&s3Replication,
		"s3-replication",
		2,
		"Replication level reported to clients for subsequent -s3-bucket-volume arguments.")
	flag.BoolVar(
		&s3UnsafeDelete,
		"s3-unsafe-delete",
		false,
		"EXPERIMENTAL. Enable deletion (garbage collection), even though there are known race conditions that can cause data loss.")
}

type S3Volume struct {
	*s3.Bucket
	readonly      bool
	replication   int
	indexPageSize int
}

// NewS3Volume returns a new S3Volume using the given auth, region,
// and bucket name. The replication argument specifies the replication
// level to report when writing data.
func NewS3Volume(auth aws.Auth, region aws.Region, bucket string, readonly bool, replication int) *S3Volume {
	return &S3Volume{
		Bucket: &s3.Bucket{
			S3:   s3.New(auth, region),
			Name: bucket,
		},
		readonly:      readonly,
		replication:   replication,
		indexPageSize: 1000,
	}
}

func (v *S3Volume) Check() error {
	return nil
}

func (v *S3Volume) Get(loc string) ([]byte, error) {
	rdr, err := v.Bucket.GetReader(loc)
	if err != nil {
		return nil, v.translateError(err)
	}
	defer rdr.Close()
	buf := bufs.Get(BlockSize)
	n, err := io.ReadFull(rdr, buf)
	switch err {
	case nil, io.EOF, io.ErrUnexpectedEOF:
		return buf[:n], nil
	default:
		bufs.Put(buf)
		return nil, v.translateError(err)
	}
}

func (v *S3Volume) Compare(loc string, expect []byte) error {
	rdr, err := v.Bucket.GetReader(loc)
	if err != nil {
		return v.translateError(err)
	}
	defer rdr.Close()
	return v.translateError(compareReaderWithBuf(rdr, expect, loc[:32]))
}

func (v *S3Volume) Put(loc string, block []byte) error {
	if v.readonly {
		return MethodDisabledError
	}
	var opts s3.Options
	if len(block) > 0 {
		md5, err := hex.DecodeString(loc)
		if err != nil {
			return err
		}
		opts.ContentMD5 = base64.StdEncoding.EncodeToString(md5)
	}
	return v.translateError(
		v.Bucket.Put(
			loc, block, "application/octet-stream", s3ACL, opts))
}

func (v *S3Volume) Touch(loc string) error {
	if v.readonly {
		return MethodDisabledError
	}
	result, err := v.Bucket.PutCopy(loc, s3ACL, s3.CopyOptions{
		ContentType:       "application/octet-stream",
		MetadataDirective: "REPLACE",
	}, v.Bucket.Name+"/"+loc)
	if err != nil {
		return v.translateError(err)
	}
	t, err := time.Parse(time.RFC3339, result.LastModified)
	if err != nil {
		return err
	}
	if time.Since(t) > maxClockSkew {
		return fmt.Errorf("PutCopy returned old LastModified %s => %s (%s ago)", result.LastModified, t, time.Since(t))
	}
	return nil
}

func (v *S3Volume) Mtime(loc string) (time.Time, error) {
	resp, err := v.Bucket.Head(loc, nil)
	if err != nil {
		return zeroTime, v.translateError(err)
	}
	hdr := resp.Header.Get("Last-Modified")
	t, err := time.Parse(time.RFC1123, hdr)
	if err != nil && hdr != "" {
		// AWS example is "Sun, 1 Jan 2006 12:00:00 GMT",
		// which isn't quite "Sun, 01 Jan 2006 12:00:00 GMT"
		// as required by HTTP spec. If it's not a valid HTTP
		// header value, it's probably AWS (or s3test) giving
		// us a nearly-RFC1123 timestamp.
		t, err = time.Parse(nearlyRFC1123, hdr)
	}
	return t, err
}

func (v *S3Volume) IndexTo(prefix string, writer io.Writer) error {
	nextMarker := ""
	for {
		listResp, err := v.Bucket.List(prefix, "", nextMarker, v.indexPageSize)
		if err != nil {
			return err
		}
		for _, key := range listResp.Contents {
			t, err := time.Parse(time.RFC3339, key.LastModified)
			if err != nil {
				return err
			}
			if !v.isKeepBlock(key.Key) {
				continue
			}
			fmt.Fprintf(writer, "%s+%d %d\n", key.Key, key.Size, t.Unix())
		}
		if !listResp.IsTruncated {
			break
		}
		nextMarker = listResp.NextMarker
	}
	return nil
}

func (v *S3Volume) Trash(loc string) error {
	if v.readonly {
		return MethodDisabledError
	}
	if trashLifetime != 0 {
		return ErrNotImplemented
	}
	if t, err := v.Mtime(loc); err != nil {
		return err
	} else if time.Since(t) < blobSignatureTTL {
		return nil
	}
	if !s3UnsafeDelete {
		return ErrS3DeleteNotAvailable
	}
	return v.Bucket.Del(loc)
}

// TBD
func (v *S3Volume) Untrash(loc string) error {
	return ErrNotImplemented
}

func (v *S3Volume) Status() *VolumeStatus {
	return &VolumeStatus{
		DeviceNum: 1,
		BytesFree: BlockSize * 1000,
		BytesUsed: 1,
	}
}

func (v *S3Volume) String() string {
	return fmt.Sprintf("s3-bucket:%+q", v.Bucket.Name)
}

func (v *S3Volume) Writable() bool {
	return !v.readonly
}
func (v *S3Volume) Replication() int {
	return v.replication
}

var s3KeepBlockRegexp = regexp.MustCompile(`^[0-9a-f]{32}$`)

func (v *S3Volume) isKeepBlock(s string) bool {
	return s3KeepBlockRegexp.MatchString(s)
}

func (v *S3Volume) translateError(err error) error {
	switch err := err.(type) {
	case *s3.Error:
		if err.StatusCode == http.StatusNotFound && err.Code == "NoSuchKey" {
			return os.ErrNotExist
		}
		// Other 404 errors like NoSuchVersion and
		// NoSuchBucket are different problems which should
		// get called out downstream, so we don't convert them
		// to os.ErrNotExist.
	}
	return err
}
