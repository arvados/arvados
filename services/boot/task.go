package main

import (
	"log"
	"sync"
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
	lock    sync.Mutex
	version version
}

var TaskState = taskStateMap{
	s: make(map[task]taskState),
}

func (m *taskStateMap) Set(t task, s taskState) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if old, ok := m.s[t]; ok && old == s {
		return
	}
	m.s[t] = s
	m.version++
}

func (m *taskStateMap) Version() version {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.version
}

func (m *taskStateMap) Get(t task) taskState {
	m.lock.Lock()
	defer m.lock.Unlock()
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

func runTasks(tasks []task) {
	for _, t := range tasks {
		TaskState.Set(t, StateChecking)
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
	ShortName() string
	String() string
	Check() error
	CanFix() bool
	Fix() error
	Children() []task
}
