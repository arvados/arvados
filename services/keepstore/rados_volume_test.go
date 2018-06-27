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
	"sync"
	"testing"
	"time"

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
	if h.race == nil {
		return
	}
	h.Unlock()
	// Signal caller that race is starting by reading from
	// h.race. If we get a channel, block until that channel is
	// ready to receive. If we get nil (or h.race is closed) just
	// proceed.
	if c := <-h.race; c != nil {
		c <- struct{}{}
	}
	h.Lock()
}

type TestableRadosVolume struct {
	*RadosVolume
	radosStubBackend *radosStubBackend
	t                TB
}

func NewTestableRadosVolume(t TB, readonly bool, replication int) *TestableRadosVolume {
	var v *RadosVolume
	radosTracef("rados test: NewTestableRadosVolume readonly=%v replication=%d", readonly, replication)
	radosStubBackend := newRadosStubBackend(uint64(replication))
	pool := radosTestPool
	if pool == "" {
		// Connect using mock radosImplementation instead of real Ceph
		log.Infof("rados test: using mock radosImplementation")
		radosMock := &radosMockImpl{
			b: radosStubBackend,
		}
		v = &RadosVolume{
			Pool:             RadosMockPool,
			MonHost:          RadosMockMonHost,
			ReadOnly:         readonly,
			RadosReplication: replication,
			rados:            radosMock,
		}
	} else {
		// Connect to real Ceph using the real radosImplementation
		log.Infof("rados test: using real radosImplementation")
		v = &RadosVolume{
			Pool:             pool,
			KeyringFile:      radosKeyringFile,
			MonHost:          radosMonHost,
			Cluster:          radosCluster,
			User:             radosUser,
			ReadOnly:         readonly,
			RadosReplication: replication,
		}
	}

	tv := &TestableRadosVolume{
		RadosVolume:      v,
		radosStubBackend: radosStubBackend,
		t:                t,
	}

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

// func TestRadosVolumeCreateBlobRace(t *testing.T) {
// 	v := NewTestableRadosVolume(t, false, 3)
// 	defer v.Teardown()

// 	var wg sync.WaitGroup

// 	v.radosStubBackend.race = make(chan chan struct{})

// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		err := v.Put(context.Background(), TestHash, TestBlock)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 	}()
// 	continuePut := make(chan struct{})
// 	// Wait for the stub's Put to create the empty blob
// 	v.radosStubBackend.race <- continuePut
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		buf := make([]byte, len(TestBlock))
// 		_, err := v.Get(context.Background(), TestHash, buf)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 	}()
// 	// Wait for the stub's Get to get the empty blob
// 	close(v.radosStubBackend.race)
// 	// Allow stub's Put to continue, so the real data is ready
// 	// when the volume's Get retries
// 	<-continuePut
// 	// Wait for Get() and Put() to finish
// 	wg.Wait()
// }

// func TestRadosVolumeCreateBlobRaceDeadline(t *testing.T) {
// 	v := NewTestableRadosVolume(t, false, 3)
// 	defer v.Teardown()

// 	v.PutRaw(TestHash, nil)

// 	buf := new(bytes.Buffer)
// 	v.IndexTo("", buf)
// 	if buf.Len() != 0 {
// 		t.Errorf("Index %+q should be empty", buf.Bytes())
// 	}

// 	v.TouchWithDate(TestHash, time.Now().Add(-1982*time.Millisecond))

// 	allDone := make(chan struct{})
// 	go func() {
// 		defer close(allDone)
// 		buf := make([]byte, BlockSize)
// 		n, err := v.Get(context.Background(), TestHash, buf)
// 		if err != nil {
// 			t.Error(err)
// 			return
// 		}
// 		if n != 0 {
// 			t.Errorf("Got %+q, expected empty buf", buf[:n])
// 		}
// 	}()
// 	select {
// 	case <-allDone:
// 	case <-time.After(time.Second):
// 		t.Error("Get should have stopped waiting for race when block was 2s old")
// 	}

// 	buf.Reset()
// 	v.IndexTo("", buf)
// 	if !bytes.HasPrefix(buf.Bytes(), []byte(TestHash+"+0")) {
// 		t.Errorf("Index %+q should have %+q", buf.Bytes(), TestHash+"+0")
// 	}
// }

// func TestRadosVolumeContextCancelGet(t *testing.T) {
// 	testRadosVolumeContextCancel(t, func(ctx context.Context, v *TestableRadosVolume) error {
// 		v.PutRaw(TestHash, TestBlock)
// 		_, err := v.Get(ctx, TestHash, make([]byte, BlockSize))
// 		return err
// 	})
// }

// func TestRadosVolumeContextCancelPut(t *testing.T) {
// 	testRadosVolumeContextCancel(t, func(ctx context.Context, v *TestableRadosVolume) error {
// 		return v.Put(ctx, TestHash, make([]byte, BlockSize))
// 	})
// }

// func TestRadosVolumeContextCancelCompare(t *testing.T) {
// 	testRadosVolumeContextCancel(t, func(ctx context.Context, v *TestableRadosVolume) error {
// 		v.PutRaw(TestHash, TestBlock)
// 		return v.Compare(ctx, TestHash, TestBlock2)
// 	})
// }

// func testRadosVolumeContextCancel(t *testing.T, testFunc func(context.Context, *TestableRadosVolume) error) {
// 	v := NewTestableRadosVolume(t, false, 3)
// 	defer v.Teardown()
// 	v.radosStubBackend.race = make(chan chan struct{})

// 	ctx, cancel := context.WithCancel(context.Background())
// 	allDone := make(chan struct{})
// 	go func() {
// 		defer close(allDone)
// 		err := testFunc(ctx, v)
// 		if err != context.Canceled {
// 			t.Errorf("got %T %q, expected %q", err, err, context.Canceled)
// 		}
// 	}()
// 	releaseHandler := make(chan struct{})
// 	select {
// 	case <-allDone:
// 		t.Error("testFunc finished without waiting for v.radosStubBackend.race")
// 	case <-time.After(10 * time.Second):
// 		t.Error("timed out waiting to enter handler")
// 	case v.radosStubBackend.race <- releaseHandler:
// 	}

// 	cancel()

// 	select {
// 	case <-time.After(10 * time.Second):
// 		t.Error("timed out waiting to cancel")
// 	case <-allDone:
// 	}

// 	go func() {
// 		<-releaseHandler
// 	}()
// }

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

func (v *TestableRadosVolume) PutRaw(locator string, data []byte) {
	radosTracef("radostest: PutRaw putting locator=%s len(data)=%d data='%s'", locator, len(data), data)

	// need to temporarily disable ReadOnly status and restore it after the call to Put
	defer func(ro bool) {
		v.ReadOnly = ro
	}(v.ReadOnly)

	v.ReadOnly = false
	err := v.Put(context.Background(), locator, data)
	if err != nil {
		v.t.Fatalf("PutRaw failed to put locator %s: %s", locator, err)
	}
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

func (v *TestableRadosVolume) Teardown() {}
