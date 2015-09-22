package main

import (
	"fmt"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// An AzureBlobVolume stores and retrieves blocks in an Azure Blob
// container.
type AzureBlobVolume struct {
	azClient      storage.Client
	bsClient      storage.BlobStorageClient
	containerName string
	readonly      bool
}

func NewAzureBlobVolume(client storage.Client, containerName string, readonly bool) *AzureBlobVolume {
	return &AzureBlobVolume{
		azClient: client,
		bsClient: client.GetBlobService(),
		containerName: containerName,
		readonly: readonly,
	}
}

func (v *AzureBlobVolume) Get(loc string) ([]byte, error) {
	rdr, err := v.bsClient.GetBlob(v.containerName, loc)
	if err != nil {
		return nil, err
	}
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

func (v *AzureBlobVolume) Compare(loc string, data []byte) error {
	return NotFoundError
}

func (v *AzureBlobVolume) Put(loc string, block []byte) error {
	return NotFoundError
}

func (v *AzureBlobVolume) Touch(loc string) error {
	return NotFoundError
}

func (v *AzureBlobVolume) Mtime(loc string) (time.Time, error) {
	return time.Time{}, NotFoundError
}

func (v *AzureBlobVolume) IndexTo(prefix string, writer io.Writer) error {
	return nil
}

func (v *AzureBlobVolume) Delete(loc string) error {
	return NotFoundError
}

func (v *AzureBlobVolume) Status() *VolumeStatus {
	return &VolumeStatus{
		DeviceNum: 1,
		BytesFree: BlockSize * 1000,
		BytesUsed: 1,
	}
}

func (v *AzureBlobVolume) String() string {
	return fmt.Sprintf("%+v", v.azClient)
}

func (v *AzureBlobVolume) Writable() bool {
	return !v.readonly
}
