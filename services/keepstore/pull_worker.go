// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
)

type PullListItem struct {
	Locator   string   `json:"locator"`
	Servers   []string `json:"servers"`
	MountUUID string   `json:"mount_uuid"` // Destination mount, or "" for "anywhere"
}

type puller struct {
	keepstore  *keepstore
	todo       []PullListItem
	cond       *sync.Cond // lock guards todo accesses; cond broadcasts when todo becomes non-empty
	inprogress atomic.Int64
}

func newPuller(ctx context.Context, keepstore *keepstore, reg *prometheus.Registry) *puller {
	p := &puller{
		keepstore: keepstore,
		cond:      sync.NewCond(&sync.Mutex{}),
	}
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "pull_queue_pending_entries",
			Help:      "Number of queued pull requests",
		},
		func() float64 {
			p.cond.L.Lock()
			defer p.cond.L.Unlock()
			return float64(len(p.todo))
		},
	))
	reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "pull_queue_inprogress_entries",
			Help:      "Number of pull requests in progress",
		},
		func() float64 {
			return float64(p.inprogress.Load())
		},
	))
	if len(p.keepstore.mountsW) == 0 {
		keepstore.logger.Infof("not running pull worker because there are no writable volumes")
		return p
	}
	for i := 0; i < 1 || i < keepstore.cluster.Collections.BlobReplicateConcurrency; i++ {
		go p.runWorker(ctx)
	}
	return p
}

func (p *puller) SetPullList(newlist []PullListItem) {
	p.cond.L.Lock()
	p.todo = newlist
	p.cond.L.Unlock()
	p.cond.Broadcast()
}

func (p *puller) runWorker(ctx context.Context) {
	if len(p.keepstore.mountsW) == 0 {
		p.keepstore.logger.Infof("not running pull worker because there are no writable volumes")
		return
	}
	c, err := arvados.NewClientFromConfig(p.keepstore.cluster)
	if err != nil {
		p.keepstore.logger.Errorf("error setting up pull worker: %s", err)
		return
	}
	c.AuthToken = "keepstore-token-used-for-pulling-data-from-same-cluster"
	ac, err := arvadosclient.New(c)
	if err != nil {
		p.keepstore.logger.Errorf("error setting up pull worker: %s", err)
		return
	}
	keepClient := &keepclient.KeepClient{
		Arvados:       ac,
		Want_replicas: 1,
		DiskCacheSize: keepclient.DiskCacheDisabled,
	}
	// Ensure the loop below wakes up and returns when ctx
	// cancels, even if pull list is empty.
	go func() {
		<-ctx.Done()
		p.cond.Broadcast()
	}()
	for {
		p.cond.L.Lock()
		for len(p.todo) == 0 && ctx.Err() == nil {
			p.cond.Wait()
		}
		if ctx.Err() != nil {
			return
		}
		item := p.todo[0]
		p.todo = p.todo[1:]
		p.inprogress.Add(1)
		p.cond.L.Unlock()

		func() {
			defer p.inprogress.Add(-1)

			logger := p.keepstore.logger.WithField("locator", item.Locator)

			li, err := parseLocator(item.Locator)
			if err != nil {
				logger.Warn("ignoring pull request for invalid locator")
				return
			}

			var dst *mount
			if item.MountUUID != "" {
				dst = p.keepstore.mounts[item.MountUUID]
				if dst == nil {
					logger.Warnf("ignoring pull list entry for nonexistent mount %s", item.MountUUID)
					return
				} else if !dst.AllowWrite {
					logger.Warnf("ignoring pull list entry for readonly mount %s", item.MountUUID)
					return
				}
			} else {
				dst = p.keepstore.rendezvous(item.Locator, p.keepstore.mountsW)[0]
			}

			serviceRoots := make(map[string]string)
			for _, addr := range item.Servers {
				serviceRoots[addr] = addr
			}
			keepClient.SetServiceRoots(serviceRoots, nil, nil)

			signedLocator := p.keepstore.signLocator(c.AuthToken, item.Locator)

			buf := bytes.NewBuffer(nil)
			_, err = keepClient.BlockRead(ctx, arvados.BlockReadOptions{
				Locator: signedLocator,
				WriteTo: buf,
			})
			if err != nil {
				logger.WithError(err).Warnf("error pulling data from remote servers (%s)", item.Servers)
				return
			}
			err = dst.BlockWrite(ctx, li.hash, buf.Bytes())
			if err != nil {
				logger.WithError(err).Warnf("error writing data to %s", dst.UUID)
				return
			}
			logger.Info("block pulled")
		}()
	}
}
