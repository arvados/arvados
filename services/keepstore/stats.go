package main

import (
	"sync/atomic"
)

type statsTicker struct {
	Errors   uint64
	InBytes  uint64
	OutBytes uint64
}

// Tick increments each of the given counters by 1 using
// atomic.AddUint64.
func (s *statsTicker) Tick(counters ...*uint64) {
	for _, counter := range counters {
		atomic.AddUint64(counter, 1)
	}
}

func (s *statsTicker) TickErr(err error) {
	if err == nil {
		return
	}
	s.Tick(&s.Errors)
}

func (s *statsTicker) TickInBytes(n uint64) {
	atomic.AddUint64(&s.InBytes, n)
}

func (s *statsTicker) TickOutBytes(n uint64) {
	atomic.AddUint64(&s.OutBytes, n)
}
