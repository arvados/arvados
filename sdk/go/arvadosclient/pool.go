package arvadosclient

import (
	"sync"
)

type ClientPool struct {
	sync.Pool
	lastErr error
}

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

func (p *ClientPool) Err() error {
	return p.lastErr
}

func (p *ClientPool) Get() *ArvadosClient {
	c, ok := p.Pool.Get().(*ArvadosClient)
	if !ok {
		return nil
	}
	return c
}

func (p *ClientPool) Put(c *ArvadosClient) {
	p.Pool.Put(c)
}
