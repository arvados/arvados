// A Volume is an interface representing a Keep back-end storage unit:
// for example, a single mounted disk, a RAID array, an Amazon S3 volume,
// etc.

package main

import (
	"errors"
	"fmt"
	"strings"
)

type Volume interface {
	Get(loc string) ([]byte, error)
	Put(loc string, block []byte) error
	Index(prefix string) string
	Status() *VolumeStatus
	String() string
}

// MockVolumes are Volumes used to test the Keep front end.
//
// If the Bad field is true, this volume should return an error
// on all writes and puts.
//
type MockVolume struct {
	Store map[string][]byte
	Bad   bool
}

func CreateMockVolume() *MockVolume {
	return &MockVolume{make(map[string][]byte), false}
}

func (v *MockVolume) Get(loc string) ([]byte, error) {
	if v.Bad {
		return nil, errors.New("Bad volume")
	} else if block, ok := v.Store[loc]; ok {
		return block, nil
	}
	return nil, errors.New("not found")
}

func (v *MockVolume) Put(loc string, block []byte) error {
	if v.Bad {
		return errors.New("Bad volume")
	}
	v.Store[loc] = block
	return nil
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
