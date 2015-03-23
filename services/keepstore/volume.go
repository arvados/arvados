// A Volume is an interface representing a Keep back-end storage unit:
// for example, a single mounted disk, a RAID array, an Amazon S3 volume,
// etc.

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
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
}

// MockVolumes are Volumes used to test the Keep front end.
//
// If the Bad field is true, this volume should return an error
// on all writes and puts.
//
// The Touchable field signifies whether the Touch method will
// succeed.  Defaults to true.  Note that Bad and Touchable are
// independent: a MockVolume may be set up so that Put fails but Touch
// works or vice versa.
//
// TODO(twp): rename Bad to something more descriptive, e.g. Writable,
// and make sure that the tests that rely on it are testing the right
// thing.  We may need to simulate Writable, Touchable and Corrupt
// volumes in different ways.
//
type MockVolume struct {
	Store      map[string][]byte
	Timestamps map[string]time.Time
	Bad        bool
	Touchable  bool
}

func CreateMockVolume() *MockVolume {
	return &MockVolume{
		Store:      make(map[string][]byte),
		Timestamps: make(map[string]time.Time),
		Bad:        false,
		Touchable:  true,
	}
}

func (v *MockVolume) Get(loc string) ([]byte, error) {
	if v.Bad {
		return nil, errors.New("Bad volume")
	} else if block, ok := v.Store[loc]; ok {
		return block, nil
	}
	return nil, os.ErrNotExist
}

func (v *MockVolume) Put(loc string, block []byte) error {
	if v.Bad {
		return errors.New("Bad volume")
	}
	v.Store[loc] = block
	return v.Touch(loc)
}

func (v *MockVolume) Touch(loc string) error {
	if v.Touchable {
		v.Timestamps[loc] = time.Now()
		return nil
	}
	return errors.New("Touch failed")
}

func (v *MockVolume) Mtime(loc string) (time.Time, error) {
	var mtime time.Time
	var err error
	if v.Bad {
		err = errors.New("Bad volume")
	} else if t, ok := v.Timestamps[loc]; ok {
		mtime = t
	} else {
		err = os.ErrNotExist
	}
	return mtime, err
}

func (v *MockVolume) Index(prefix string) string {
	var result string
	for loc, block := range v.Store {
		if IsValidLocator(loc) && strings.HasPrefix(loc, prefix) {
			result = result + fmt.Sprintf("%s+%d %d\n",
				loc, len(block), 123456789)
		}
	}
	return result
}

func (v *MockVolume) Delete(loc string) error {
	if _, ok := v.Store[loc]; ok {
		if time.Since(v.Timestamps[loc]) < permission_ttl {
			return nil
		}
		delete(v.Store, loc)
		return nil
	}
	return os.ErrNotExist
}

func (v *MockVolume) Status() *VolumeStatus {
	var used uint64
	for _, block := range v.Store {
		used = used + uint64(len(block))
	}
	return &VolumeStatus{"/bogo", 123, 1000000 - used, used}
}

func (v *MockVolume) String() string {
	return "[MockVolume]"
}

// A VolumeManager manages a collection of volumes.
//
// - Volumes is a slice of available Volumes.
// - Choose() returns a Volume suitable for writing to.
// - Quit() instructs the VolumeManager to shut down gracefully.
//
type VolumeManager interface {
	Volumes() []Volume
	Choose() Volume
	Quit()
}

type RRVolumeManager struct {
	volumes   []Volume
	nextwrite chan Volume
	quit      chan int
}

func MakeRRVolumeManager(vols []Volume) *RRVolumeManager {
	// Create a new VolumeManager struct with the specified volumes,
	// and with new Nextwrite and Quit channels.
	// The Quit channel is buffered with a capacity of 1 so that
	// another routine may write to it without blocking.
	vm := &RRVolumeManager{vols, make(chan Volume), make(chan int, 1)}

	// This goroutine implements round-robin volume selection.
	// It sends each available Volume in turn to the Nextwrite
	// channel, until receiving a notification on the Quit channel
	// that it should terminate.
	go func() {
		var i int = 0
		for {
			select {
			case <-vm.quit:
				return
			case vm.nextwrite <- vm.volumes[i]:
				i = (i + 1) % len(vm.volumes)
			}
		}
	}()

	return vm
}

func (vm *RRVolumeManager) Volumes() []Volume {
	return vm.volumes
}

func (vm *RRVolumeManager) Choose() Volume {
	return <-vm.nextwrite
}

func (vm *RRVolumeManager) Quit() {
	vm.quit <- 1
}
