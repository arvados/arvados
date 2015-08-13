package arvadosclient

import (
	"sync"
)

// A ClientPool is a pool of ArvadosClients. This is useful for
// applications that make API calls using a dynamic set of tokens,
// like web services that pass through their own clients'
// credentials. See arvados-git-httpd for an example, and sync.Pool
// for more information about garbage collection.
type ClientPool struct {
	sync.Pool
	lastErr error
}

// MakeClientPool returns a new empty ClientPool.
func MakeClientPool() *ClientPool {
	p := &ClientPool{}
	p.Pool = sync.Pool{New: func() interface{} {
		arv, err := MakeArvadosClient()
		if err != nil {
			p.lastErr = err
			return nil
		}
		return &arv
	}}
	return p
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
	c, ok := p.Pool.Get().(*ArvadosClient)
	if !ok {
		return nil
	}
	return c
}

// Put puts an ArvadosClient back in the pool.
func (p *ClientPool) Put(c *ArvadosClient) {
	p.Pool.Put(c)
}
