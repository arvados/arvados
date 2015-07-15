// Code for generating trash lists
package summary

import (
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"log"
	"time"
)

func BuildTrashLists(kc *keepclient.KeepClient,
	keepServerInfo *keep.ReadServers,
	keepBlocksNotInCollections BlockSet) (m map[string]keep.TrashList) {

	// Servers that are writeable
	writableServers := map[string]struct{}{}
	for _, url := range kc.WritableLocalRoots() {
		writableServers[url] = struct{}{}
	}

	_ttl, err := kc.Arvados.Discovery("blobSignatureTtl")
	if err != nil {
		log.Printf("Failed to get blobSignatureTtl: %v", err)
		return map[string]keep.TrashList{}
	}

	ttl := int64(_ttl.(float64))

	// expire unreferenced blocks more than "ttl" seconds old.
	expiry := time.Now().UTC().Unix() - ttl

	return BuildTrashListsInternal(writableServers, keepServerInfo, expiry, keepBlocksNotInCollections)
}

func BuildTrashListsInternal(writableServers map[string]struct{},
	keepServerInfo *keep.ReadServers,
	expiry int64,
	keepBlocksNotInCollections BlockSet) (m map[string]keep.TrashList) {

	m = make(map[string]keep.TrashList)

	for block, _ := range keepBlocksNotInCollections {
		for _, block_on_server := range keepServerInfo.BlockToServers[block] {
			if block_on_server.Mtime < expiry {
				// block is older than expire cutoff
				srv := keepServerInfo.KeepServerIndexToAddress[block_on_server.ServerIndex].String()

				_, writable := writableServers[srv]

				if writable {
					m[srv] = append(m[srv], keep.TrashRequest{Locator: block.Digest.String(), BlockMtime: block_on_server.Mtime})
				}
			}
		}
	}
	return

}
