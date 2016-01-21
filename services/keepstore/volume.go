package main

import (
	"io"
	"sync/atomic"
	"time"
)

// A Volume is an interface representing a Keep back-end storage unit:
// for example, a single mounted disk, a RAID array, an Amazon S3 volume,
// etc.
type Volume interface {
	// Get a block. IFF the returned error is nil, the caller must
	// put the returned slice back into the buffer pool when it's
	// finished with it. (Otherwise, the buffer pool will be
	// depleted and eventually -- when all available buffers are
	// used and not returned -- operations will reach deadlock.)
	//
	// loc is guaranteed to consist of 32 or more lowercase hex
	// digits.
	//
	// Get should not verify the integrity of the returned data:
	// it should just return whatever was found in its backing
	// store. (Integrity checking is the caller's responsibility.)
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
	// If the data in the backing store is bigger than BlockSize,
	// Get is permitted to return an error without reading any of
	// the data.
	Get(loc string) ([]byte, error)

	// Compare the given data with the stored data (i.e., what Get
	// would return). If equal, return nil. If not, return
	// CollisionError or DiskHashError (depending on whether the
	// data on disk matches the expected hash), or whatever error
	// was encountered opening/reading the stored data.
	Compare(loc string, data []byte) error

	// Put writes a block to an underlying storage device.
	//
	// loc is as described in Get.
	//
	// len(block) is guaranteed to be between 0 and BlockSize.
	//
	// If a block is already stored under the same name (loc) with
	// different content, Put must either overwrite the existing
	// data with the new data or return a non-nil error. When
	// overwriting existing data, it must never leave the storage
	// device in an inconsistent state: a subsequent call to Get
	// must return either the entire old block, the entire new
	// block, or an error. (An implementation that cannot peform
	// atomic updates must leave the old data alone and return an
	// error.)
	//
	// Put also sets the timestamp for the given locator to the
	// current time.
	//
	// Put must return a non-nil error unless it can guarantee
	// that the entire block has been written and flushed to
	// persistent storage, and that its timestamp is current. Of
	// course, this guarantee is only as good as the underlying
	// storage device, but it is Put's responsibility to at least
	// get whatever guarantee is offered by the storage device.
	//
	// Put should not verify that loc==hash(block): this is the
	// caller's responsibility.
	Put(loc string, block []byte) error

	// Touch sets the timestamp for the given locator to the
	// current time.
	//
	// loc is as described in Get.
	//
	// If invoked at time t0, Touch must guarantee that a
	// subsequent call to Mtime will return a timestamp no older
	// than {t0 minus one second}. For example, if Touch is called
	// at 2015-07-07T01:23:45.67890123Z, it is acceptable for a
	// subsequent Mtime to return any of the following:
	//
	//   - 2015-07-07T01:23:45.00000000Z
	//   - 2015-07-07T01:23:45.67890123Z
	//   - 2015-07-07T01:23:46.67890123Z
	//   - 2015-07-08T00:00:00.00000000Z
	//
	// It is not acceptable for a subsequente Mtime to return
	// either of the following:
	//
	//   - 2015-07-07T00:00:00.00000000Z -- ERROR
	//   - 2015-07-07T01:23:44.00000000Z -- ERROR
	//
	// Touch must return a non-nil error if the timestamp cannot
	// be updated.
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

	// Trash moves the block data from the underlying storage
	// device to trash area. The block then stays in trash for
	// -trash-lifetime interval before it is actually deleted.
	//
	// loc is as described in Get.
	//
	// If the timestamp for the given locator is newer than
	// blobSignatureTTL, Trash must not trash the data.
	//
	// If a Trash operation overlaps with any Touch or Put
	// operations on the same locator, the implementation must
	// ensure one of the following outcomes:
	//
	//   - Touch and Put return a non-nil error, or
	//   - Trash does not trash the block, or
	//   - Both of the above.
	//
	// If it is possible for the storage device to be accessed by
	// a different process or host, the synchronization mechanism
	// should also guard against races with other processes and
	// hosts. If such a mechanism is not available, there must be
	// a mechanism for detecting unsafe configurations, alerting
	// the operator, and aborting or falling back to a read-only
	// state. In other words, running multiple keepstore processes
	// with the same underlying storage device must either work
	// reliably or fail outright.
	//
	// Corollary: A successful Touch or Put guarantees a block
	// will not be trashed for at least blobSignatureTTL
	// seconds.
	Trash(loc string) error

	// Untrash moves block from trash back into store
	Untrash(loc string) error

	// Status returns a *VolumeStatus representing the current
	// in-use and available storage capacity and an
	// implementation-specific volume identifier (e.g., "mount
	// point" for a UnixVolume).
	Status() *VolumeStatus

	// String returns an identifying label for this volume,
	// suitable for including in log messages. It should contain
	// enough information to uniquely identify the underlying
	// storage device, but should not contain any credentials or
	// secrets.
	String() string

	// Writable returns false if all future Put, Mtime, and Delete
	// calls are expected to fail.
	//
	// If the volume is only temporarily unwritable -- or if Put
	// will fail because it is full, but Mtime or Delete can
	// succeed -- then Writable should return false.
	Writable() bool

	// Replication returns the storage redundancy of the
	// underlying device. It will be passed on to clients in
	// responses to PUT requests.
	Replication() int
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

// RRVolumeManager is a round-robin VolumeManager: the Nth call to
// NextWritable returns the (N % len(writables))th writable Volume
// (where writables are all Volumes v where v.Writable()==true).
type RRVolumeManager struct {
	readables []Volume
	writables []Volume
	counter   uint32
}

// MakeRRVolumeManager initializes RRVolumeManager
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

// AllReadable returns an array of all readable volumes
func (vm *RRVolumeManager) AllReadable() []Volume {
	return vm.readables
}

// AllWritable returns an array of all writable volumes
func (vm *RRVolumeManager) AllWritable() []Volume {
	return vm.writables
}

// NextWritable returns the next writable
func (vm *RRVolumeManager) NextWritable() Volume {
	if len(vm.writables) == 0 {
		return nil
	}
	i := atomic.AddUint32(&vm.counter, 1)
	return vm.writables[i%uint32(len(vm.writables))]
}

// Close the RRVolumeManager
func (vm *RRVolumeManager) Close() {
}

// VolumeStatus provides status information of the volume consisting of:
//   * mount_point
//   * device_num (an integer identifying the underlying storage system)
//   * bytes_free
//   * bytes_used
type VolumeStatus struct {
	MountPoint string `json:"mount_point"`
	DeviceNum  uint64 `json:"device_num"`
	BytesFree  uint64 `json:"bytes_free"`
	BytesUsed  uint64 `json:"bytes_used"`
}
