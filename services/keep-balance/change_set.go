// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"encoding/json"
	"fmt"
	"sync"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// Pull is a request to retrieve a block from a remote server, and
// store it locally.
type Pull struct {
	arvados.SizedDigest
	From *KeepService
	To   *KeepMount
}

// MarshalJSON formats a pull request the way keepstore wants to see
// it.
func (p Pull) MarshalJSON() ([]byte, error) {
	type KeepstorePullRequest struct {
		Locator   string   `json:"locator"`
		Servers   []string `json:"servers"`
		MountUUID string   `json:"mount_uuid"`
	}
	return json.Marshal(KeepstorePullRequest{
		Locator:   string(p.SizedDigest[:32]),
		Servers:   []string{p.From.URLBase()},
		MountUUID: p.To.KeepMount.UUID,
	})
}

// Trash is a request to delete a block.
type Trash struct {
	arvados.SizedDigest
	Mtime int64
	From  *KeepMount
}

// MarshalJSON formats a trash request the way keepstore wants to see
// it, i.e., as a bare locator with no +size hint.
func (t Trash) MarshalJSON() ([]byte, error) {
	type KeepstoreTrashRequest struct {
		Locator    string `json:"locator"`
		BlockMtime int64  `json:"block_mtime"`
		MountUUID  string `json:"mount_uuid"`
	}
	return json.Marshal(KeepstoreTrashRequest{
		Locator:    string(t.SizedDigest[:32]),
		BlockMtime: t.Mtime,
		MountUUID:  t.From.KeepMount.UUID,
	})
}

// ChangeSet is a set of change requests that will be sent to a
// keepstore server.
type ChangeSet struct {
	PullLimit  int
	TrashLimit int

	Pulls           []Pull
	PullsDeferred   int // number that weren't added because of PullLimit
	Trashes         []Trash
	TrashesDeferred int // number that weren't added because of TrashLimit
	mutex           sync.Mutex
}

// AddPull adds a Pull operation.
func (cs *ChangeSet) AddPull(p Pull) {
	cs.mutex.Lock()
	if len(cs.Pulls) < cs.PullLimit {
		cs.Pulls = append(cs.Pulls, p)
	} else {
		cs.PullsDeferred++
	}
	cs.mutex.Unlock()
}

// AddTrash adds a Trash operation
func (cs *ChangeSet) AddTrash(t Trash) {
	cs.mutex.Lock()
	if len(cs.Trashes) < cs.TrashLimit {
		cs.Trashes = append(cs.Trashes, t)
	} else {
		cs.TrashesDeferred++
	}
	cs.mutex.Unlock()
}

// String implements fmt.Stringer.
func (cs *ChangeSet) String() string {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	return fmt.Sprintf("ChangeSet{Pulls:%d, Trashes:%d} Deferred{Pulls:%d Trashes:%d}", len(cs.Pulls), len(cs.Trashes), cs.PullsDeferred, cs.TrashesDeferred)
}
