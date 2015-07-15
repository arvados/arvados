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

type TrashRequest struct {
	Locator    string `json:"locator"`
	BlockMtime int64  `json:"block_mtime"`
}

type TrashList []TrashRequest

func BuildTrashLists(kc *keepclient.KeepClient,
	keepServerInfo *keep.ReadServers,
	keepBlocksNotInCollections BlockSet) (m map[string]TrashList) {

	m = make(map[string]TrashList)

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
				m[srv] = append(m[srv], TrashRequest{Locator: block.String(), BlockMtime: block_on_server.Mtime})
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
	trashLists map[string]TrashList) {
	r := strings.NewReplacer(":", ".")
	for host, list := range trashLists {
		filename := fmt.Sprintf("trash_list.%s", r.Replace(host))
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
