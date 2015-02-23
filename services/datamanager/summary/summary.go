/* Computes Summary based on data read from API server. */

package summary

import (
	"encoding/gob"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"git.curoverse.com/arvados.git/services/datamanager/loggerutil"
	"log"
	"os"
)

type ReplicationSummary struct {
	CollectionBlocksNotInKeep  map[blockdigest.BlockDigest]struct{}
	UnderReplicatedBlocks      map[blockdigest.BlockDigest]struct{}
	OverReplicatedBlocks       map[blockdigest.BlockDigest]struct{}
	CorrectlyReplicatedBlocks  map[blockdigest.BlockDigest]struct{}
	KeepBlocksNotInCollections map[blockdigest.BlockDigest]struct{}
}

type serializedData struct {
	ReadCollections collection.ReadCollections
	KeepServerInfo  keep.ReadServers
}

var (
	writeDataTo  string
	readDataFrom string
)

func init() {
	flag.StringVar(&writeDataTo,
		"write-data-to",
		"",
		"Write summary of data received to this file. Used for development only.")
	flag.StringVar(&readDataFrom,
		"read-data-from",
		"",
		"Avoid network i/o and read summary data from this file instead. Used for development only.")
}

// Writes data we've read to a file.
//
// This is useful for development, so that we don't need to read all our data from the network every time we tweak something.
//
// This should not be used outside of development, since you'll be
// working with stale data.
func MaybeWriteData(arvLogger *logger.Logger,
	readCollections collection.ReadCollections,
	keepServerInfo keep.ReadServers) bool {
	if writeDataTo == "" {
		return false
	} else {
		summaryFile, err := os.Create(writeDataTo)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to open %s: %v", writeDataTo, err))
		}
		defer summaryFile.Close()

		enc := gob.NewEncoder(summaryFile)
		data := serializedData{
			ReadCollections: readCollections,
			KeepServerInfo:  keepServerInfo}
		err = enc.Encode(data)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to write summary data: %v", err))
		}
		log.Printf("Wrote summary data to: %s", writeDataTo)
		return true
	}
}

// Reads data that we've read to a file.
//
// This is useful for development, so that we don't need to read all our data from the network every time we tweak something.
//
// This should not be used outside of development, since you'll be
// working with stale data.
func MaybeReadData(arvLogger *logger.Logger,
	readCollections *collection.ReadCollections,
	keepServerInfo *keep.ReadServers) bool {
	if readDataFrom == "" {
		return false
	} else {
		summaryFile, err := os.Open(readDataFrom)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to open %s: %v", readDataFrom, err))
		}
		defer summaryFile.Close()

		dec := gob.NewDecoder(summaryFile)
		data := serializedData{}
		err = dec.Decode(&data)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to read summary data: %v", err))
		}
		*readCollections = data.ReadCollections
		*keepServerInfo = data.KeepServerInfo
		log.Printf("Read summary data from: %s", readDataFrom)
		return true
	}
}

func SummarizeReplication(arvLogger *logger.Logger,
	readCollections collection.ReadCollections,
	keepServerInfo keep.ReadServers) (rs ReplicationSummary) {
	rs.CollectionBlocksNotInKeep = make(map[blockdigest.BlockDigest]struct{})
	rs.UnderReplicatedBlocks = make(map[blockdigest.BlockDigest]struct{})
	rs.OverReplicatedBlocks = make(map[blockdigest.BlockDigest]struct{})
	rs.CorrectlyReplicatedBlocks = make(map[blockdigest.BlockDigest]struct{})
	rs.KeepBlocksNotInCollections = make(map[blockdigest.BlockDigest]struct{})

	for block, requestedReplication := range readCollections.BlockToReplication {
		actualReplication := len(keepServerInfo.BlockToServers[block])
		if actualReplication == 0 {
			rs.CollectionBlocksNotInKeep[block] = struct{}{}
		} else if actualReplication < requestedReplication {
			rs.UnderReplicatedBlocks[block] = struct{}{}
		} else if actualReplication > requestedReplication {
			rs.OverReplicatedBlocks[block] = struct{}{}
		} else {
			rs.CorrectlyReplicatedBlocks[block] = struct{}{}
		}
	}

	for block, _ := range keepServerInfo.BlockToServers {
		if 0 == readCollections.BlockToReplication[block] {
			rs.KeepBlocksNotInCollections[block] = struct{}{}
		}
	}

	return rs
}
