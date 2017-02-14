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
	trackers map[string]*runTracker
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
		tracked := d.trackedUUIDs()
		d.checkForUpdates([][]interface{}{
			{"uuid", "in", tracked}})
		d.checkForUpdates([][]interface{}{
			{"locked_by_uuid", "=", d.auth.UUID},
			{"uuid", "not in", tracked}})
		d.checkForUpdates([][]interface{}{
			{"state", "=", Queued},
			{"priority", ">", "0"},
			{"uuid", "not in", tracked}})
		select {
		case <-poll.C:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *Dispatcher) trackedUUIDs() []string {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if len(d.trackers) == 0 {
		// API bug: ["uuid", "not in", []] does not work as
		// expected, but this does:
		return []string{"this-uuid-does-not-exist"}
	}
	uuids := make([]string, 0, len(d.trackers))
	for x := range d.trackers {
		uuids = append(uuids, x)
	}
	return uuids
}

// Start a runner in a new goroutine, and send the initial container
// record to its updates channel.
func (d *Dispatcher) start(c arvados.Container) *runTracker {
	tracker := &runTracker{updates: make(chan arvados.Container, 1)}
	tracker.updates <- c
	go func() {
		d.RunContainer(d, c, tracker.updates)

		d.mtx.Lock()
		delete(d.trackers, c.UUID)
		d.mtx.Unlock()
	}()
	return tracker
}

func (d *Dispatcher) checkForUpdates(filters [][]interface{}) {
	params := arvadosclient.Dict{
		"filters": filters,
		"order":   []string{"priority desc"}}

	var list arvados.ContainerList
	for offset, more := 0, true; more; offset += len(list.Items) {
		params["offset"] = offset
		err := d.Arv.List("containers", params, &list)
		if err != nil {
			log.Printf("Error getting list of containers: %q", err)
			return
		}
		more = list.ItemsAvailable > len(list.Items)
		d.checkListForUpdates(list.Items)
	}
}

func (d *Dispatcher) checkListForUpdates(containers []arvados.Container) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	if d.trackers == nil {
		d.trackers = make(map[string]*runTracker)
	}

	for _, c := range containers {
		tracker, alreadyTracking := d.trackers[c.UUID]
		if c.LockedByUUID != "" && c.LockedByUUID != d.auth.UUID {
			log.Printf("debug: ignoring %s locked by %s", c.UUID, c.LockedByUUID)
		} else if alreadyTracking {
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
				d.trackers[c.UUID] = d.start(c)
			case Locked, Running:
				if !d.throttle.Check(c.UUID) {
					break
				}
				d.trackers[c.UUID] = d.start(c)
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
	updates chan arvados.Container
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
