// Package dispatch is a helper library for building Arvados container
// dispatchers.
package dispatch

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
)

const (
	Queued    = arvados.ContainerStateQueued
	Locked    = arvados.ContainerStateLocked
	Running   = arvados.ContainerStateRunning
	Complete  = arvados.ContainerStateComplete
	Cancelled = arvados.ContainerStateCancelled
)

type Dispatcher struct {
	Arv *arvadosclient.ArvadosClient

	// Queue polling frequency
	PollPeriod time.Duration

	// Time to wait between successive attempts to run the same container
	MinRetryPeriod time.Duration

	// Func that implements the container lifecycle. Must be set
	// to a non-nil DispatchFunc before calling Run().
	RunContainer DispatchFunc

	auth     arvados.APIClientAuthorization
	mtx      sync.Mutex
	running  map[string]*runTracker
	throttle throttle
}

// A DispatchFunc executes a container (if the container record is
// Locked) or resume monitoring an already-running container, and wait
// until that container exits.
//
// While the container runs, the DispatchFunc should listen for
// updated container records on the provided channel. When the channel
// closes, the DispatchFunc should stop the container if it's still
// running, and return.
//
// The DispatchFunc should not return until the container is finished.
type DispatchFunc func(*Dispatcher, arvados.Container, <-chan arvados.Container)

// Run watches the API server's queue for containers that are either
// ready to run and available to lock, or are already locked by this
// dispatcher's token. When a new one appears, Run calls RunContainer
// in a new goroutine.
func (d *Dispatcher) Run(ctx context.Context) error {
	err := d.Arv.Call("GET", "api_client_authorizations", "", "current", nil, &d.auth)
	if err != nil {
		return fmt.Errorf("error getting my token UUID: %v", err)
	}

	d.throttle.hold = d.MinRetryPeriod

	poll := time.NewTicker(d.PollPeriod)
	defer poll.Stop()

	for {
		d.checkForUpdates([][]interface{}{
			{"uuid", "in", d.runningUUIDs()}})
		d.checkForUpdates([][]interface{}{
			{"locked_by_uuid", "=", d.auth.UUID},
			{"uuid", "not in", d.runningUUIDs()}})
		d.checkForUpdates([][]interface{}{
			{"state", "=", Queued},
			{"priority", ">", "0"},
			{"uuid", "not in", d.runningUUIDs()}})
		select {
		case <-poll.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *Dispatcher) runningUUIDs() []string {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if len(d.running) == 0 {
		// API bug: ["uuid", "not in", []] does not match everything
		return []string{"X"}
	}
	uuids := make([]string, 0, len(d.running))
	for x := range d.running {
		uuids = append(uuids, x)
	}
	return uuids
}

// Start a runner in a new goroutine, and send the initial container
// record to its updates channel.
func (d *Dispatcher) start(c arvados.Container) *runTracker {
	updates := make(chan arvados.Container, 1)
	tracker := &runTracker{updates: updates}
	tracker.updates <- c
	go func() {
		d.RunContainer(d, c, tracker.updates)

		d.mtx.Lock()
		delete(d.running, c.UUID)
		d.mtx.Unlock()
	}()
	return tracker
}

func (d *Dispatcher) checkForUpdates(filters [][]interface{}) {
	params := arvadosclient.Dict{
		"filters": filters,
		"order":   []string{"priority desc"},
		"limit":   "1000"}

	var list arvados.ContainerList
	err := d.Arv.List("containers", params, &list)
	if err != nil {
		log.Printf("Error getting list of containers: %q", err)
		return
	}

	if list.ItemsAvailable > len(list.Items) {
		// TODO: support paging
		log.Printf("Warning!  %d containers are available but only received %d, paged requests are not yet supported, some containers may be ignored.",
			list.ItemsAvailable,
			len(list.Items))
	}

	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.running == nil {
		d.running = make(map[string]*runTracker)
	}

	for _, c := range list.Items {
		tracker, running := d.running[c.UUID]
		if c.LockedByUUID != "" && c.LockedByUUID != d.auth.UUID {
			log.Printf("debug: ignoring %s locked by %s", c.UUID, c.LockedByUUID)
		} else if running {
			switch c.State {
			case Queued:
				tracker.close()
			case Locked, Running:
				tracker.update(c)
			case Cancelled, Complete:
				tracker.close()
			}
		} else {
			switch c.State {
			case Queued:
				if !d.throttle.Check(c.UUID) {
					break
				}
				err := d.lock(c.UUID)
				if err != nil {
					log.Printf("debug: error locking container %s: %s", c.UUID, err)
					break
				}
				c.State = Locked
				d.running[c.UUID] = d.start(c)
			case Locked, Running:
				if !d.throttle.Check(c.UUID) {
					break
				}
				d.running[c.UUID] = d.start(c)
			case Cancelled, Complete:
				tracker.close()
			}
		}
	}
}

// UpdateState makes an API call to change the state of a container.
func (d *Dispatcher) UpdateState(uuid string, state arvados.ContainerState) error {
	err := d.Arv.Update("containers", uuid,
		arvadosclient.Dict{
			"container": arvadosclient.Dict{"state": state},
		}, nil)
	if err != nil {
		log.Printf("Error updating container %s to state %q: %s", uuid, state, err)
	}
	return err
}

// Lock makes the lock API call which updates the state of a container to Locked.
func (d *Dispatcher) lock(uuid string) error {
	return d.Arv.Call("POST", "containers", uuid, "lock", nil, nil)
}

// Unlock makes the unlock API call which updates the state of a container to Queued.
func (d *Dispatcher) Unlock(uuid string) error {
	return d.Arv.Call("POST", "containers", uuid, "unlock", nil, nil)
}

type runTracker struct {
	closing bool
	updates chan<- arvados.Container
}

func (tracker *runTracker) close() {
	if !tracker.closing {
		close(tracker.updates)
	}
	tracker.closing = true
}

func (tracker *runTracker) update(c arvados.Container) {
	if tracker.closing {
		return
	}
	select {
	case <-tracker.updates:
		log.Printf("debug: runner is handling updates slowly, discarded previous update for %s", c.UUID)
	default:
	}
	tracker.updates <- c
}
