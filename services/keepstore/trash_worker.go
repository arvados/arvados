package main

import (
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

func RunTrashWorker(trashq *WorkQueue) {
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
				if time.Duration(currentTime-trashRequest.BlockMtime)*time.Second >= permission_ttl {
					err = volume.Delete(trashRequest.Locator)
				}
			}
		}
	}
	return
}
