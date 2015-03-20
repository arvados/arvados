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

var DEFAULT_TRASH_LIFE_TIME int64 = 1209600 // Use 2 weeks for now

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
				if (currentTime - trashRequest.BlockMtime) > DEFAULT_TRASH_LIFE_TIME {
					err = volume.Delete(trashRequest.Locator)
				}
			}
		}
	}
	return
}
