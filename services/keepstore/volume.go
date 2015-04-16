// A Volume is an interface representing a Keep back-end storage unit:
// for example, a single mounted disk, a RAID array, an Amazon S3 volume,
// etc.

package main

import (
	"sync/atomic"
	"time"
)

type Volume interface {
	Get(loc string) ([]byte, error)
	Put(loc string, block []byte) error
	Touch(loc string) error
	Mtime(loc string) (time.Time, error)
	Index(prefix string) string
	Delete(loc string) error
	Status() *VolumeStatus
	String() string
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
	return vm.writables[i % uint32(len(vm.writables))]
}

func (vm *RRVolumeManager) Close() {
}
