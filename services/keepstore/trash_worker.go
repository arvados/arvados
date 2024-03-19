// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/prometheus/client_golang/prometheus"
)

type TrashListItem struct {
	Locator    string `json:"locator"`
	BlockMtime int64  `json:"block_mtime"`
	MountUUID  string `json:"mount_uuid"` // Target mount, or "" for "everywhere"
}

type trasher struct {
	keepstore  *keepstore
	todo       []TrashListItem
	cond       *sync.Cond // lock guards todo accesses; cond broadcasts when todo becomes non-empty
	inprogress atomic.Int64
}

func newTrasher(ctx context.Context, keepstore *keepstore, reg *prometheus.Registry) *trasher {
	t := &trasher{
		keepstore: keepstore,
		cond:      sync.NewCond(&sync.Mutex{}),
	}
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "trash_queue_pending_entries",
			Help:      "Number of queued trash requests",
		},
		func() float64 {
			t.cond.L.Lock()
			defer t.cond.L.Unlock()
			return float64(len(t.todo))
		},
	))
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "trash_queue_inprogress_entries",
			Help:      "Number of trash requests in progress",
		},
		func() float64 {
			return float64(t.inprogress.Load())
		},
	))
	if !keepstore.cluster.Collections.BlobTrash {
		keepstore.logger.Info("not running trash worker because Collections.BlobTrash == false")
		return t
	}

	var mntsAllowTrash []*mount
	for _, mnt := range t.keepstore.mounts {
		if mnt.AllowTrash {
			mntsAllowTrash = append(mntsAllowTrash, mnt)
		}
	}
	if len(mntsAllowTrash) == 0 {
		t.keepstore.logger.Info("not running trash worker because there are no writable or trashable volumes")
	} else {
		for i := 0; i < keepstore.cluster.Collections.BlobTrashConcurrency; i++ {
			go t.runWorker(ctx, mntsAllowTrash)
		}
	}
	return t
}

func (t *trasher) SetTrashList(newlist []TrashListItem) {
	t.cond.L.Lock()
	t.todo = newlist
	t.cond.L.Unlock()
	t.cond.Broadcast()
}

func (t *trasher) runWorker(ctx context.Context, mntsAllowTrash []*mount) {
	go func() {
		<-ctx.Done()
		t.cond.Broadcast()
	}()
	for {
		t.cond.L.Lock()
		for len(t.todo) == 0 && ctx.Err() == nil {
			t.cond.Wait()
		}
		if ctx.Err() != nil {
			t.cond.L.Unlock()
			return
		}
		item := t.todo[0]
		t.todo = t.todo[1:]
		t.inprogress.Add(1)
		t.cond.L.Unlock()

		func() {
			defer t.inprogress.Add(-1)
			logger := t.keepstore.logger.WithField("locator", item.Locator)

			li, err := getLocatorInfo(item.Locator)
			if err != nil {
				logger.Warn("ignoring trash request for invalid locator")
				return
			}

			reqMtime := time.Unix(0, item.BlockMtime)
			if time.Since(reqMtime) < t.keepstore.cluster.Collections.BlobSigningTTL.Duration() {
				logger.Warnf("client asked to delete a %v old block (BlockMtime %d = %v), but my blobSignatureTTL is %v! Skipping.",
					arvados.Duration(time.Since(reqMtime)),
					item.BlockMtime,
					reqMtime,
					t.keepstore.cluster.Collections.BlobSigningTTL)
				return
			}

			var mnts []*mount
			if item.MountUUID == "" {
				mnts = mntsAllowTrash
			} else if mnt := t.keepstore.mounts[item.MountUUID]; mnt == nil {
				logger.Warnf("ignoring trash request for nonexistent mount %s", item.MountUUID)
				return
			} else if !mnt.AllowTrash {
				logger.Warnf("ignoring trash request for readonly mount %s with AllowTrashWhenReadOnly==false", item.MountUUID)
				return
			} else {
				mnts = []*mount{mnt}
			}

			for _, mnt := range mnts {
				logger := logger.WithField("mount", mnt.UUID)
				mtime, err := mnt.Mtime(li.hash)
				if err != nil {
					logger.WithError(err).Error("error getting stored mtime")
					continue
				}
				if !mtime.Equal(reqMtime) {
					logger.Infof("stored mtime (%v) does not match trash list mtime (%v); skipping", mtime, reqMtime)
					continue
				}
				err = mnt.BlockTrash(li.hash)
				if err != nil {
					logger.WithError(err).Info("error trashing block")
					continue
				}
				logger.Info("block trashed")
			}
		}()
	}
}

type trashEmptier struct{}

func newTrashEmptier(ctx context.Context, ks *keepstore, reg *prometheus.Registry) *trashEmptier {
	d := ks.cluster.Collections.BlobTrashCheckInterval.Duration()
	if d <= 0 ||
		!ks.cluster.Collections.BlobTrash ||
		ks.cluster.Collections.BlobDeleteConcurrency <= 0 {
		ks.logger.Infof("not running trash emptier because disabled by config (enabled=%t, interval=%v, concurrency=%d)", ks.cluster.Collections.BlobTrash, d, ks.cluster.Collections.BlobDeleteConcurrency)
		return &trashEmptier{}
	}
	go func() {
		ticker := time.NewTicker(d)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			for _, mnt := range ks.mounts {
				if mnt.KeepMount.AllowTrash {
					mnt.volume.EmptyTrash()
				}
			}
		}
	}()
	return &trashEmptier{}
}
