// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"errors"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

// RunTrashWorker processes the trash request queue.
func RunTrashWorker(volmgr *RRVolumeManager, logger logrus.FieldLogger, cluster *arvados.Cluster, trashq *WorkQueue) {
	for item := range trashq.NextItem {
		trashRequest := item.(TrashRequest)
		TrashItem(volmgr, logger, cluster, trashRequest)
		trashq.DoneItem <- struct{}{}
	}
}

// TrashItem deletes the indicated block from every writable volume.
func TrashItem(volmgr *RRVolumeManager, logger logrus.FieldLogger, cluster *arvados.Cluster, trashRequest TrashRequest) {
	reqMtime := time.Unix(0, trashRequest.BlockMtime)
	if time.Since(reqMtime) < cluster.Collections.BlobSigningTTL.Duration() {
		logger.Warnf("client asked to delete a %v old block %v (BlockMtime %d = %v), but my blobSignatureTTL is %v! Skipping.",
			arvados.Duration(time.Since(reqMtime)),
			trashRequest.Locator,
			trashRequest.BlockMtime,
			reqMtime,
			cluster.Collections.BlobSigningTTL)
		return
	}

	var volumes []*VolumeMount
	if uuid := trashRequest.MountUUID; uuid == "" {
		volumes = volmgr.Mounts()
	} else if mnt := volmgr.Lookup(uuid, false); mnt == nil {
		logger.Warnf("trash request for nonexistent mount: %v", trashRequest)
		return
	} else if !mnt.KeepMount.AllowTrash {
		logger.Warnf("trash request for mount with ReadOnly=true, AllowTrashWhenReadOnly=false: %v", trashRequest)
	} else {
		volumes = []*VolumeMount{mnt}
	}

	for _, volume := range volumes {
		mtime, err := volume.Mtime(trashRequest.Locator)
		if err != nil {
			logger.WithError(err).Errorf("%v Trash(%v)", volume, trashRequest.Locator)
			continue
		}
		if trashRequest.BlockMtime != mtime.UnixNano() {
			logger.Infof("%v Trash(%v): stored mtime %v does not match trash list value %v; skipping", volume, trashRequest.Locator, mtime.UnixNano(), trashRequest.BlockMtime)
			continue
		}

		if !cluster.Collections.BlobTrash {
			err = errors.New("skipping because Collections.BlobTrash is false")
		} else {
			err = volume.Trash(trashRequest.Locator)
		}

		if err != nil {
			logger.WithError(err).Errorf("%v Trash(%v)", volume, trashRequest.Locator)
		} else {
			logger.Infof("%v Trash(%v) OK", volume, trashRequest.Locator)
		}
	}
}
