// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"errors"
	"log"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// RunTrashWorker is used by Keepstore to initiate trash worker channel goroutine.
//	The channel will process trash list.
//		For each (next) trash request:
//      Delete the block indicated by the trash request Locator
//		Repeat
//
func RunTrashWorker(volmgr *RRVolumeManager, cluster *arvados.Cluster, trashq *WorkQueue) {
	for item := range trashq.NextItem {
		trashRequest := item.(TrashRequest)
		TrashItem(volmgr, cluster, trashRequest)
		trashq.DoneItem <- struct{}{}
	}
}

// TrashItem deletes the indicated block from every writable volume.
func TrashItem(volmgr *RRVolumeManager, cluster *arvados.Cluster, trashRequest TrashRequest) {
	reqMtime := time.Unix(0, trashRequest.BlockMtime)
	if time.Since(reqMtime) < cluster.Collections.BlobSigningTTL.Duration() {
		log.Printf("WARNING: data manager asked to delete a %v old block %v (BlockMtime %d = %v), but my blobSignatureTTL is %v! Skipping.",
			arvados.Duration(time.Since(reqMtime)),
			trashRequest.Locator,
			trashRequest.BlockMtime,
			reqMtime,
			cluster.Collections.BlobSigningTTL)
		return
	}

	var volumes []*VolumeMount
	if uuid := trashRequest.MountUUID; uuid == "" {
		volumes = volmgr.AllWritable()
	} else if mnt := volmgr.Lookup(uuid, true); mnt == nil {
		log.Printf("warning: trash request for nonexistent mount: %v", trashRequest)
		return
	} else {
		volumes = []*VolumeMount{mnt}
	}

	for _, volume := range volumes {
		mtime, err := volume.Mtime(trashRequest.Locator)
		if err != nil {
			log.Printf("%v Trash(%v): %v", volume, trashRequest.Locator, err)
			continue
		}
		if trashRequest.BlockMtime != mtime.UnixNano() {
			log.Printf("%v Trash(%v): stored mtime %v does not match trash list value %v", volume, trashRequest.Locator, mtime.UnixNano(), trashRequest.BlockMtime)
			continue
		}

		if !cluster.Collections.BlobTrash {
			err = errors.New("skipping because Collections.BlobTrash is false")
		} else {
			err = volume.Trash(trashRequest.Locator)
		}

		if err != nil {
			log.Printf("%v Trash(%v): %v", volume, trashRequest.Locator, err)
		} else {
			log.Printf("%v Trash(%v) OK", volume, trashRequest.Locator)
		}
	}
}
