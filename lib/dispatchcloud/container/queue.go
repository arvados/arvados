// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package container

import (
	"io"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type typeChooser func(*arvados.Container) (arvados.InstanceType, error)

// An APIClient performs Arvados API requests. It is typically an
// *arvados.Client.
type APIClient interface {
	RequestAndDecode(dst interface{}, method, path string, body io.Reader, params interface{}) error
}

// A QueueEnt is an entry in the queue, consisting of a container
// record and the instance type that should be used to run it.
type QueueEnt struct {
	// The container to run. Only the UUID, State, Priority, and
	// RuntimeConstraints fields are populated.
	Container    arvados.Container    `json:"container"`
	InstanceType arvados.InstanceType `json:"instance_type"`
}

// String implements fmt.Stringer by returning the queued container's
// UUID.
func (c *QueueEnt) String() string {
	return c.Container.UUID
}

// A Queue is an interface to an Arvados cluster's container
// database. It presents only the containers that are eligible to be
// run by, are already being run by, or have recently been run by the
// present dispatcher.
//
// The Entries, Get, and Forget methods do not block: they return
// immediately, using cached data.
//
// The updating methods (Cancel, Lock, Unlock, Update) do block: they
// return only after the operation has completed.
//
// A Queue's Update method should be called periodically to keep the
// cache up to date.
type Queue struct {
	logger     logrus.FieldLogger
	reg        *prometheus.Registry
	chooseType typeChooser
	client     APIClient

	auth    *arvados.APIClientAuthorization
	current map[string]QueueEnt
	updated time.Time
	mtx     sync.Mutex

	// Methods that modify the Queue (like Lock) add the affected
	// container UUIDs to dontupdate. When applying a batch of
	// updates received from the network, anything appearing in
	// dontupdate is skipped, in case the received update has
	// already been superseded by the locally initiated change.
	// When no network update is in progress, this protection is
	// not needed, and dontupdate is nil.
	dontupdate map[string]struct{}

	// active notification subscribers (see Subscribe)
	subscribers map[<-chan struct{}]chan struct{}
}

// NewQueue returns a new Queue. When a new container appears in the
// Arvados cluster's queue during Update, chooseType will be called to
// assign an appropriate arvados.InstanceType for the queue entry.
func NewQueue(logger logrus.FieldLogger, reg *prometheus.Registry, chooseType typeChooser, client APIClient) *Queue {
	return &Queue{
		logger:      logger,
		reg:         reg,
		chooseType:  chooseType,
		client:      client,
		current:     map[string]QueueEnt{},
		subscribers: map[<-chan struct{}]chan struct{}{},
	}
}

// Subscribe returns a channel that becomes ready to receive when an
// entry in the Queue is updated.
//
//	ch := q.Subscribe()
//	defer q.Unsubscribe(ch)
//	for range ch {
//		// ...
//	}
func (cq *Queue) Subscribe() <-chan struct{} {
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	ch := make(chan struct{}, 1)
	cq.subscribers[ch] = ch
	return ch
}

// Unsubscribe stops sending updates to the given channel. See
// Subscribe.
func (cq *Queue) Unsubscribe(ch <-chan struct{}) {
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	delete(cq.subscribers, ch)
}

// Caller must have lock.
func (cq *Queue) notify() {
	for _, ch := range cq.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// Forget drops the specified container from the cache. It should be
// called on finalized containers to avoid leaking memory over
// time. It is a no-op if the indicated container is not in a
// finalized state.
func (cq *Queue) Forget(uuid string) {
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	ctr := cq.current[uuid].Container
	if ctr.State == arvados.ContainerStateComplete || ctr.State == arvados.ContainerStateCancelled {
		delete(cq.current, uuid)
	}
}

// Get returns the (partial) Container record for the specified
// container. Like a map lookup, its second return value is false if
// the specified container is not in the Queue.
func (cq *Queue) Get(uuid string) (arvados.Container, bool) {
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	if ctr, ok := cq.current[uuid]; !ok {
		return arvados.Container{}, false
	} else {
		return ctr.Container, true
	}
}

// Entries returns all cache entries, keyed by container UUID.
//
// The returned threshold indicates the maximum age of any cached data
// returned in the map. This makes it possible for a scheduler to
// determine correctly the outcome of a remote process that updates
// container state. It must first wait for the remote process to exit,
// then wait for the Queue to start and finish its next Update --
// i.e., it must wait until threshold > timeProcessExited.
func (cq *Queue) Entries() (entries map[string]QueueEnt, threshold time.Time) {
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	entries = make(map[string]QueueEnt, len(cq.current))
	for uuid, ctr := range cq.current {
		entries[uuid] = ctr
	}
	threshold = cq.updated
	return
}

// Update refreshes the cache from the Arvados API. It adds newly
// queued containers, and updates the state of previously queued
// containers.
func (cq *Queue) Update() error {
	cq.mtx.Lock()
	cq.dontupdate = map[string]struct{}{}
	updateStarted := time.Now()
	cq.mtx.Unlock()

	next, err := cq.poll()
	if err != nil {
		return err
	}

	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	for uuid, ctr := range next {
		if _, keep := cq.dontupdate[uuid]; keep {
			continue
		}
		if cur, ok := cq.current[uuid]; !ok {
			cq.addEnt(uuid, *ctr)
		} else {
			cur.Container = *ctr
			cq.current[uuid] = cur
		}
	}
	for uuid := range cq.current {
		if _, keep := cq.dontupdate[uuid]; keep {
			continue
		} else if _, keep = next[uuid]; keep {
			continue
		} else {
			delete(cq.current, uuid)
		}
	}
	cq.dontupdate = nil
	cq.updated = updateStarted
	cq.notify()
	return nil
}

func (cq *Queue) addEnt(uuid string, ctr arvados.Container) {
	it, err := cq.chooseType(&ctr)
	if err != nil && (ctr.State == arvados.ContainerStateQueued || ctr.State == arvados.ContainerStateLocked) {
		// We assume here that any chooseType error is a hard
		// error: it wouldn't help to try again, or to leave
		// it for a different dispatcher process to attempt.
		errorString := err.Error()
		cq.logger.WithField("ContainerUUID", ctr.UUID).Warn("cancel container with no suitable instance type")
		go func() {
			var err error
			defer func() {
				if err == nil {
					return
				}
				// On failure, check current container
				// state, and don't log the error if
				// the failure came from losing a
				// race.
				var latest arvados.Container
				cq.client.RequestAndDecode(&latest, "GET", "arvados/v1/containers/"+ctr.UUID, nil, map[string][]string{"select": {"state"}})
				if latest.State == arvados.ContainerStateCancelled {
					return
				}
				cq.logger.WithField("ContainerUUID", ctr.UUID).WithError(err).Warn("error while trying to cancel unsatisfiable container")
			}()
			if ctr.State == arvados.ContainerStateQueued {
				err = cq.Lock(ctr.UUID)
				if err != nil {
					return
				}
			}
			err = cq.setRuntimeError(ctr.UUID, errorString)
			if err != nil {
				return
			}
			err = cq.Cancel(ctr.UUID)
			if err != nil {
				return
			}
		}()
		return
	}
	cq.current[uuid] = QueueEnt{Container: ctr, InstanceType: it}
}

// Lock acquires the dispatch lock for the given container.
func (cq *Queue) Lock(uuid string) error {
	return cq.apiUpdate(uuid, "lock")
}

// Unlock releases the dispatch lock for the given container.
func (cq *Queue) Unlock(uuid string) error {
	return cq.apiUpdate(uuid, "unlock")
}

// setRuntimeError sets runtime_status["error"] to the given value.
// Container should already have state==Locked or Running.
func (cq *Queue) setRuntimeError(uuid, errorString string) error {
	return cq.client.RequestAndDecode(nil, "PUT", "arvados/v1/containers/"+uuid, nil, map[string]map[string]map[string]interface{}{
		"container": {
			"runtime_status": {
				"error": errorString,
			},
		},
	})
}

// Cancel cancels the given container.
func (cq *Queue) Cancel(uuid string) error {
	err := cq.client.RequestAndDecode(nil, "PUT", "arvados/v1/containers/"+uuid, nil, map[string]map[string]interface{}{
		"container": {"state": arvados.ContainerStateCancelled},
	})
	if err != nil {
		return err
	}
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	cq.notify()
	return nil
}

func (cq *Queue) apiUpdate(uuid, action string) error {
	var resp arvados.Container
	err := cq.client.RequestAndDecode(&resp, "POST", "arvados/v1/containers/"+uuid+"/"+action, nil, nil)
	if err != nil {
		return err
	}

	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	if cq.dontupdate != nil {
		cq.dontupdate[uuid] = struct{}{}
	}
	if ent, ok := cq.current[uuid]; !ok {
		cq.addEnt(uuid, resp)
	} else {
		ent.Container.State, ent.Container.Priority, ent.Container.LockedByUUID = resp.State, resp.Priority, resp.LockedByUUID
		cq.current[uuid] = ent
	}
	cq.notify()
	return nil
}

func (cq *Queue) poll() (map[string]*arvados.Container, error) {
	cq.mtx.Lock()
	size := len(cq.current)
	auth := cq.auth
	cq.mtx.Unlock()

	if auth == nil {
		auth = &arvados.APIClientAuthorization{}
		err := cq.client.RequestAndDecode(auth, "GET", "arvados/v1/api_client_authorizations/current", nil, nil)
		if err != nil {
			return nil, err
		}
		cq.mtx.Lock()
		cq.auth = auth
		cq.mtx.Unlock()
	}

	next := make(map[string]*arvados.Container, size)
	apply := func(updates []arvados.Container) {
		for _, upd := range updates {
			if next[upd.UUID] == nil {
				next[upd.UUID] = &arvados.Container{}
			}
			*next[upd.UUID] = upd
		}
	}
	selectParam := []string{"uuid", "state", "priority", "runtime_constraints"}
	limitParam := 1000

	mine, err := cq.fetchAll(arvados.ResourceListParams{
		Select:  selectParam,
		Order:   "uuid",
		Limit:   &limitParam,
		Count:   "none",
		Filters: []arvados.Filter{{"locked_by_uuid", "=", auth.UUID}},
	})
	if err != nil {
		return nil, err
	}
	apply(mine)

	avail, err := cq.fetchAll(arvados.ResourceListParams{
		Select:  selectParam,
		Order:   "uuid",
		Limit:   &limitParam,
		Count:   "none",
		Filters: []arvados.Filter{{"state", "=", arvados.ContainerStateQueued}, {"priority", ">", "0"}},
	})
	if err != nil {
		return nil, err
	}
	apply(avail)

	var missing []string
	cq.mtx.Lock()
	for uuid, ent := range cq.current {
		if next[uuid] == nil &&
			ent.Container.State != arvados.ContainerStateCancelled &&
			ent.Container.State != arvados.ContainerStateComplete {
			missing = append(missing, uuid)
		}
	}
	cq.mtx.Unlock()

	for i, page := 0, 20; i < len(missing); i += page {
		batch := missing[i:]
		if len(batch) > page {
			batch = batch[:page]
		}
		ended, err := cq.fetchAll(arvados.ResourceListParams{
			Select:  selectParam,
			Order:   "uuid",
			Count:   "none",
			Filters: []arvados.Filter{{"uuid", "in", batch}},
		})
		if err != nil {
			return nil, err
		}
		apply(ended)
	}
	return next, nil
}

func (cq *Queue) fetchAll(initialParams arvados.ResourceListParams) ([]arvados.Container, error) {
	var results []arvados.Container
	params := initialParams
	params.Offset = 0
	for {
		// This list variable must be a new one declared
		// inside the loop: otherwise, items in the API
		// response would get deep-merged into the items
		// loaded in previous iterations.
		var list arvados.ContainerList

		err := cq.client.RequestAndDecode(&list, "GET", "arvados/v1/containers", nil, params)
		if err != nil {
			return nil, err
		}
		if len(list.Items) == 0 {
			break
		}

		results = append(results, list.Items...)
		if len(params.Order) == 1 && params.Order == "uuid" {
			params.Filters = append(initialParams.Filters, arvados.Filter{"uuid", ">", list.Items[len(list.Items)-1].UUID})
		} else {
			params.Offset += len(list.Items)
		}
	}
	return results, nil
}
