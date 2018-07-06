// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
/*******************************************************************************
 * Copyright (c) 2018 Genome Research Ltd.
 *
 * Author: Joshua C. Randall <jcrandall@alum.mit.edu>
 *
 * This file is part of Arvados.
 *
 * Arvados is free software: you can redistribute it and/or modify it under
 * the terms of the GNU Affero General Public License as published by the Free
 * Software Foundation; either version 3 of the License, or (at your option) any
 * later version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
 * FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more
 * details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 ******************************************************************************/

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"sync"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
	check "gopkg.in/check.v1"
)

const (
	RadosMockPool      = "mocktestpool"
	RadosMockMonHost   = "mocktestmonhost"
	RadosMockTotalSize = 1 * 1024 * 1024 * 1024
	RadosMockFSID      = "mockmock-mock-mock-mock-mockmockmock"
)

var radosTestPool string
var RadosMockPools []string

func init() {
	flag.StringVar(
		&radosTestPool,
		"test.rados-pool-volume",
		"",
		"Rados pool to use for testing (i.e. to test against a 'real' Ceph cluster such as a ceph/demo docker container). Do not use a pool with real data for testing! Use normal rados volume arguments (e.g. -rados-mon-host, -rados-user, -rados-keyring-file) to supply required parameters to access the pool.")

	RadosMockPools = []string{"mocktestpool0", "mocktestpool1", "mocktestpool2", "mocktestpool3", "mocktestpool4", RadosMockPool}
}

type radosStubObj struct {
	data           []byte
	xattrs         map[string][]byte
	exclusiveLocks map[string]string
	sharedLocks    map[string]map[string]bool
}

func newRadosStubObj(data []byte) *radosStubObj {
	return &radosStubObj{
		data:           data,
		xattrs:         make(map[string][]byte),
		exclusiveLocks: make(map[string]string),
		sharedLocks:    make(map[string]map[string]bool),
	}
}

type radosStubNamespace struct {
	objects map[string]*radosStubObj
}

func newRadosStubNamespace() *radosStubNamespace {
	return &radosStubNamespace{
		objects: make(map[string]*radosStubObj),
	}
}

type radosStubPool struct {
	namespaces map[string]*radosStubNamespace
}

func newRadosStubPool() *radosStubPool {
	return &radosStubPool{
		namespaces: make(map[string]*radosStubNamespace),
	}
}

type radosStubBackend struct {
	sync.Mutex
	config      map[string]string
	totalSize   uint64
	numReplicas uint64
	fsid        string
	pools       map[string]*radosStubPool
	race        chan chan struct{}
}

func newRadosStubBackend(numReplicas uint64) *radosStubBackend {
	return &radosStubBackend{
		config:      make(map[string]string),
		totalSize:   RadosMockTotalSize,
		numReplicas: numReplicas,
		fsid:        "00000000-0000-0000-0000-000000000000",
		pools:       make(map[string]*radosStubPool),
	}
}

func (h *radosStubBackend) unlockAndRace() {
	radosTracef("rados stub: unlockAndRace()")
	if h.race == nil {
		radosTracef("rados stub: unlockAndRace() race is nil, returning")
		return
	}

	radosTracef("rados stub: unlockAndRace() unlocking backend")
	h.Unlock()

	// Signal caller that race is starting by reading from
	// h.race. If we get a channel, block until that channel is
	// ready to receive. If we get nil (or h.race is closed) just
	// proceed.
	radosTracef("rados stub: unlockAndRace() reading from h.race")
	c := <-h.race
	radosTracef("rados stub: unlockAndRace() read from h.race")
	if c != nil {
		radosTracef("rados stub: unlockAndRace() blocking while waiting to write to channel received on h.race")
		c <- struct{}{}
	}

	radosTracef("rados stub: unlockAndRace() locking backend")
	h.Lock()

	radosTracef("rados stub: unlockAndRace() completed, returning")
}

type TestableRadosVolume struct {
	*RadosVolume
	radosStubBackend *radosStubBackend
	t                TB
	useMock          bool
}

func NewTestableRadosVolume(t TB, readonly bool, replication int) *TestableRadosVolume {
	var tv *TestableRadosVolume
	radosTracef("radostest: NewTestableRadosVolume readonly=%v replication=%d", readonly, replication)
	radosStubBackend := newRadosStubBackend(uint64(replication))
	pool := radosTestPool
	useMock := pool == ""

	if useMock {
		// Connect using mock radosImplementation instead of real Ceph
		log.Infof("radostest: using mock radosImplementation")
		radosMock := &radosMockImpl{
			b: radosStubBackend,
		}
		v := &RadosVolume{
			Pool:              RadosMockPool,
			MonHost:           RadosMockMonHost,
			ReadOnly:          readonly,
			RadosReplication:  replication,
			RadosIndexWorkers: 4,
			ReadTimeout:       arvados.Duration(10 * time.Second),
			WriteTimeout:      arvados.Duration(10 * time.Second),
			MetadataTimeout:   arvados.Duration(10 * time.Second),
			rados:             radosMock,
		}
		tv = &TestableRadosVolume{
			RadosVolume:      v,
			radosStubBackend: radosStubBackend,
			t:                t,
			useMock:          useMock,
		}
	} else {
		// Connect to real Ceph using the real radosImplementation
		log.Infof("radostest: using real radosImplementation")
		v := &RadosVolume{
			Pool:              pool,
			KeyringFile:       radosKeyringFile,
			MonHost:           radosMonHost,
			Cluster:           radosCluster,
			User:              radosUser,
			ReadOnly:          readonly,
			RadosReplication:  replication,
			RadosIndexWorkers: 4,
			ReadTimeout:       arvados.Duration(DefaultRadosReadTimeoutSeconds * time.Second),
			WriteTimeout:      arvados.Duration(DefaultRadosWriteTimeoutSeconds * time.Second),
			MetadataTimeout:   arvados.Duration(DefaultRadosMetadataTimeoutSeconds * time.Second),
		}
		tv = &TestableRadosVolume{
			RadosVolume: v,
			t:           t,
			useMock:     useMock,
		}
	}

	// Start
	err := tv.Start()
	if err != nil {
		t.Error(err)
	}

	return tv
}

var _ = check.Suite(&StubbedRadosSuite{})

type StubbedRadosSuite struct {
	volume *TestableRadosVolume
}

func (s *StubbedRadosSuite) SetUpTest(c *check.C) {
	s.volume = NewTestableRadosVolume(c, false, 3)
}

func (s *StubbedRadosSuite) TearDownTest(c *check.C) {
	s.volume.Teardown()
}

// Rados Volume Tests
func TestRadosVolumeWithGeneric(t *testing.T) {
	DoGenericVolumeTests(t, func(t TB) TestableVolume {
		return NewTestableRadosVolume(t, false, radosReplication)
	})
}

func TestRadosReadonlyVolumeWithGeneric(t *testing.T) {
	DoGenericVolumeTests(t, func(t TB) TestableVolume {
		return NewTestableRadosVolume(t, true, radosReplication)
	})
}

func TestRadosVolumeReplication(t *testing.T) {
	for r := 1; r <= 4; r++ {
		v := NewTestableRadosVolume(t, false, r)
		defer v.Teardown()
		if n := v.Replication(); n != r {
			t.Errorf("Got replication %d, expected %d", n, r)
		}
	}
}

func TestRadosVolumeContextCancelGet(t *testing.T) {
	testRadosVolumeContextCancel(t, func(ctx context.Context, v *TestableRadosVolume) error {
		v.PutRaw(TestHash, TestBlock)
		_, err := v.Get(ctx, TestHash, make([]byte, BlockSize))
		return err
	})
}

func TestRadosVolumeContextCancelPut(t *testing.T) {
	testRadosVolumeContextCancel(t, func(ctx context.Context, v *TestableRadosVolume) error {
		return v.Put(ctx, TestHash, make([]byte, BlockSize))
	})
}

func TestRadosVolumeContextCancelCompare(t *testing.T) {
	testRadosVolumeContextCancel(t, func(ctx context.Context, v *TestableRadosVolume) error {
		v.PutRaw(TestHash, TestBlock)
		return v.Compare(ctx, TestHash, TestBlock2)
	})
}

func testRadosVolumeContextCancel(t *testing.T, testFunc func(context.Context, *TestableRadosVolume) error) {
	v := NewTestableRadosVolume(t, false, 3)
	defer v.Teardown()

	if v.radosStubBackend == nil {
		t.Skip("radostest: testRadosVolumeContextCancel can only be run with radosStubBackend")
	}
	v.radosStubBackend.race = make(chan chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	allDone := make(chan struct{})
	testFuncErr := make(chan error, 1)
	go func() {
		defer close(allDone)
		defer close(testFuncErr)
		err := testFunc(ctx, v)
		if err != context.Canceled {
			err = fmt.Errorf("radostest: testRadosVolumeContextCancel testFunc returned %T %q, expected %q", err, err, context.Canceled)
			testFuncErr <- err
		}
	}()
	releaseHandler := make(chan struct{})
	select {
	case <-allDone:
		t.Error("radostest: testRadosVolumeContextCancel testFunc finished without waiting for v.radosStubBackend.race")
	case <-time.After(10 * time.Second):
		t.Error("radostest: testRadosVolumeContextCancel timed out waiting to enter handler")
	case v.radosStubBackend.race <- releaseHandler:
	}

	radosTracef("radostest: testRadosVolumeContextCancel cancelling context")
	cancel()

	select {
	case <-time.After(10 * time.Second):
		t.Error("radostest: testRadosVolumeContextCancel timed out waiting to cancel")
	case <-allDone:
	}

	err := <-testFuncErr
	if err != nil {
		t.Errorf("radostest: testRadosVolumeContextCancel error from testFunc: %v", err)
	}

	go func() {
		radosTracef("radostest: testRadosVolumeContextCancel receiving from releaseHandler to release the backend from the race")
		<-releaseHandler
	}()
}

func (s *StubbedRadosSuite) TestStats(c *check.C) {
	stats := func() string {
		buf, err := json.Marshal(s.volume.InternalStats())
		c.Check(err, check.IsNil)
		return string(buf)
	}

	c.Check(stats(), check.Matches, `.*"Ops":0,.*`)
	c.Check(stats(), check.Matches, `.*"Errors":0,.*`)

	loc := "acbd18db4cc2f85cedef654fccc4a4d8"
	_, err := s.volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.NotNil)
	c.Check(stats(), check.Matches, `.*"Ops":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"Errors":[^0],.*`)
	c.Check(stats(), check.Matches, `.*"rados\.RadosErrorNotFound.*?":[^0].*`)
	c.Check(stats(), check.Matches, `.*"InBytes":0,.*`)

	err = s.volume.Put(context.Background(), loc, []byte("foo"))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"OutBytes":3,.*`)
	c.Check(stats(), check.Matches, `.*"CreateOps":1,.*`)

	_, err = s.volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.IsNil)
	_, err = s.volume.Get(context.Background(), loc, make([]byte, 3))
	c.Check(err, check.IsNil)
	c.Check(stats(), check.Matches, `.*"InBytes":6,.*`)
}

func (s *StubbedRadosSuite) TestConfig(c *check.C) {
	var cfg Config
	err := yaml.Unmarshal([]byte(`
Volumes:
  - Type: Rados
    StorageClasses: ["class_a", "class_b"]
`), &cfg)

	c.Check(err, check.IsNil)
	c.Check(cfg.Volumes[0].GetStorageClasses(), check.DeepEquals, []string{"class_a", "class_b"})
}

func (v *TestableRadosVolume) PutRaw(loc string, data []byte) {
	radosTracef("radostest: PutRaw loc=%s len(data)=%d data='%s'", loc, len(data), data)

	if v.ReadOnly {
		// need to temporarily disable ReadOnly status and restore it after the call to Put
		defer func(ro bool) {
			v.ReadOnly = ro
		}(v.ReadOnly)
		v.ReadOnly = false
	}

	if v.radosStubBackend != nil && v.radosStubBackend.race != nil {
		// also need to temporarily disable backend race
		defer func(race chan chan struct{}) {
			v.radosStubBackend.race = race
		}(v.radosStubBackend.race)
		v.radosStubBackend.race = nil
	}

	err := v.Put(context.Background(), loc, data)
	if err != nil {
		v.t.Fatalf("radostest: PutRaw failed to put loc %s: %s", loc, err)
	}

	radosTracef("radostest: PutRaw loc=%s len(data)=%d data='%s' complete, returning", loc, len(data), data)
	return
}

func (v *TestableRadosVolume) TouchWithDate(loc string, mtime time.Time) {
	radosTracef("radostest: TouchWithDate loc=%s mtime=%v", loc, mtime)
	err := v.setMtime(loc, mtime)
	if err != nil {
		radosTracef("radostest: TouchWithDate loc=%s mtime=%v setMtime returned err=%v", loc, mtime, err)
		v.t.Fatalf("TouchWithDate failed to set mtime for block %s", loc)
	}
	radosTracef("radostest: TouchWithDate loc=%s mtime=%v complete, returning.", loc, mtime)
	return
}

func (v *TestableRadosVolume) Teardown() {
	if !v.useMock {
		// When using a real Ceph pool we need to clean out all data
		// after each test.
		err := v.deleteAllObjects()
		if err != nil {
			v.t.Error(err)
		}
	}
	// we also must call conn.Shutdown or else librados will leak threads like crazy every time we abandon a RadosVolume and create a new one
	v.conn.Shutdown()
}

type errListEntry struct {
	err error
}

func (ile *errListEntry) String() string {
	return fmt.Sprintf("%s", ile.err)
}

func (ile *errListEntry) Err() error {
	return ile.err
}

func (v *TestableRadosVolume) deleteAllObjects() (err error) {
	radosTracef("radostest: deleteAllObjects()")

	// filter to include all objects
	filterFunc := func(loc string) (bool, error) {
		return true, nil
	}

	// delete each loc and return empty listEntry
	mapFunc := func(loc string) listEntry {
		delErr := v.delete(loc)
		if delErr != nil {
			log.Warnf("radostest: deleteAllObjects() failed to delete %s: %v", loc, delErr)
			return &errListEntry{
				err: delErr,
			}
		}
		return &errListEntry{}
	}

	// count number of objects deleted and errors
	deleted := 0
	errors := 0
	reduceFunc := func(le listEntry) {
		if le.Err() != nil {
			errors++
		} else {
			deleted++
		}
	}

	workers := 1
	err = v.listObjects(filterFunc, mapFunc, reduceFunc, workers)
	if err != nil {
		log.Printf("radostest: deleteAllObjects() failed to listObjects: %s", err)
		return
	}
	log.Infof("radostest: deleteAllObjects() deleted %d objects and had %d errors", deleted, errors)

	radosTracef("radostest: deleteAllObjects() finished, returning")
	return
}
