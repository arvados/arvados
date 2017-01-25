package main

import (
	"context"
	"log"
	"sync"
	"time"
)

type taskState string

const (
	StateUnchecked taskState = "Unchecked"
	StateChecking            = "Checking"
	StateFixing              = "Fixing"
	StateFailed              = "Failed"
	StateOK                  = "OK"
)

type version int64

type taskStateMap struct {
	s       map[task]taskState
	cond    *sync.Cond
	version version
}

var TaskState = taskStateMap{
	s:    make(map[task]taskState),
	cond: sync.NewCond(&sync.Mutex{}),
}

func (m *taskStateMap) Set(t task, s taskState) {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	if old, ok := m.s[t]; ok && old == s {
		return
	}
	m.s[t] = s
	m.version++
	m.cond.Broadcast()
}

func (m *taskStateMap) Version() version {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	return m.version
}

func (m *taskStateMap) Get(t task) taskState {
	m.cond.L.Lock()
	defer m.cond.L.Unlock()
	if s, ok := m.s[t]; ok {
		return s
	} else {
		return StateUnchecked
	}
}

type repEnt struct {
	ShortName   string
	Description string
	State       taskState
	Children    []repEnt
}

func (m *taskStateMap) Wait(v version, t time.Duration, ctx context.Context) bool {
	ready := make(chan struct{})
	var done bool
	go func() {
		m.cond.L.Lock()
		defer m.cond.L.Unlock()
		for v == m.version && !done {
			m.cond.Wait()
		}
		close(ready)
	}()
	select {
	case <-ready:
		return true
	case <-ctx.Done():
	case <-time.After(t):
	}
	done = true
	m.cond.Broadcast()
	return false
}

func report(tasks []task) ([]repEnt, version) {
	v := TaskState.Version()
	if len(tasks) == 0 {
		return nil, v
	}
	var rep []repEnt
	for _, t := range tasks {
		crep, _ := report(t.Children())
		rep = append(rep, repEnt{
			ShortName:   t.ShortName(),
			Description: t.String(),
			State:       TaskState.Get(t),
			Children:    crep,
		})
	}
	return rep, v
}

func runTasks(cfg *Config, tasks []task) {
	for _, t := range tasks {
		t.Init(cfg)
	}
	for _, t := range tasks {
		if TaskState.Get(t) == taskState("") {
			TaskState.Set(t, StateChecking)
		}
		err := t.Check()
		if err == nil {
			log.Printf("%s: OK", t)
			TaskState.Set(t, StateOK)
			continue
		}
		log.Printf("%s: %s", t, err)
		if !t.CanFix() {
			log.Printf("%s: can't fix")
			TaskState.Set(t, StateFailed)
			continue
		}
		TaskState.Set(t, StateFixing)
		if err = t.Fix(); err != nil {
			log.Printf("%s: can't fix: %s", t, err)
			TaskState.Set(t, StateFailed)
			continue
		}
		if err = t.Check(); err != nil {
			log.Printf("%s: fixed, but still broken?!: %s", t, err)
			TaskState.Set(t, StateFailed)
			continue
		}
		log.Printf("%s: OK", t)
		TaskState.Set(t, StateOK)
	}
}

type task interface {
	Init(*Config)
	ShortName() string
	String() string
	Check() error
	CanFix() bool
	Fix() error
	Children() []task
}
