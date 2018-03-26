// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type BlockWriter interface {
	// WriteBlock reads all data from r, writes it to a backing
	// store as "loc", and returns the number of bytes written.
	WriteBlock(ctx context.Context, loc string, r io.Reader) error
}

type BlockReader interface {
	// ReadBlock retrieves data previously stored as "loc" and
	// writes it to w.
	ReadBlock(ctx context.Context, loc string, w io.Writer) error
}

// A Volume is an interface representing a Keep back-end storage unit:
// for example, a single mounted disk, a RAID array, an Amazon S3 volume,
// etc.
type Volume interface {
	// Volume type as specified in config file. Examples: "S3",
	// "Directory".
	Type() string

	// Do whatever private setup tasks and configuration checks
	// are needed. Return non-nil if the volume is unusable (e.g.,
	// invalid config).
	Start() error

	// Get a block: copy the block data into buf, and return the
	// number of bytes copied.
	//
	// loc is guaranteed to consist of 32 or more lowercase hex
	// digits.
	//
	// Get should not verify the integrity of the data: it should
	// just return whatever was found in its backing
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
	// If the data in the backing store is bigger than len(buf),
	// then Get is permitted to return an error without reading
	// any of the data.
	//
	// len(buf) will not exceed BlockSize.
	Get(ctx context.Context, loc string, buf []byte) (int, error)

	// Compare the given data with the stored data (i.e., what Get
	// would return). If equal, return nil. If not, return
	// CollisionError or DiskHashError (depending on whether the
	// data on disk matches the expected hash), or whatever error
	// was encountered opening/reading the stored data.
	Compare(ctx context.Context, loc string, data []byte) error

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
	Put(ctx context.Context, loc string, block []byte) error

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
	// BlobSignatureTTL, Trash must not trash the data.
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
	// will not be trashed for at least BlobSignatureTTL
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

	// EmptyTrash looks for trashed blocks that exceeded TrashLifetime
	// and deletes them from the volume.
	EmptyTrash()

	// Return a globally unique ID of the underlying storage
	// device if possible, otherwise "".
	DeviceID() string

	// Get the storage classes associated with this volume
	GetStorageClasses() []string
}

// A VolumeWithExamples provides example configs to display in the
// -help message.
type VolumeWithExamples interface {
	Volume
	Examples() []Volume
}

// A VolumeManager tells callers which volumes can read, which volumes
// can write, and on which volume the next write should be attempted.
type VolumeManager interface {
	// Mounts returns all mounts (volume attachments).
	Mounts() []*VolumeMount

	// Lookup returns the volume under the given mount
	// UUID. Returns nil if the mount does not exist. If
	// write==true, returns nil if the volume is not writable.
	Lookup(uuid string, write bool) Volume

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

	// VolumeStats returns the ioStats used for tracking stats for
	// the given Volume.
	VolumeStats(Volume) *ioStats

	// Close shuts down the volume manager cleanly.
	Close()
}

// A VolumeMount is an attachment of a Volume to a VolumeManager.
type VolumeMount struct {
	arvados.KeepMount
	volume Volume
}

// Generate a UUID the way API server would for a "KeepVolumeMount"
// object.
func (*VolumeMount) generateUUID() string {
	var max big.Int
	_, ok := max.SetString("zzzzzzzzzzzzzzz", 36)
	if !ok {
		panic("big.Int parse failed")
	}
	r, err := rand.Int(rand.Reader, &max)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("zzzzz-ivpuk-%015s", r.Text(36))
}

// RRVolumeManager is a round-robin VolumeManager: the Nth call to
// NextWritable returns the (N % len(writables))th writable Volume
// (where writables are all Volumes v where v.Writable()==true).
type RRVolumeManager struct {
	mounts    []*VolumeMount
	mountMap  map[string]*VolumeMount
	readables []Volume
	writables []Volume
	counter   uint32
	iostats   map[Volume]*ioStats
}

// MakeRRVolumeManager initializes RRVolumeManager
func MakeRRVolumeManager(volumes []Volume) *RRVolumeManager {
	vm := &RRVolumeManager{
		iostats: make(map[Volume]*ioStats),
	}
	vm.mountMap = make(map[string]*VolumeMount)
	for _, v := range volumes {
		sc := v.GetStorageClasses()
		if len(sc) == 0 {
			sc = []string{"default"}
		}
		mnt := &VolumeMount{
			KeepMount: arvados.KeepMount{
				UUID:           (*VolumeMount)(nil).generateUUID(),
				DeviceID:       v.DeviceID(),
				ReadOnly:       !v.Writable(),
				Replication:    v.Replication(),
				StorageClasses: sc,
			},
			volume: v,
		}
		vm.iostats[v] = &ioStats{}
		vm.mounts = append(vm.mounts, mnt)
		vm.mountMap[mnt.UUID] = mnt
		vm.readables = append(vm.readables, v)
		if v.Writable() {
			vm.writables = append(vm.writables, v)
		}
	}
	return vm
}

func (vm *RRVolumeManager) Mounts() []*VolumeMount {
	return vm.mounts
}

func (vm *RRVolumeManager) Lookup(uuid string, needWrite bool) Volume {
	if mnt, ok := vm.mountMap[uuid]; ok && (!needWrite || !mnt.ReadOnly) {
		return mnt.volume
	} else {
		return nil
	}
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

// VolumeStats returns an ioStats for the given volume.
func (vm *RRVolumeManager) VolumeStats(v Volume) *ioStats {
	return vm.iostats[v]
}

// Close the RRVolumeManager
func (vm *RRVolumeManager) Close() {
}

// VolumeStatus describes the current condition of a volume
type VolumeStatus struct {
	MountPoint string
	DeviceNum  uint64
	BytesFree  uint64
	BytesUsed  uint64
}

// ioStats tracks I/O statistics for a volume or server
type ioStats struct {
	Errors     uint64
	Ops        uint64
	CompareOps uint64
	GetOps     uint64
	PutOps     uint64
	TouchOps   uint64
	InBytes    uint64
	OutBytes   uint64
}

type InternalStatser interface {
	InternalStats() interface{}
}
