package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

var (
	azureStorageAccountName    string
	azureStorageAccountKeyFile string
)

type azureVolumeAdder struct {
	*volumeSet
}

func (s *azureVolumeAdder) Set(containerName string) error {
	if containerName == "" {
		return errors.New("no container name given")
	}
	buf, err := ioutil.ReadFile(azureStorageAccountKeyFile)
	if err != nil {
		return errors.New("reading key from " + azureStorageAccountKeyFile + ": " + err.Error())
	}
	accountKey := strings.TrimSpace(string(buf))
	if accountKey == "" {
		return errors.New("empty account key in " + azureStorageAccountKeyFile)
	}
	azClient, err := storage.NewBasicClient(azureStorageAccountName, accountKey)
	if err != nil {
		return errors.New("creating Azure storage client: " + err.Error())
	}
	if flagSerializeIO {
		log.Print("Notice: -serialize is not supported by azure-blob-container volumes.")
	}
	*s.volumeSet = append(*s.volumeSet, NewAzureBlobVolume(azClient, containerName, flagReadonly))
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
}

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
	return fmt.Sprintf("azure-storage-container:%+q", v.containerName)
}

func (v *AzureBlobVolume) Writable() bool {
	return !v.readonly
}
