// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package test

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	math_rand "math/rand"
	"sync"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/ssh"
)

type StubExecFunc func(instance cloud.Instance, command string, stdin io.Reader, stdout, stderr io.Writer) uint32

// A StubDriver implements cloud.Driver by setting up local SSH
// servers that pass their command execution requests to the provided
// SSHExecFunc.
type StubDriver struct {
	Exec           StubExecFunc
	HostKey        ssh.Signer
	AuthorizedKeys []ssh.PublicKey
	instanceSets   []*StubInstanceSet
}

// InstanceSet returns a new *StubInstanceSet.
func (sd *StubDriver) InstanceSet(params map[string]interface{}, id cloud.InstanceSetID) (cloud.InstanceSet, error) {
	sis := StubInstanceSet{
		driver:  sd,
		servers: map[cloud.InstanceID]*stubServer{},
	}
	sd.instanceSets = append(sd.instanceSets, &sis)
	return &sis, mapstructure.Decode(params, &sis)
}

// InstanceSets returns all instances that have been created by the
// driver. This can be used to test a component that uses the driver
// but doesn't expose the InstanceSets it has created.
func (sd *StubDriver) InstanceSets() []*StubInstanceSet {
	return sd.instanceSets
}

type StubInstanceSet struct {
	driver  *StubDriver
	servers map[cloud.InstanceID]*stubServer
	mtx     sync.RWMutex
	stopped bool
}

func (sis *StubInstanceSet) Create(it arvados.InstanceType, image cloud.ImageID, tags cloud.InstanceTags, authKey ssh.PublicKey) (cloud.Instance, error) {
	sis.mtx.Lock()
	defer sis.mtx.Unlock()
	if sis.stopped {
		return nil, errors.New("StubInstanceSet: Create called after Stop")
	}
	ak := sis.driver.AuthorizedKeys
	if authKey != nil {
		ak = append([]ssh.PublicKey{authKey}, ak...)
	}
	var ss *stubServer
	ss = &stubServer{
		sis:          sis,
		id:           cloud.InstanceID(fmt.Sprintf("stub-%s-%x", it.ProviderType, math_rand.Int63())),
		tags:         copyTags(tags),
		providerType: it.ProviderType,
		SSHService: SSHService{
			HostKey:        sis.driver.HostKey,
			AuthorizedKeys: ak,
			Exec: func(command string, stdin io.Reader, stdout, stderr io.Writer) uint32 {
				return sis.driver.Exec(ss.Instance(), command, stdin, stdout, stderr)
			},
		},
	}

	sis.servers[ss.id] = ss
	return ss.Instance(), nil
}

func (sis *StubInstanceSet) Instances(cloud.InstanceTags) ([]cloud.Instance, error) {
	sis.mtx.RLock()
	defer sis.mtx.RUnlock()
	var r []cloud.Instance
	for _, ss := range sis.servers {
		r = append(r, ss.Instance())
	}
	return r, nil
}

func (sis *StubInstanceSet) Stop() {
	sis.mtx.Lock()
	defer sis.mtx.Unlock()
	if sis.stopped {
		panic("Stop called twice")
	}
	sis.stopped = true
}

type stubServer struct {
	sis          *StubInstanceSet
	id           cloud.InstanceID
	tags         cloud.InstanceTags
	providerType string
	SSHService   SSHService
	sync.Mutex
}

func (ss *stubServer) Instance() stubInstance {
	ss.Lock()
	defer ss.Unlock()
	return stubInstance{
		ss:   ss,
		addr: ss.SSHService.Address(),
		// We deliberately return a cached/stale copy of the
		// real tags here, so that (Instance)Tags() sometimes
		// returns old data after a call to
		// (Instance)SetTags().  This is permitted by the
		// driver interface, and this might help remind
		// callers that they need to tolerate it.
		tags: copyTags(ss.tags),
	}
}

type stubInstance struct {
	ss   *stubServer
	addr string
	tags cloud.InstanceTags
}

func (si stubInstance) ID() cloud.InstanceID {
	return si.ss.id
}

func (si stubInstance) Address() string {
	return si.addr
}

func (si stubInstance) Destroy() error {
	si.ss.SSHService.Close()
	sis := si.ss.sis
	sis.mtx.Lock()
	defer sis.mtx.Unlock()
	delete(sis.servers, si.ss.id)
	return nil
}

func (si stubInstance) ProviderType() string {
	return si.ss.providerType
}

func (si stubInstance) SetTags(tags cloud.InstanceTags) error {
	tags = copyTags(tags)
	ss := si.ss
	go func() {
		ss.Lock()
		defer ss.Unlock()
		ss.tags = tags
	}()
	return nil
}

func (si stubInstance) Tags() cloud.InstanceTags {
	return si.tags
}

func (si stubInstance) String() string {
	return string(si.ss.id)
}

func (si stubInstance) VerifyHostKey(key ssh.PublicKey, client *ssh.Client) error {
	buf := make([]byte, 512)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return err
	}
	sig, err := si.ss.sis.driver.HostKey.Sign(rand.Reader, buf)
	if err != nil {
		return err
	}
	return key.Verify(buf, sig)
}

func copyTags(src cloud.InstanceTags) cloud.InstanceTags {
	dst := cloud.InstanceTags{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
