// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"golang.org/x/crypto/ssh"
)

// LameInstanceSet creates instances that boot but can't run
// containers.
type LameInstanceSet struct {
	Hold chan bool // set to make(chan bool) to hold operations until Release is called

	mtx       sync.Mutex
	instances map[*lameInstance]bool
}

// Create returns a new instance.
func (p *LameInstanceSet) Create(_ context.Context, instType arvados.InstanceType, imageID cloud.ImageID, tags cloud.InstanceTags, pubkey ssh.PublicKey) (cloud.Instance, error) {
	inst := &lameInstance{
		p:            p,
		id:           cloud.InstanceID(fmt.Sprintf("lame-%x", rand.Uint64())),
		providerType: instType.ProviderType,
	}
	inst.SetTags(context.TODO(), tags)
	if p.Hold != nil {
		p.Hold <- true
	}
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if p.instances == nil {
		p.instances = map[*lameInstance]bool{}
	}
	p.instances[inst] = true
	return inst, nil
}

// Instances returns the instances that haven't been destroyed.
func (p *LameInstanceSet) Instances(context.Context, cloud.InstanceTags) ([]cloud.Instance, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	var instances []cloud.Instance
	for i := range p.instances {
		instances = append(instances, i)
	}
	return instances, nil
}

// Stop is a no-op, but exists to satisfy cloud.InstanceSet.
func (p *LameInstanceSet) Stop() {
}

// Release n held calls. Blocks if n calls aren't already
// waiting. Blocks forever if Hold is nil.
func (p *LameInstanceSet) Release(n int) {
	for i := 0; i < n; i++ {
		<-p.Hold
	}
}

type lameInstance struct {
	p            *LameInstanceSet
	id           cloud.InstanceID
	providerType string
	tags         cloud.InstanceTags
}

func (inst *lameInstance) ID() cloud.InstanceID {
	return inst.id
}

func (inst *lameInstance) String() string {
	return fmt.Sprint(inst.id)
}

func (inst *lameInstance) ProviderType() string {
	return inst.providerType
}

func (inst *lameInstance) Address() string {
	return "0.0.0.0:1234"
}

func (inst *lameInstance) SetTags(_ context.Context, tags cloud.InstanceTags) error {
	inst.p.mtx.Lock()
	defer inst.p.mtx.Unlock()
	inst.tags = cloud.InstanceTags{}
	for k, v := range tags {
		inst.tags[k] = v
	}
	return nil
}

func (inst *lameInstance) Destroy(context.Context) error {
	if inst.p.Hold != nil {
		inst.p.Hold <- true
	}
	inst.p.mtx.Lock()
	defer inst.p.mtx.Unlock()
	delete(inst.p.instances, inst)
	return nil
}

func (inst *lameInstance) Tags() cloud.InstanceTags {
	return inst.tags
}

func (inst *lameInstance) VerifyHostKey(context.Context, ssh.PublicKey, *ssh.Client) error {
	return nil
}
