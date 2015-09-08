// A Volume is an interface representing a Keep back-end storage unit:
// for example, a single mounted disk, a RAID array, an Amazon S3 volume,
// etc.

package main

import (
	"io"
	"sync/atomic"
	"time"
)

type Volume interface {
	// Get a block. IFF the returned error is nil, the caller must
	// put the returned slice back into the buffer pool when it's
	// finished with it.
	//
	// loc is guaranteed to consist of 32 or more lowercase hex
	// digits.
	//
	// Get should not verify the integrity of the returned data:
	// it should just return whatever was found in its backing
	// store.
	//
	// If an error is encountered that prevents it from
	// retrieving the data, that error should be returned so the
	// caller can log (and send to the client) a more useful
	// message.
	//
	// If the error is "not found", and there's no particular
	// reason to expect the block to be found (other than that a
	// caller is asking for it), the returned error should satisfy
	// os.IsNotExist(err): this is a normal condition and will not
	// be logged as an error (except that a 404 will appear in the
	// access log if the block is not found on any other volumes
	// either).
	//
	// If the data in the backing store is bigger than BLOCKSIZE,
	// Get is permitted to return an error without reading any of
	// the data.
	Get(loc string) ([]byte, error)

	// Put writes a block to an underlying storage device.
	//
	// loc is as described in Get.
	//
	// len(block) is guaranteed to be between 0 and BLOCKSIZE.
	//
	// If a block is already stored under the same name (loc) with
	// different content, Put must either overwrite the existing
	// data with the new data or return a non-nil error.
	//
	// Put must return a non-nil error unless it can guarantee
	// that the entire block has been written and flushed to
	// persistent storage. Of course, this guarantee is only as
	// good as the underlying storage device, but it is Put's
	// responsibility to at least get whatever guarantee is
	// offered by the storage device.
	//
	// Put should not verify that loc==hash(block): this is the
	// caller's responsibility.
	Put(loc string, block []byte) error

	// Touch sets the timestamp for the given locator to the
	// current time.
	//
	// loc is as described in Get.
	//
	// Touch must return a non-nil error unless it can guarantee
	// that a future call to Mtime() will return a timestamp newer
	// than {now minus one second}.
	Touch(loc string) error

	// Mtime returns the stored timestamp for the given locator.
	//
	// loc is as described in Get.
	//
	// Mtime must return a non-nil error if the given block is not
	// found or the timestamp could not be retrieved.
	Mtime(loc string) (time.Time, error)

	// IndexTo writes a complete list of locators with the given
	// prefix for which Get() can retrieve data.
	//
	// prefix consists of zero or more lowercase hexadecimal
	// digits.
	//
	// Each locator must be written to the given writer using the
	// following format:
	//
	//   loc "+" size " " timestamp "\n"
	//
	// where:
	//
	//   - size is the number of bytes of content, given as a
	//     decimal number with one or more digits
	//     
	//   - timestamp is the timestamp stored for the locator,
	//     given as a decimal number of seconds after January 1,
	//     1970 UTC.
	//
	// IndexTo must not write any other data to writer: for
	// example, it must not write any blank lines.
	//
	// If an error makes it impossible to provide a complete
	// index, IndexTo must return a non-nil error. It is
	// acceptable to return a non-nil error after writing a
	// partial index to writer.
	//
	// The resulting index is not expected to be sorted in any
	// particular order.
	IndexTo(prefix string, writer io.Writer) error

	// Delete deletes the block data from the underlying storage
	// device.
	//
	// loc is as described in Get.
	//
	// If the timestamp for the given locator is newer than
	// blob_signature_ttl, Delete must not delete the data.
	//
	// If callers in different goroutines invoke overlapping
	// Delete() and Touch() operations on the same locator, the
	// implementation must guarantee that Touch() returns a
	// non-nil error, or Delete() does not delete the block, or
	// both.
	Delete(loc string) error

	// Status() returns a *VolumeStatus representing the current
	// in-use and available storage capacity and an
	// implementation-specific volume identifier (e.g., "mount
	// point" for a UnixVolume).
	Status() *VolumeStatus

	// String() returns an identifying label for this volume,
	// suitable for including in log messages. It should contain
	// enough information to uniquely identify the underlying
	// storage device, but should not contain any credentials or
	// secrets.
	String() string

	// Writable() returns false if all future Put(), Mtime(), and
	// Delete() calls are expected to fail.
	//
	// If the volume is only temporarily unwritable -- or if Put()
	// will fail because it is full, but Mtime() or Delete() can
	// succeed -- then Writable() should return false.
	Writable() bool
}

// A VolumeManager tells callers which volumes can read, which volumes
// can write, and on which volume the next write should be attempted.
type VolumeManager interface {
	// AllReadable returns all volumes.
	AllReadable() []Volume
	// AllWritable returns all volumes that aren't known to be in
	// a read-only state. (There is no guarantee that a write to
	// one will succeed, though.)
	AllWritable() []Volume
	// NextWritable returns the volume where the next new block
	// should be written. A VolumeManager can select a volume in
	// order to distribute activity across spindles, fill up disks
	// with more free space, etc.
	NextWritable() Volume
	// Close shuts down the volume manager cleanly.
	Close()
}

type RRVolumeManager struct {
	readables []Volume
	writables []Volume
	counter   uint32
}

func MakeRRVolumeManager(volumes []Volume) *RRVolumeManager {
	vm := &RRVolumeManager{}
	for _, v := range volumes {
		vm.readables = append(vm.readables, v)
		if v.Writable() {
			vm.writables = append(vm.writables, v)
		}
	}
	return vm
}

func (vm *RRVolumeManager) AllReadable() []Volume {
	return vm.readables
}

func (vm *RRVolumeManager) AllWritable() []Volume {
	return vm.writables
}

func (vm *RRVolumeManager) NextWritable() Volume {
	if len(vm.writables) == 0 {
		return nil
	}
	i := atomic.AddUint32(&vm.counter, 1)
	return vm.writables[i%uint32(len(vm.writables))]
}

func (vm *RRVolumeManager) Close() {
}
