// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

var (
	TestBlock       = []byte("The quick brown fox jumps over the lazy dog.")
	TestHash        = "e4d909c290d0fb1ca068ffaddf22cbd0"
	TestHashPutResp = "e4d909c290d0fb1ca068ffaddf22cbd0+44\n"

	TestBlock2 = []byte("Pack my box with five dozen liquor jugs.")
	TestHash2  = "f15ac516f788aec4f30932ffb6395c39"

	TestBlock3 = []byte("Now is the time for all good men to come to the aid of their country.")
	TestHash3  = "eed29bbffbc2dbe5e5ee0bb71888e61f"

	// BadBlock is used to test collisions and corruption.
	// It must not match any test hashes.
	BadBlock = []byte("The magic words are squeamish ossifrage.")

	EmptyHash  = "d41d8cd98f00b204e9800998ecf8427e"
	EmptyBlock = []byte("")
)

// A TestableVolume allows test suites to manipulate the state of an
// underlying Volume, in order to test behavior in cases that are
// impractical to achieve with a sequence of normal Volume operations.
type TestableVolume interface {
	Volume

	// [Over]write content for a locator with the given data,
	// bypassing all constraints like readonly and serialize.
	PutRaw(locator string, data []byte)

	// Returns the strings that a driver uses to record read/write operations.
	ReadWriteOperationLabelValues() (r, w string)

	// Specify the value Mtime() should return, until the next
	// call to Touch, TouchWithDate, or Put.
	TouchWithDate(locator string, lastPut time.Time)

	// Clean up, delete temporary files.
	Teardown()
}

func init() {
	driver["mock"] = newMockVolume
}

// MockVolumes are test doubles for Volumes, used to test handlers.
type MockVolume struct {
	Store      map[string][]byte
	Timestamps map[string]time.Time

	// Bad volumes return an error for every operation.
	Bad            bool
	BadVolumeError error

	// Touchable volumes' Touch() method succeeds for a locator
	// that has been Put().
	Touchable bool

	// Gate is a "starting gate", allowing test cases to pause
	// volume operations long enough to inspect state. Every
	// operation (except Status) starts by receiving from
	// Gate. Sending one value unblocks one operation; closing the
	// channel unblocks all operations. By default, Gate is a
	// closed channel, so all operations proceed without
	// blocking. See trash_worker_test.go for an example.
	Gate chan struct{} `json:"-"`

	cluster *arvados.Cluster
	volume  arvados.Volume
	logger  logrus.FieldLogger
	metrics *volumeMetricsVecs
	called  map[string]int
	mutex   sync.Mutex
}

// newMockVolume returns a non-Bad, non-Readonly, Touchable mock
// volume.
func newMockVolume(cluster *arvados.Cluster, volume arvados.Volume, logger logrus.FieldLogger, metrics *volumeMetricsVecs) (Volume, error) {
	gate := make(chan struct{})
	close(gate)
	return &MockVolume{
		Store:      make(map[string][]byte),
		Timestamps: make(map[string]time.Time),
		Bad:        false,
		Touchable:  true,
		called:     map[string]int{},
		Gate:       gate,
		cluster:    cluster,
		volume:     volume,
		logger:     logger,
		metrics:    metrics,
	}, nil
}

// CallCount returns how many times the named method has been called.
func (v *MockVolume) CallCount(method string) int {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	c, ok := v.called[method]
	if !ok {
		return 0
	}
	return c
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

func (v *MockVolume) Compare(ctx context.Context, loc string, buf []byte) error {
	v.gotCall("Compare")
	<-v.Gate
	if v.Bad {
		return v.BadVolumeError
	} else if block, ok := v.Store[loc]; ok {
		if fmt.Sprintf("%x", md5.Sum(block)) != loc {
			return DiskHashError
		}
		if bytes.Compare(buf, block) != 0 {
			return CollisionError
		}
		return nil
	} else {
		return os.ErrNotExist
	}
}

func (v *MockVolume) Get(ctx context.Context, loc string, buf []byte) (int, error) {
	v.gotCall("Get")
	<-v.Gate
	if v.Bad {
		return 0, v.BadVolumeError
	} else if block, ok := v.Store[loc]; ok {
		copy(buf[:len(block)], block)
		return len(block), nil
	}
	return 0, os.ErrNotExist
}

func (v *MockVolume) Put(ctx context.Context, loc string, block []byte) error {
	v.gotCall("Put")
	<-v.Gate
	if v.Bad {
		return v.BadVolumeError
	}
	if v.volume.ReadOnly {
		return MethodDisabledError
	}
	v.Store[loc] = block
	return v.Touch(loc)
}

func (v *MockVolume) Touch(loc string) error {
	return v.TouchWithDate(loc, time.Now())
}

func (v *MockVolume) TouchWithDate(loc string, t time.Time) error {
	v.gotCall("Touch")
	<-v.Gate
	if v.volume.ReadOnly {
		return MethodDisabledError
	}
	if _, exists := v.Store[loc]; !exists {
		return os.ErrNotExist
	}
	if v.Touchable {
		v.Timestamps[loc] = t
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
		err = v.BadVolumeError
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

func (v *MockVolume) Trash(loc string) error {
	v.gotCall("Delete")
	<-v.Gate
	if v.volume.ReadOnly {
		return MethodDisabledError
	}
	if _, ok := v.Store[loc]; ok {
		if time.Since(v.Timestamps[loc]) < time.Duration(v.cluster.Collections.BlobSigningTTL) {
			return nil
		}
		delete(v.Store, loc)
		return nil
	}
	return os.ErrNotExist
}

func (v *MockVolume) GetDeviceID() string {
	return "mock-device-id"
}

func (v *MockVolume) Untrash(loc string) error {
	return nil
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

func (v *MockVolume) EmptyTrash() {
}

func (v *MockVolume) GetStorageClasses() []string {
	return nil
}
