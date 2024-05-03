// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadosclient

import (
	"sync"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// A ClientPool is a pool of ArvadosClients. This is useful for
// applications that make API calls using a dynamic set of tokens,
// like web services that pass through their own clients'
// credentials. See sync.Pool for more information about garbage
// collection.
type ClientPool struct {
	// Initialize new clients by copying this one.
	Prototype *ArvadosClient

	pool      *sync.Pool
	lastErr   error
	setupOnce sync.Once
}

// MakeClientPool returns a new empty ClientPool, using environment
// variables to initialize the prototype.
func MakeClientPool() *ClientPool {
	return MakeClientPoolWith(nil)
}

// MakeClientPoolWith returns a new empty ClientPool with a previously
// initialized arvados.Client.
func MakeClientPoolWith(client *arvados.Client) *ClientPool {
	var err error
	var proto *ArvadosClient

	if client == nil {
		proto, err = MakeArvadosClient()
	} else {
		proto, err = New(client)
	}
	return &ClientPool{
		Prototype: proto,
		lastErr:   err,
	}
}

func (p *ClientPool) setup() {
	p.pool = &sync.Pool{New: func() interface{} {
		if p.lastErr != nil {
			return nil
		}
		c := *p.Prototype
		return &c
	}}
}

// Err returns the error that was encountered last time Get returned
// nil.
func (p *ClientPool) Err() error {
	return p.lastErr
}

// Get returns an ArvadosClient taken from the pool, or a new one if
// the pool is empty. If an existing client is returned, its state
// (including its ApiToken) will be just as it was when it was Put
// back in the pool.
func (p *ClientPool) Get() *ArvadosClient {
	p.setupOnce.Do(p.setup)
	c, ok := p.pool.Get().(*ArvadosClient)
	if !ok {
		return nil
	}
	return c
}

// Put puts an ArvadosClient back in the pool.
func (p *ClientPool) Put(c *ArvadosClient) {
	p.setupOnce.Do(p.setup)
	p.pool.Put(c)
}
