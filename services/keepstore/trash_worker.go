package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
	"time"
)

/*
	Keepstore initiates trash worker channel goroutine.
	The channel will process trash list.
		For each (next) trash request:
      Delete the block indicated by the trash request Locator
		Repeat
*/

var defaultTrashLifetime int64 = 0

func RunTrashWorker(arv *arvadosclient.ArvadosClient, trashq *WorkQueue) {
	if arv != nil {
		defaultTrashLifetimeMap, err := arv.Discovery("defaultTrashLifetime")
		if err != nil {
			log.Fatalf("Error setting up arvados client %s", err.Error())
		}
		defaultTrashLifetime = int64(defaultTrashLifetimeMap["defaultTrashLifetime"].(float64))
	}

	nextItem := trashq.NextItem
	for item := range nextItem {
		trashRequest := item.(TrashRequest)
		err := TrashItem(trashRequest)
		if err != nil {
			log.Printf("Trash request error for %s: %s", trashRequest, err)
		}
	}
}

/*
	Delete the block indicated by the Locator in TrashRequest.
*/
func TrashItem(trashRequest TrashRequest) (err error) {
	// Verify if the block is to be deleted based on its Mtime
	for _, volume := range KeepVM.Volumes() {
		mtime, err := volume.Mtime(trashRequest.Locator)
		if err == nil {
			if trashRequest.BlockMtime == mtime.Unix() {
				currentTime := time.Now().Unix()
				if (currentTime - trashRequest.BlockMtime) > defaultTrashLifetime {
					err = volume.Delete(trashRequest.Locator)
				}
			}
		}
	}
	return
}
