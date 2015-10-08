package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/curoverse/azure-sdk-for-go/storage"
)

var (
	azureStorageAccountName    string
	azureStorageAccountKeyFile string
	azureStorageReplication    int
	azureWriteRaceInterval     time.Duration = 15 * time.Second
	azureWriteRacePollTime     time.Duration = time.Second
)

func readKeyFromFile(file string) (string, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return "", errors.New("reading key from " + file + ": " + err.Error())
	}
	accountKey := strings.TrimSpace(string(buf))
	if accountKey == "" {
		return "", errors.New("empty account key in " + file)
	}
	return accountKey, nil
}

type azureVolumeAdder struct {
	*volumeSet
}

func (s *azureVolumeAdder) Set(containerName string) error {
	if containerName == "" {
		return errors.New("no container name given")
	}
	if azureStorageAccountName == "" || azureStorageAccountKeyFile == "" {
		return errors.New("-azure-storage-account-name and -azure-storage-account-key-file arguments must given before -azure-storage-container-volume")
	}
	accountKey, err := readKeyFromFile(azureStorageAccountKeyFile)
	if err != nil {
		return err
	}
	azClient, err := storage.NewBasicClient(azureStorageAccountName, accountKey)
	if err != nil {
		return errors.New("creating Azure storage client: " + err.Error())
	}
	if flagSerializeIO {
		log.Print("Notice: -serialize is not supported by azure-blob-container volumes.")
	}
	v := NewAzureBlobVolume(azClient, containerName, flagReadonly, azureStorageReplication)
	if err := v.Check(); err != nil {
		return err
	}
	*s.volumeSet = append(*s.volumeSet, v)
	return nil
}

func init() {
	flag.Var(&azureVolumeAdder{&volumes},
		"azure-storage-container-volume",
		"Use the given container as a storage volume. Can be given multiple times.")
	flag.StringVar(
		&azureStorageAccountName,
		"azure-storage-account-name",
		"",
		"Azure storage account name used for subsequent --azure-storage-container-volume arguments.")
	flag.StringVar(
		&azureStorageAccountKeyFile,
		"azure-storage-account-key-file",
		"",
		"File containing the account key used for subsequent --azure-storage-container-volume arguments.")
	flag.IntVar(
		&azureStorageReplication,
		"azure-storage-replication",
		3,
		"Replication level to report to clients when data is stored in an Azure container.")
}

// An AzureBlobVolume stores and retrieves blocks in an Azure Blob
// container.
type AzureBlobVolume struct {
	azClient      storage.Client
	bsClient      storage.BlobStorageClient
	containerName string
	readonly      bool
	replication   int
}

func NewAzureBlobVolume(client storage.Client, containerName string, readonly bool, replication int) *AzureBlobVolume {
	return &AzureBlobVolume{
		azClient:      client,
		bsClient:      client.GetBlobService(),
		containerName: containerName,
		readonly:      readonly,
		replication:   replication,
	}
}

// Check returns nil if the volume is usable.
func (v *AzureBlobVolume) Check() error {
	ok, err := v.bsClient.ContainerExists(v.containerName)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("container does not exist")
	}
	return nil
}

func (v *AzureBlobVolume) Get(loc string) ([]byte, error) {
	var deadline time.Time
	haveDeadline := false
	buf, err := v.get(loc)
	for err == nil && len(buf) == 0 && loc[:32] != "d41d8cd98f00b204e9800998ecf8427e" {
		// Seeing a brand new empty block probably means we're
		// in a race with CreateBlob, which under the hood
		// (apparently) does "CreateEmpty" and "CommitData"
		// with no additional transaction locking.
		if !haveDeadline {
			t, err := v.Mtime(loc)
			if err != nil {
				log.Print("Got empty block (possible race) but Mtime failed: ", err)
				break
			}
			deadline = t.Add(azureWriteRaceInterval)
			haveDeadline = true
		}
		if time.Now().After(deadline) {
			break
		}
		bufs.Put(buf)
		time.Sleep(azureWriteRacePollTime)
		buf, err = v.get(loc)
	}
	return buf, err
}

func (v *AzureBlobVolume) get(loc string) ([]byte, error) {
	rdr, err := v.bsClient.GetBlob(v.containerName, loc)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			// "storage: service returned without a response body (404 Not Found)"
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	switch err := err.(type) {
	case nil:
	default:
		log.Printf("ERROR IN Get(): %T %#v", err, err)
		return nil, err
	}
	defer rdr.Close()
	buf := bufs.Get(BlockSize)
	n, err := io.ReadFull(rdr, buf)
	switch err {
	case io.EOF, io.ErrUnexpectedEOF:
		return buf[:n], nil
	default:
		bufs.Put(buf)
		return nil, err
	}
}

func (v *AzureBlobVolume) Compare(loc string, expect []byte) error {
	rdr, err := v.bsClient.GetBlob(v.containerName, loc)
	if err != nil {
		return err
	}
	defer rdr.Close()
	return compareReaderWithBuf(rdr, expect, loc[:32])
}

func (v *AzureBlobVolume) Put(loc string, block []byte) error {
	if v.readonly {
		return MethodDisabledError
	}
	return v.bsClient.CreateBlockBlobFromReader(v.containerName, loc, uint64(len(block)), bytes.NewReader(block))
}

func (v *AzureBlobVolume) Touch(loc string) error {
	if v.readonly {
		return MethodDisabledError
	}
	return v.bsClient.SetBlobMetadata(v.containerName, loc, map[string]string{
		"touch": fmt.Sprintf("%d", time.Now()),
	})
}

func (v *AzureBlobVolume) Mtime(loc string) (time.Time, error) {
	props, err := v.bsClient.GetBlobProperties(v.containerName, loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC1123, props.LastModified)
}

func (v *AzureBlobVolume) IndexTo(prefix string, writer io.Writer) error {
	params := storage.ListBlobsParameters{
		Prefix: prefix,
	}
	for {
		resp, err := v.bsClient.ListBlobs(v.containerName, params)
		if err != nil {
			return err
		}
		for _, b := range resp.Blobs {
			t, err := time.Parse(time.RFC1123, b.Properties.LastModified)
			if err != nil {
				return err
			}
			if b.Properties.ContentLength == 0 && t.Add(azureWriteRaceInterval).After(time.Now()) {
				// A new zero-length blob is probably
				// just a new non-empty blob that
				// hasn't committed its data yet (see
				// Get()), and in any case has no
				// value.
				continue
			}
			fmt.Fprintf(writer, "%s+%d %d\n", b.Name, b.Properties.ContentLength, t.Unix())
		}
		if resp.NextMarker == "" {
			return nil
		}
		params.Marker = resp.NextMarker
	}
}

func (v *AzureBlobVolume) Delete(loc string) error {
	if v.readonly {
		return MethodDisabledError
	}
	// Ideally we would use If-Unmodified-Since, but that
	// particular condition seems to be ignored by Azure. Instead,
	// we get the Etag before checking Mtime, and use If-Match to
	// ensure we don't delete data if Put() or Touch() happens
	// between our calls to Mtime() and DeleteBlob().
	props, err := v.bsClient.GetBlobProperties(v.containerName, loc)
	if err != nil {
		return err
	}
	if t, err := v.Mtime(loc); err != nil {
		return err
	} else if time.Since(t) < blobSignatureTTL {
		return nil
	}
	return v.bsClient.DeleteBlob(v.containerName, loc, map[string]string{
		"If-Match": props.Etag,
	})
}

func (v *AzureBlobVolume) Status() *VolumeStatus {
	return &VolumeStatus{
		DeviceNum: 1,
		BytesFree: BlockSize * 1000,
		BytesUsed: 1,
	}
}

func (v *AzureBlobVolume) String() string {
	return fmt.Sprintf("azure-storage-container:%+q", v.containerName)
}

func (v *AzureBlobVolume) Writable() bool {
	return !v.readonly
}

func (v *AzureBlobVolume) Replication() int {
	return v.replication
}
