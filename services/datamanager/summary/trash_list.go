// Code for generating trash lists
package summary

import (
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"time"
)

func BuildTrashLists(kc *keepclient.KeepClient,
	keepServerInfo *keep.ReadServers,
	keepBlocksNotInCollections BlockSet) (m map[string]keep.TrashList, err error) {

	// Servers that are writeable
	writableServers := map[string]struct{}{}
	for _, url := range kc.WritableLocalRoots() {
		writableServers[url] = struct{}{}
	}

	_ttl, err := kc.Arvados.Discovery("blobSignatureTtl")
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to get blobSignatureTtl, can't build trash lists: %v", err))
	}

	ttl := int64(_ttl.(float64))

	// expire unreferenced blocks more than "ttl" seconds old.
	expiry := time.Now().UTC().Unix() - ttl

	return buildTrashListsInternal(writableServers, keepServerInfo, expiry, keepBlocksNotInCollections), nil
}

func buildTrashListsInternal(writableServers map[string]struct{},
	keepServerInfo *keep.ReadServers,
	expiry int64,
	keepBlocksNotInCollections BlockSet) (m map[string]keep.TrashList) {

	m = make(map[string]keep.TrashList)

	for block := range keepBlocksNotInCollections {
		for _, block_on_server := range keepServerInfo.BlockToServers[block] {
			if block_on_server.Mtime >= expiry {
				continue
			}

			// block is older than expire cutoff
			srv := keepServerInfo.KeepServerIndexToAddress[block_on_server.ServerIndex].String()

			if _, writable := writableServers[srv]; !writable {
				continue
			}

			m[srv] = append(m[srv], keep.TrashRequest{Locator: block.Digest.String(), BlockMtime: block_on_server.Mtime})
		}
	}
	return

}
