// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func init() {
	driver["Azure"] = newAzureBlobVolume
}

func newAzureBlobVolume(cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) (Volume, error) {
	v := &AzureBlobVolume{
		RequestTimeout:    azureDefaultRequestTimeout,
		WriteRaceInterval: azureDefaultWriteRaceInterval,
		WriteRacePollTime: azureDefaultWriteRacePollTime,
		cluster:           cluster,
		volume:            volume,
		logger:            logger,
		metrics:           metrics,
	}
	err := json.Unmarshal(volume.DriverParameters, &v)
	if err != nil {
		return nil, err
	}
	if v.ListBlobsRetryDelay == 0 {
		v.ListBlobsRetryDelay = azureDefaultListBlobsRetryDelay
	}
	if v.ListBlobsMaxAttempts == 0 {
		v.ListBlobsMaxAttempts = azureDefaultListBlobsMaxAttempts
	}
	if v.StorageBaseURL == "" {
		v.StorageBaseURL = storage.DefaultBaseURL
	}
	if v.ContainerName == "" || v.StorageAccountName == "" || v.StorageAccountKey == "" {
		return nil, errors.New("DriverParameters: ContainerName, StorageAccountName, and StorageAccountKey must be provided")
	}
	azc, err := storage.NewClient(v.StorageAccountName, v.StorageAccountKey, v.StorageBaseURL, storage.DefaultAPIVersion, true)
	if err != nil {
		return nil, fmt.Errorf("creating Azure storage client: %s", err)
	}
	v.azClient = azc
	v.azClient.Sender = &singleSender{}
	v.azClient.HTTPClient = &http.Client{
		Timeout: time.Duration(v.RequestTimeout),
	}
	bs := v.azClient.GetBlobService()
	v.container = &azureContainer{
		ctr: bs.GetContainerReference(v.ContainerName),
	}

	if ok, err := v.container.Exists(); err != nil {
		return nil, err
	} else if !ok {
		return nil, fmt.Errorf("Azure container %q does not exist: %s", v.ContainerName, err)
	}
	return v, v.check()
}

func (v *AzureBlobVolume) check() error {
	lbls := prometheus.Labels{"device_id": v.GetDeviceID()}
	v.container.stats.opsCounters, v.container.stats.errCounters, v.container.stats.ioBytes = v.metrics.getCounterVecsFor(lbls)
	return nil
}

const (
	azureDefaultRequestTimeout       = arvados.Duration(10 * time.Minute)
	azureDefaultListBlobsMaxAttempts = 12
	azureDefaultListBlobsRetryDelay  = arvados.Duration(10 * time.Second)
	azureDefaultWriteRaceInterval    = arvados.Duration(15 * time.Second)
	azureDefaultWriteRacePollTime    = arvados.Duration(time.Second)
)

// An AzureBlobVolume stores and retrieves blocks in an Azure Blob
// container.
type AzureBlobVolume struct {
	StorageAccountName   string
	StorageAccountKey    string
	StorageBaseURL       string // "" means default, "core.windows.net"
	ContainerName        string
	RequestTimeout       arvados.Duration
	ListBlobsRetryDelay  arvados.Duration
	ListBlobsMaxAttempts int
	MaxGetBytes          int
	WriteRaceInterval    arvados.Duration
	WriteRacePollTime    arvados.Duration

	cluster   *arvados.Cluster
	volume    arvados.Volume
	logger    logrus.FieldLogger
	metrics   *volumeMetricsVecs
	azClient  storage.Client
	container *azureContainer
}

// singleSender is a single-attempt storage.Sender.
type singleSender struct{}

// Send performs req exactly once.
func (*singleSender) Send(c *storage.Client, req *http.Request) (resp *http.Response, err error) {
	return c.HTTPClient.Do(req)
}

// Type implements Volume.
func (v *AzureBlobVolume) Type() string {
	return "Azure"
}

// GetDeviceID returns a globally unique ID for the storage container.
func (v *AzureBlobVolume) GetDeviceID() string {
	return "azure://" + v.StorageBaseURL + "/" + v.StorageAccountName + "/" + v.ContainerName
}

// Return true if expires_at metadata attribute is found on the block
func (v *AzureBlobVolume) checkTrashed(loc string) (bool, map[string]string, error) {
	metadata, err := v.container.GetBlobMetadata(loc)
	if err != nil {
		return false, metadata, v.translateError(err)
	}
	if metadata["expires_at"] != "" {
		return true, metadata, nil
	}
	return false, metadata, nil
}

// Get reads a Keep block that has been stored as a block blob in the
// container.
//
// If the block is younger than azureWriteRaceInterval and is
// unexpectedly empty, assume a PutBlob operation is in progress, and
// wait for it to finish writing.
func (v *AzureBlobVolume) Get(ctx context.Context, loc string, buf []byte) (int, error) {
	trashed, _, err := v.checkTrashed(loc)
	if err != nil {
		return 0, err
	}
	if trashed {
		return 0, os.ErrNotExist
	}
	var deadline time.Time
	haveDeadline := false
	size, err := v.get(ctx, loc, buf)
	for err == nil && size == 0 && loc != "d41d8cd98f00b204e9800998ecf8427e" {
		// Seeing a brand new empty block probably means we're
		// in a race with CreateBlob, which under the hood
		// (apparently) does "CreateEmpty" and "CommitData"
		// with no additional transaction locking.
		if !haveDeadline {
			t, err := v.Mtime(loc)
			if err != nil {
				ctxlog.FromContext(ctx).Print("Got empty block (possible race) but Mtime failed: ", err)
				break
			}
			deadline = t.Add(v.WriteRaceInterval.Duration())
			if time.Now().After(deadline) {
				break
			}
			ctxlog.FromContext(ctx).Printf("Race? Block %s is 0 bytes, %s old. Polling until %s", loc, time.Since(t), deadline)
			haveDeadline = true
		} else if time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(v.WriteRacePollTime.Duration()):
		}
		size, err = v.get(ctx, loc, buf)
	}
	if haveDeadline {
		ctxlog.FromContext(ctx).Printf("Race ended with size==%d", size)
	}
	return size, err
}

func (v *AzureBlobVolume) get(ctx context.Context, loc string, buf []byte) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pieceSize := BlockSize
	if v.MaxGetBytes > 0 && v.MaxGetBytes < BlockSize {
		pieceSize = v.MaxGetBytes
	}

	pieces := 1
	expectSize := len(buf)
	if pieceSize < BlockSize {
		// Unfortunately the handler doesn't tell us how long the blob
		// is expected to be, so we have to ask Azure.
		props, err := v.container.GetBlobProperties(loc)
		if err != nil {
			return 0, v.translateError(err)
		}
		if props.ContentLength > int64(BlockSize) || props.ContentLength < 0 {
			return 0, fmt.Errorf("block %s invalid size %d (max %d)", loc, props.ContentLength, BlockSize)
		}
		expectSize = int(props.ContentLength)
		pieces = (expectSize + pieceSize - 1) / pieceSize
	}

	if expectSize == 0 {
		return 0, nil
	}

	// We'll update this actualSize if/when we get the last piece.
	actualSize := -1
	errors := make(chan error, pieces)
	var wg sync.WaitGroup
	wg.Add(pieces)
	for p := 0; p < pieces; p++ {
		// Each goroutine retrieves one piece. If we hit an
		// error, it is sent to the errors chan so get() can
		// return it -- but only if the error happens before
		// ctx is done. This way, if ctx is done before we hit
		// any other error (e.g., requesting client has hung
		// up), we return the original ctx.Err() instead of
		// the secondary errors from the transfers that got
		// interrupted as a result.
		go func(p int) {
			defer wg.Done()
			startPos := p * pieceSize
			endPos := startPos + pieceSize
			if endPos > expectSize {
				endPos = expectSize
			}
			var rdr io.ReadCloser
			var err error
			gotRdr := make(chan struct{})
			go func() {
				defer close(gotRdr)
				if startPos == 0 && endPos == expectSize {
					rdr, err = v.container.GetBlob(loc)
				} else {
					rdr, err = v.container.GetBlobRange(loc, startPos, endPos-1, nil)
				}
			}()
			select {
			case <-ctx.Done():
				go func() {
					<-gotRdr
					if err == nil {
						rdr.Close()
					}
				}()
				return
			case <-gotRdr:
			}
			if err != nil {
				errors <- err
				cancel()
				return
			}
			go func() {
				// Close the reader when the client
				// hangs up or another piece fails
				// (possibly interrupting ReadFull())
				// or when all pieces succeed and
				// get() returns.
				<-ctx.Done()
				rdr.Close()
			}()
			n, err := io.ReadFull(rdr, buf[startPos:endPos])
			if pieces == 1 && (err == io.ErrUnexpectedEOF || err == io.EOF) {
				// If we don't know the actual size,
				// and just tried reading 64 MiB, it's
				// normal to encounter EOF.
			} else if err != nil {
				if ctx.Err() == nil {
					errors <- err
				}
				cancel()
				return
			}
			if p == pieces-1 {
				actualSize = startPos + n
			}
		}(p)
	}
	wg.Wait()
	close(errors)
	if len(errors) > 0 {
		return 0, v.translateError(<-errors)
	}
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}
	return actualSize, nil
}

// Compare the given data with existing stored data.
func (v *AzureBlobVolume) Compare(ctx context.Context, loc string, expect []byte) error {
	trashed, _, err := v.checkTrashed(loc)
	if err != nil {
		return err
	}
	if trashed {
		return os.ErrNotExist
	}
	var rdr io.ReadCloser
	gotRdr := make(chan struct{})
	go func() {
		defer close(gotRdr)
		rdr, err = v.container.GetBlob(loc)
	}()
	select {
	case <-ctx.Done():
		go func() {
			<-gotRdr
			if err == nil {
				rdr.Close()
			}
		}()
		return ctx.Err()
	case <-gotRdr:
	}
	if err != nil {
		return v.translateError(err)
	}
	defer rdr.Close()
	return compareReaderWithBuf(ctx, rdr, expect, loc[:32])
}

// Put stores a Keep block as a block blob in the container.
func (v *AzureBlobVolume) Put(ctx context.Context, loc string, block []byte) error {
	if v.volume.ReadOnly {
		return MethodDisabledError
	}
	// Send the block data through a pipe, so that (if we need to)
	// we can close the pipe early and abandon our
	// CreateBlockBlobFromReader() goroutine, without worrying
	// about CreateBlockBlobFromReader() accessing our block
	// buffer after we release it.
	bufr, bufw := io.Pipe()
	go func() {
		io.Copy(bufw, bytes.NewReader(block))
		bufw.Close()
	}()
	errChan := make(chan error)
	go func() {
		var body io.Reader = bufr
		if len(block) == 0 {
			// We must send a "Content-Length: 0" header,
			// but the http client interprets
			// ContentLength==0 as "unknown" unless it can
			// confirm by introspection that Body will
			// read 0 bytes.
			body = http.NoBody
			bufr.Close()
		}
		errChan <- v.container.CreateBlockBlobFromReader(loc, len(block), body, nil)
	}()
	select {
	case <-ctx.Done():
		ctxlog.FromContext(ctx).Debugf("%s: taking CreateBlockBlobFromReader's input away: %s", v, ctx.Err())
		// Our pipe might be stuck in Write(), waiting for
		// io.Copy() to read. If so, un-stick it. This means
		// CreateBlockBlobFromReader will get corrupt data,
		// but that's OK: the size won't match, so the write
		// will fail.
		go io.Copy(ioutil.Discard, bufr)
		// CloseWithError() will return once pending I/O is done.
		bufw.CloseWithError(ctx.Err())
		ctxlog.FromContext(ctx).Debugf("%s: abandoning CreateBlockBlobFromReader goroutine", v)
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// Touch updates the last-modified property of a block blob.
func (v *AzureBlobVolume) Touch(loc string) error {
	if v.volume.ReadOnly {
		return MethodDisabledError
	}
	trashed, metadata, err := v.checkTrashed(loc)
	if err != nil {
		return err
	}
	if trashed {
		return os.ErrNotExist
	}

	metadata["touch"] = fmt.Sprintf("%d", time.Now().Unix())
	return v.container.SetBlobMetadata(loc, metadata, nil)
}

// Mtime returns the last-modified property of a block blob.
func (v *AzureBlobVolume) Mtime(loc string) (time.Time, error) {
	trashed, _, err := v.checkTrashed(loc)
	if err != nil {
		return time.Time{}, err
	}
	if trashed {
		return time.Time{}, os.ErrNotExist
	}

	props, err := v.container.GetBlobProperties(loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Time(props.LastModified), nil
}

// IndexTo writes a list of Keep blocks that are stored in the
// container.
func (v *AzureBlobVolume) IndexTo(prefix string, writer io.Writer) error {
	params := storage.ListBlobsParameters{
		Prefix:  prefix,
		Include: &storage.IncludeBlobDataset{Metadata: true},
	}
	for page := 1; ; page++ {
		resp, err := v.listBlobs(page, params)
		if err != nil {
			return err
		}
		for _, b := range resp.Blobs {
			if !v.isKeepBlock(b.Name) {
				continue
			}
			modtime := time.Time(b.Properties.LastModified)
			if b.Properties.ContentLength == 0 && modtime.Add(v.WriteRaceInterval.Duration()).After(time.Now()) {
				// A new zero-length blob is probably
				// just a new non-empty blob that
				// hasn't committed its data yet (see
				// Get()), and in any case has no
				// value.
				continue
			}
			if b.Metadata["expires_at"] != "" {
				// Trashed blob; exclude it from response
				continue
			}
			fmt.Fprintf(writer, "%s+%d %d\n", b.Name, b.Properties.ContentLength, modtime.UnixNano())
		}
		if resp.NextMarker == "" {
			return nil
		}
		params.Marker = resp.NextMarker
	}
}

// call v.container.ListBlobs, retrying if needed.
func (v *AzureBlobVolume) listBlobs(page int, params storage.ListBlobsParameters) (resp storage.BlobListResponse, err error) {
	for i := 0; i < v.ListBlobsMaxAttempts; i++ {
		resp, err = v.container.ListBlobs(params)
		err = v.translateError(err)
		if err == VolumeBusyError {
			v.logger.Printf("ListBlobs: will retry page %d in %s after error: %s", page, v.ListBlobsRetryDelay, err)
			time.Sleep(time.Duration(v.ListBlobsRetryDelay))
			continue
		} else {
			break
		}
	}
	return
}

// Trash a Keep block.
func (v *AzureBlobVolume) Trash(loc string) error {
	if v.volume.ReadOnly {
		return MethodDisabledError
	}

	// Ideally we would use If-Unmodified-Since, but that
	// particular condition seems to be ignored by Azure. Instead,
	// we get the Etag before checking Mtime, and use If-Match to
	// ensure we don't delete data if Put() or Touch() happens
	// between our calls to Mtime() and DeleteBlob().
	props, err := v.container.GetBlobProperties(loc)
	if err != nil {
		return err
	}
	if t, err := v.Mtime(loc); err != nil {
		return err
	} else if time.Since(t) < v.cluster.Collections.BlobSigningTTL.Duration() {
		return nil
	}

	// If BlobTrashLifetime == 0, just delete it
	if v.cluster.Collections.BlobTrashLifetime == 0 {
		return v.container.DeleteBlob(loc, &storage.DeleteBlobOptions{
			IfMatch: props.Etag,
		})
	}

	// Otherwise, mark as trash
	return v.container.SetBlobMetadata(loc, storage.BlobMetadata{
		"expires_at": fmt.Sprintf("%d", time.Now().Add(v.cluster.Collections.BlobTrashLifetime.Duration()).Unix()),
	}, &storage.SetBlobMetadataOptions{
		IfMatch: props.Etag,
	})
}

// Untrash a Keep block.
// Delete the expires_at metadata attribute
func (v *AzureBlobVolume) Untrash(loc string) error {
	// if expires_at does not exist, return NotFoundError
	metadata, err := v.container.GetBlobMetadata(loc)
	if err != nil {
		return v.translateError(err)
	}
	if metadata["expires_at"] == "" {
		return os.ErrNotExist
	}

	// reset expires_at metadata attribute
	metadata["expires_at"] = ""
	err = v.container.SetBlobMetadata(loc, metadata, nil)
	return v.translateError(err)
}

// Status returns a VolumeStatus struct with placeholder data.
func (v *AzureBlobVolume) Status() *VolumeStatus {
	return &VolumeStatus{
		DeviceNum: 1,
		BytesFree: BlockSize * 1000,
		BytesUsed: 1,
	}
}

// String returns a volume label, including the container name.
func (v *AzureBlobVolume) String() string {
	return fmt.Sprintf("azure-storage-container:%+q", v.ContainerName)
}

// If possible, translate an Azure SDK error to a recognizable error
// like os.ErrNotExist.
func (v *AzureBlobVolume) translateError(err error) error {
	switch {
	case err == nil:
		return err
	case strings.Contains(err.Error(), "StatusCode=503"):
		// "storage: service returned error: StatusCode=503, ErrorCode=ServerBusy, ErrorMessage=The server is busy" (See #14804)
		return VolumeBusyError
	case strings.Contains(err.Error(), "Not Found"):
		// "storage: service returned without a response body (404 Not Found)"
		return os.ErrNotExist
	case strings.Contains(err.Error(), "ErrorCode=BlobNotFound"):
		// "storage: service returned error: StatusCode=404, ErrorCode=BlobNotFound, ErrorMessage=The specified blob does not exist.\n..."
		return os.ErrNotExist
	default:
		return err
	}
}

var keepBlockRegexp = regexp.MustCompile(`^[0-9a-f]{32}$`)

func (v *AzureBlobVolume) isKeepBlock(s string) bool {
	return keepBlockRegexp.MatchString(s)
}

// EmptyTrash looks for trashed blocks that exceeded BlobTrashLifetime
// and deletes them from the volume.
func (v *AzureBlobVolume) EmptyTrash() {
	if v.cluster.Collections.BlobDeleteConcurrency < 1 {
		return
	}

	var bytesDeleted, bytesInTrash int64
	var blocksDeleted, blocksInTrash int64

	doBlob := func(b storage.Blob) {
		// Check whether the block is flagged as trash
		if b.Metadata["expires_at"] == "" {
			return
		}

		atomic.AddInt64(&blocksInTrash, 1)
		atomic.AddInt64(&bytesInTrash, b.Properties.ContentLength)

		expiresAt, err := strconv.ParseInt(b.Metadata["expires_at"], 10, 64)
		if err != nil {
			v.logger.Printf("EmptyTrash: ParseInt(%v): %v", b.Metadata["expires_at"], err)
			return
		}

		if expiresAt > time.Now().Unix() {
			return
		}

		err = v.container.DeleteBlob(b.Name, &storage.DeleteBlobOptions{
			IfMatch: b.Properties.Etag,
		})
		if err != nil {
			v.logger.Printf("EmptyTrash: DeleteBlob(%v): %v", b.Name, err)
			return
		}
		atomic.AddInt64(&blocksDeleted, 1)
		atomic.AddInt64(&bytesDeleted, b.Properties.ContentLength)
	}

	var wg sync.WaitGroup
	todo := make(chan storage.Blob, v.cluster.Collections.BlobDeleteConcurrency)
	for i := 0; i < v.cluster.Collections.BlobDeleteConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range todo {
				doBlob(b)
			}
		}()
	}

	params := storage.ListBlobsParameters{Include: &storage.IncludeBlobDataset{Metadata: true}}
	for page := 1; ; page++ {
		resp, err := v.listBlobs(page, params)
		if err != nil {
			v.logger.Printf("EmptyTrash: ListBlobs: %v", err)
			break
		}
		for _, b := range resp.Blobs {
			todo <- b
		}
		if resp.NextMarker == "" {
			break
		}
		params.Marker = resp.NextMarker
	}
	close(todo)
	wg.Wait()

	v.logger.Printf("EmptyTrash stats for %v: Deleted %v bytes in %v blocks. Remaining in trash: %v bytes in %v blocks.", v.String(), bytesDeleted, blocksDeleted, bytesInTrash-bytesDeleted, blocksInTrash-blocksDeleted)
}

// InternalStats returns bucket I/O and API call counters.
func (v *AzureBlobVolume) InternalStats() interface{} {
	return &v.container.stats
}

type azureBlobStats struct {
	statsTicker
	Ops              uint64
	GetOps           uint64
	GetRangeOps      uint64
	GetMetadataOps   uint64
	GetPropertiesOps uint64
	CreateOps        uint64
	SetMetadataOps   uint64
	DelOps           uint64
	ListOps          uint64
}

func (s *azureBlobStats) TickErr(err error) {
	if err == nil {
		return
	}
	errType := fmt.Sprintf("%T", err)
	if err, ok := err.(storage.AzureStorageServiceError); ok {
		errType = errType + fmt.Sprintf(" %d (%s)", err.StatusCode, err.Code)
	}
	s.statsTicker.TickErr(err, errType)
}

// azureContainer wraps storage.Container in order to count I/O and
// API usage stats.
type azureContainer struct {
	ctr   *storage.Container
	stats azureBlobStats
}

func (c *azureContainer) Exists() (bool, error) {
	c.stats.TickOps("exists")
	c.stats.Tick(&c.stats.Ops)
	ok, err := c.ctr.Exists()
	c.stats.TickErr(err)
	return ok, err
}

func (c *azureContainer) GetBlobMetadata(bname string) (storage.BlobMetadata, error) {
	c.stats.TickOps("get_metadata")
	c.stats.Tick(&c.stats.Ops, &c.stats.GetMetadataOps)
	b := c.ctr.GetBlobReference(bname)
	err := b.GetMetadata(nil)
	c.stats.TickErr(err)
	return b.Metadata, err
}

func (c *azureContainer) GetBlobProperties(bname string) (*storage.BlobProperties, error) {
	c.stats.TickOps("get_properties")
	c.stats.Tick(&c.stats.Ops, &c.stats.GetPropertiesOps)
	b := c.ctr.GetBlobReference(bname)
	err := b.GetProperties(nil)
	c.stats.TickErr(err)
	return &b.Properties, err
}

func (c *azureContainer) GetBlob(bname string) (io.ReadCloser, error) {
	c.stats.TickOps("get")
	c.stats.Tick(&c.stats.Ops, &c.stats.GetOps)
	b := c.ctr.GetBlobReference(bname)
	rdr, err := b.Get(nil)
	c.stats.TickErr(err)
	return NewCountingReader(rdr, c.stats.TickInBytes), err
}

func (c *azureContainer) GetBlobRange(bname string, start, end int, opts *storage.GetBlobOptions) (io.ReadCloser, error) {
	c.stats.TickOps("get_range")
	c.stats.Tick(&c.stats.Ops, &c.stats.GetRangeOps)
	b := c.ctr.GetBlobReference(bname)
	rdr, err := b.GetRange(&storage.GetBlobRangeOptions{
		Range: &storage.BlobRange{
			Start: uint64(start),
			End:   uint64(end),
		},
		GetBlobOptions: opts,
	})
	c.stats.TickErr(err)
	return NewCountingReader(rdr, c.stats.TickInBytes), err
}

// If we give it an io.Reader that doesn't also have a Len() int
// method, the Azure SDK determines data size by copying the data into
// a new buffer, which is not a good use of memory.
type readerWithAzureLen struct {
	io.Reader
	len int
}

// Len satisfies the private lener interface in azure-sdk-for-go.
func (r *readerWithAzureLen) Len() int {
	return r.len
}

func (c *azureContainer) CreateBlockBlobFromReader(bname string, size int, rdr io.Reader, opts *storage.PutBlobOptions) error {
	c.stats.TickOps("create")
	c.stats.Tick(&c.stats.Ops, &c.stats.CreateOps)
	if size != 0 {
		rdr = &readerWithAzureLen{
			Reader: NewCountingReader(rdr, c.stats.TickOutBytes),
			len:    size,
		}
	}
	b := c.ctr.GetBlobReference(bname)
	err := b.CreateBlockBlobFromReader(rdr, opts)
	c.stats.TickErr(err)
	return err
}

func (c *azureContainer) SetBlobMetadata(bname string, m storage.BlobMetadata, opts *storage.SetBlobMetadataOptions) error {
	c.stats.TickOps("set_metadata")
	c.stats.Tick(&c.stats.Ops, &c.stats.SetMetadataOps)
	b := c.ctr.GetBlobReference(bname)
	b.Metadata = m
	err := b.SetMetadata(opts)
	c.stats.TickErr(err)
	return err
}

func (c *azureContainer) ListBlobs(params storage.ListBlobsParameters) (storage.BlobListResponse, error) {
	c.stats.TickOps("list")
	c.stats.Tick(&c.stats.Ops, &c.stats.ListOps)
	resp, err := c.ctr.ListBlobs(params)
	c.stats.TickErr(err)
	return resp, err
}

func (c *azureContainer) DeleteBlob(bname string, opts *storage.DeleteBlobOptions) error {
	c.stats.TickOps("delete")
	c.stats.Tick(&c.stats.Ops, &c.stats.DelOps)
	b := c.ctr.GetBlobReference(bname)
	err := b.Delete(opts)
	c.stats.TickErr(err)
	return err
}
