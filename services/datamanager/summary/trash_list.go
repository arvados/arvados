// Code for generating trash lists
package summary

import (
	"encoding/json"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"git.curoverse.com/arvados.git/services/datamanager/loggerutil"
	"log"
	"os"
	"strings"
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

	m = make(map[string]keep.TrashList)

	_ttl, err := kc.Arvados.Discovery("blobSignatureTtl")
	if err != nil {
		log.Printf("Failed to get blobSignatureTtl: %v", err)
		return
	}

	ttl := int64(_ttl.(float64))

	// expire unreferenced blocks more than "ttl" seconds old.
	expiry := time.Now().UTC().Unix() - ttl

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

// Writes each pull list to a file.
// The filename is based on the hostname.
//
// This is just a hack for prototyping, it is not expected to be used
// in production.
func WriteTrashLists(arvLogger *logger.Logger,
	trashLists map[string]keep.TrashList) {
	r := strings.NewReplacer(":", ".")
	for host, list := range trashLists {
		filename := fmt.Sprintf("trash_list.%s", r.Replace(RemoveProtocolPrefix(host)))
		trashListFile, err := os.Create(filename)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to open %s: %v", filename, err))
		}
		defer trashListFile.Close()

		enc := json.NewEncoder(trashListFile)
		err = enc.Encode(list)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to write trash list to %s: %v", filename, err))
		}
		log.Printf("Wrote trash list to %s.", filename)
	}
}
