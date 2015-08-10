package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// MockVolumes are test doubles for Volumes, used to test handlers.
type MockVolume struct {
	Store      map[string][]byte
	Timestamps map[string]time.Time
	// Bad volumes return an error for every operation.
	Bad bool
	// Touchable volumes' Touch() method succeeds for a locator
	// that has been Put().
	Touchable bool
	// Readonly volumes return an error for Put, Delete, and
	// Touch.
	Readonly bool
	// Every operation (except Status) starts by receiving from
	// Gate. Send one value to unblock one operation; close the
	// channel to unblock all. By default, it is a closed channel,
	// so all operations proceed without blocking.
	Gate   chan struct{}
	called map[string]int
	mutex  sync.Mutex
}

// CreateMockVolume returns a non-Bad, non-Readonly, Touchable mock
// volume.
func CreateMockVolume() *MockVolume {
	gate := make(chan struct{})
	close(gate)
	return &MockVolume{
		Store:      make(map[string][]byte),
		Timestamps: make(map[string]time.Time),
		Bad:        false,
		Touchable:  true,
		Readonly:   false,
		called:     map[string]int{},
		Gate:       gate,
	}
}

// CallCount returns how many times the named method has been called.
func (v *MockVolume) CallCount(method string) int {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if c, ok := v.called[method]; !ok {
		return 0
	} else {
		return c
	}
}

func (v *MockVolume) gotCall(method string) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if _, ok := v.called[method]; !ok {
		v.called[method] = 1
	} else {
		v.called[method]++
	}
}

func (v *MockVolume) Get(loc string) ([]byte, error) {
	v.gotCall("Get")
	<-v.Gate
	if v.Bad {
		return nil, errors.New("Bad volume")
	} else if block, ok := v.Store[loc]; ok {
		buf := bufs.Get(len(block))
		copy(buf, block)
		return buf, nil
	}
	return nil, os.ErrNotExist
}

func (v *MockVolume) Put(loc string, block []byte) error {
	v.gotCall("Put")
	<-v.Gate
	if v.Bad {
		return errors.New("Bad volume")
	}
	if v.Readonly {
		return MethodDisabledError
	}
	v.Store[loc] = block
	return v.Touch(loc)
}

func (v *MockVolume) Touch(loc string) error {
	v.gotCall("Touch")
	<-v.Gate
	if v.Readonly {
		return MethodDisabledError
	}
	if v.Touchable {
		v.Timestamps[loc] = time.Now()
		return nil
	}
	return errors.New("Touch failed")
}

func (v *MockVolume) Mtime(loc string) (time.Time, error) {
	v.gotCall("Mtime")
	<-v.Gate
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

func (v *MockVolume) IndexTo(prefix string, w io.Writer) error {
	v.gotCall("IndexTo")
	<-v.Gate
	for loc, block := range v.Store {
		if !IsValidLocator(loc) || !strings.HasPrefix(loc, prefix) {
			continue
		}
		_, err := fmt.Fprintf(w, "%s+%d %d\n",
			loc, len(block), 123456789)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *MockVolume) Delete(loc string) error {
	v.gotCall("Delete")
	<-v.Gate
	if v.Readonly {
		return MethodDisabledError
	}
	if _, ok := v.Store[loc]; ok {
		if time.Since(v.Timestamps[loc]) < blob_signature_ttl {
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

func (v *MockVolume) Writable() bool {
	return !v.Readonly
}
