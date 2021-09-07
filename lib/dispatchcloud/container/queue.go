// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package container

import (
	"errors"
	"io"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
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
	// The container to run. Only the UUID, State, Priority,
	// RuntimeConstraints, Mounts, and ContainerImage fields are
	// populated.
	Container    arvados.Container    `json:"container"`
	InstanceType arvados.InstanceType `json:"instance_type"`
	FirstSeenAt  time.Time            `json:"first_seen_at"`
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
	cq := &Queue{
		logger:      logger,
		chooseType:  chooseType,
		client:      client,
		current:     map[string]QueueEnt{},
		subscribers: map[<-chan struct{}]chan struct{}{},
	}
	if reg != nil {
		go cq.runMetrics(reg)
	}
	return cq
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
	if ctr.State == arvados.ContainerStateComplete || ctr.State == arvados.ContainerStateCancelled || (ctr.State == arvados.ContainerStateQueued && ctr.Priority == 0) {
		cq.delEnt(uuid, ctr.State)
	}
}

// Get returns the (partial) Container record for the specified
// container. Like a map lookup, its second return value is false if
// the specified container is not in the Queue.
func (cq *Queue) Get(uuid string) (arvados.Container, bool) {
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	ctr, ok := cq.current[uuid]
	if !ok {
		return arvados.Container{}, false
	}
	return ctr.Container, true
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
		if _, dontupdate := cq.dontupdate[uuid]; dontupdate {
			// Don't clobber a local update that happened
			// after we started polling.
			continue
		}
		if cur, ok := cq.current[uuid]; !ok {
			cq.addEnt(uuid, *ctr)
		} else {
			cur.Container = *ctr
			cq.current[uuid] = cur
		}
	}
	for uuid, ent := range cq.current {
		if _, dontupdate := cq.dontupdate[uuid]; dontupdate {
			// Don't expunge an entry that was
			// added/updated locally after we started
			// polling.
			continue
		} else if _, stillpresent := next[uuid]; !stillpresent {
			// Expunge an entry that no longer appears in
			// the poll response (evidently it's
			// cancelled, completed, deleted, or taken by
			// a different dispatcher).
			cq.delEnt(uuid, ent.Container.State)
		}
	}
	cq.dontupdate = nil
	cq.updated = updateStarted
	cq.notify()
	return nil
}

// Caller must have lock.
func (cq *Queue) delEnt(uuid string, state arvados.ContainerState) {
	cq.logger.WithFields(logrus.Fields{
		"ContainerUUID": uuid,
		"State":         state,
	}).Info("dropping container from queue")
	delete(cq.current, uuid)
}

// Caller must have lock.
func (cq *Queue) addEnt(uuid string, ctr arvados.Container) {
	it, err := cq.chooseType(&ctr)
	if err != nil && (ctr.State == arvados.ContainerStateQueued || ctr.State == arvados.ContainerStateLocked) {
		// We assume here that any chooseType error is a hard
		// error: it wouldn't help to try again, or to leave
		// it for a different dispatcher process to attempt.
		errorString := err.Error()
		logger := cq.logger.WithField("ContainerUUID", ctr.UUID)
		logger.WithError(err).Warn("cancel container with no suitable instance type")
		go func() {
			if ctr.State == arvados.ContainerStateQueued {
				// Can't set runtime error without
				// locking first.
				err := cq.Lock(ctr.UUID)
				if err != nil {
					logger.WithError(err).Warn("lock failed")
					return
					// ...and try again on the
					// next Update, if the problem
					// still exists.
				}
			}
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
				logger.WithError(err).Warn("error while trying to cancel unsatisfiable container")
			}()
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
	cq.logger.WithFields(logrus.Fields{
		"ContainerUUID": ctr.UUID,
		"State":         ctr.State,
		"Priority":      ctr.Priority,
		"InstanceType":  it.Name,
	}).Info("adding container to queue")
	cq.current[uuid] = QueueEnt{Container: ctr, InstanceType: it, FirstSeenAt: time.Now()}
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
	var resp arvados.Container
	err := cq.client.RequestAndDecode(&resp, "PUT", "arvados/v1/containers/"+uuid, nil, map[string]map[string]interface{}{
		"container": {"state": arvados.ContainerStateCancelled},
	})
	if err != nil {
		return err
	}
	cq.updateWithResp(uuid, resp)
	return nil
}

func (cq *Queue) apiUpdate(uuid, action string) error {
	var resp arvados.Container
	err := cq.client.RequestAndDecode(&resp, "POST", "arvados/v1/containers/"+uuid+"/"+action, nil, nil)
	if err != nil {
		return err
	}
	cq.updateWithResp(uuid, resp)
	return nil
}

// Update the local queue with the response received from a
// state-changing API request (lock/unlock/cancel).
func (cq *Queue) updateWithResp(uuid string, resp arvados.Container) {
	cq.mtx.Lock()
	defer cq.mtx.Unlock()
	if cq.dontupdate != nil {
		cq.dontupdate[uuid] = struct{}{}
	}
	ent, ok := cq.current[uuid]
	if !ok {
		// Container is not in queue (e.g., it was not added
		// because there is no suitable instance type, and
		// we're just locking/updating it in order to set an
		// error message). No need to add it, and we don't
		// necessarily have enough information to add it here
		// anyway because lock/unlock responses don't include
		// runtime_constraints.
		return
	}
	ent.Container.State, ent.Container.Priority, ent.Container.LockedByUUID = resp.State, resp.Priority, resp.LockedByUUID
	cq.current[uuid] = ent
	cq.notify()
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
	selectParam := []string{"uuid", "state", "priority", "runtime_constraints", "container_image", "mounts", "scheduling_parameters", "created_at"}
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

	missing := map[string]bool{}
	cq.mtx.Lock()
	for uuid, ent := range cq.current {
		if next[uuid] == nil &&
			ent.Container.State != arvados.ContainerStateCancelled &&
			ent.Container.State != arvados.ContainerStateComplete {
			missing[uuid] = true
		}
	}
	cq.mtx.Unlock()

	for len(missing) > 0 {
		var batch []string
		for uuid := range missing {
			batch = append(batch, uuid)
			if len(batch) == 20 {
				break
			}
		}
		filters := []arvados.Filter{{"uuid", "in", batch}}
		ended, err := cq.fetchAll(arvados.ResourceListParams{
			Select:  selectParam,
			Order:   "uuid",
			Count:   "none",
			Filters: filters,
		})
		if err != nil {
			return nil, err
		}
		apply(ended)
		if len(ended) == 0 {
			// This is the only case where we can conclude
			// a container has been deleted from the
			// database. A short (but non-zero) page, on
			// the other hand, can be caused by a response
			// size limit.
			for _, uuid := range batch {
				cq.logger.WithField("ContainerUUID", uuid).Warn("container not found by controller (deleted?)")
				delete(missing, uuid)
				cq.mtx.Lock()
				cq.delEnt(uuid, cq.current[uuid].Container.State)
				cq.mtx.Unlock()
			}
			continue
		}
		for _, ctr := range ended {
			if _, ok := missing[ctr.UUID]; !ok {
				msg := "BUG? server response did not match requested filters, erroring out rather than risk deadlock"
				cq.logger.WithFields(logrus.Fields{
					"ContainerUUID": ctr.UUID,
					"Filters":       filters,
				}).Error(msg)
				return nil, errors.New(msg)
			}
			delete(missing, ctr.UUID)
		}
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

func (cq *Queue) runMetrics(reg *prometheus.Registry) {
	mEntries := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "queue_entries",
		Help:      "Number of active container entries in the controller database.",
	}, []string{"state", "instance_type"})
	reg.MustRegister(mEntries)

	type entKey struct {
		state arvados.ContainerState
		inst  string
	}
	count := map[entKey]int{}

	ch := cq.Subscribe()
	defer cq.Unsubscribe(ch)
	for range ch {
		for k := range count {
			count[k] = 0
		}
		ents, _ := cq.Entries()
		for _, ent := range ents {
			count[entKey{ent.Container.State, ent.InstanceType.Name}]++
		}
		for k, v := range count {
			mEntries.WithLabelValues(string(k.state), k.inst).Set(float64(v))
		}
	}
}
