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
func RunTrashWorker(trashq *WorkQueue) {
	for item := range trashq.NextItem {
		trashRequest := item.(TrashRequest)
		TrashItem(trashRequest)
		trashq.DoneItem <- struct{}{}
	}
}

// TrashItem deletes the indicated block from every writable volume.
func TrashItem(trashRequest TrashRequest) {
	reqMtime := time.Unix(0, trashRequest.BlockMtime)
	if time.Since(reqMtime) < theConfig.BlobSignatureTTL.Duration() {
		log.Printf("WARNING: data manager asked to delete a %v old block %v (BlockMtime %d = %v), but my blobSignatureTTL is %v! Skipping.",
			arvados.Duration(time.Since(reqMtime)),
			trashRequest.Locator,
			trashRequest.BlockMtime,
			reqMtime,
			theConfig.BlobSignatureTTL)
		return
	}

	for _, volume := range KeepVM.AllWritable() {
		mtime, err := volume.Mtime(trashRequest.Locator)
		if err != nil {
			log.Printf("%v Delete(%v): %v", volume, trashRequest.Locator, err)
			continue
		}
		if trashRequest.BlockMtime != mtime.UnixNano() {
			log.Printf("%v Delete(%v): stored mtime %v does not match trash list value %v", volume, trashRequest.Locator, mtime.UnixNano(), trashRequest.BlockMtime)
			continue
		}

		if !theConfig.EnableDelete {
			err = errors.New("did not delete block because EnableDelete is false")
		} else {
			err = volume.Trash(trashRequest.Locator)
		}

		if err != nil {
			log.Printf("%v Delete(%v): %v", volume, trashRequest.Locator, err)
		} else {
			log.Printf("%v Delete(%v) OK", volume, trashRequest.Locator)
		}
	}
}
