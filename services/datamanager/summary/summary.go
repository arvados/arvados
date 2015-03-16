// Summarizes Collection Data and Keep Server Contents.
package summary

// TODO(misha): Check size of blocks as well as their digest.

import (
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
)

type BlockSet map[blockdigest.BlockDigest]struct{}

func (bs BlockSet) Insert(digest blockdigest.BlockDigest) {
	bs[digest] = struct{}{}
}

// We use the collection index to save space. To convert to and from
// the uuid, use collection.ReadCollections' fields
// CollectionIndexToUuid and CollectionUuidToIndex.
type CollectionIndexSet map[int]struct{}

func (cis CollectionIndexSet) Insert(collectionIndex int) {
	cis[collectionIndex] = struct{}{}
}

func (bs BlockSet) ToCollectionIndexSet(
	readCollections collection.ReadCollections,
	collectionIndexSet *CollectionIndexSet) {
	for block := range bs {
		for _,collectionIndex := range readCollections.BlockToCollectionIndices[block] {
			collectionIndexSet.Insert(collectionIndex)
		}
	}
}

type ReplicationSummary struct {
	CollectionBlocksNotInKeep  BlockSet
	UnderReplicatedBlocks      BlockSet
	OverReplicatedBlocks       BlockSet
	CorrectlyReplicatedBlocks  BlockSet
	KeepBlocksNotInCollections BlockSet

	CollectionsNotFullyInKeep      CollectionIndexSet
	UnderReplicatedCollections     CollectionIndexSet
	OverReplicatedCollections      CollectionIndexSet
	CorrectlyReplicatedCollections CollectionIndexSet
}

// This struct counts the elements in each set in ReplicationSummary.
type ReplicationSummaryCounts struct {
	CollectionBlocksNotInKeep      int
	UnderReplicatedBlocks          int
	OverReplicatedBlocks           int
	CorrectlyReplicatedBlocks      int
	KeepBlocksNotInCollections     int
	CollectionsNotFullyInKeep      int
	UnderReplicatedCollections     int
	OverReplicatedCollections      int
	CorrectlyReplicatedCollections int
}

func (rs ReplicationSummary) ComputeCounts() (rsc ReplicationSummaryCounts) {
	// TODO(misha): Consider replacing this brute-force approach by
	// iterating through the fields using reflection.
	rsc.CollectionBlocksNotInKeep = len(rs.CollectionBlocksNotInKeep)
	rsc.UnderReplicatedBlocks = len(rs.UnderReplicatedBlocks)
	rsc.OverReplicatedBlocks = len(rs.OverReplicatedBlocks)
	rsc.CorrectlyReplicatedBlocks = len(rs.CorrectlyReplicatedBlocks)
	rsc.KeepBlocksNotInCollections = len(rs.KeepBlocksNotInCollections)
	rsc.CollectionsNotFullyInKeep = len(rs.CollectionsNotFullyInKeep)
	rsc.UnderReplicatedCollections = len(rs.UnderReplicatedCollections)
	rsc.OverReplicatedCollections = len(rs.OverReplicatedCollections)
	rsc.CorrectlyReplicatedCollections = len(rs.CorrectlyReplicatedCollections)
	return rsc
}

func (rsc ReplicationSummaryCounts) PrettyPrint() string {
	return fmt.Sprintf("Replication Block Counts:"+
		"\n Missing From Keep: %d, "+
		"\n Under Replicated: %d, "+
		"\n Over Replicated: %d, "+
		"\n Replicated Just Right: %d, "+
		"\n Not In Any Collection: %d. "+
		"\nReplication Collection Counts:"+
		"\n Missing From Keep: %d, "+
		"\n Under Replicated: %d, "+
		"\n Over Replicated: %d, "+
		"\n Replicated Just Right: %d.",
		rsc.CollectionBlocksNotInKeep,
		rsc.UnderReplicatedBlocks,
		rsc.OverReplicatedBlocks,
		rsc.CorrectlyReplicatedBlocks,
		rsc.KeepBlocksNotInCollections,
		rsc.CollectionsNotFullyInKeep,
		rsc.UnderReplicatedCollections,
		rsc.OverReplicatedCollections,
		rsc.CorrectlyReplicatedCollections)
}

func SummarizeReplication(readCollections collection.ReadCollections,
	keepServerInfo keep.ReadServers) (rs ReplicationSummary) {
	rs.CollectionBlocksNotInKeep = make(BlockSet)
	rs.UnderReplicatedBlocks = make(BlockSet)
	rs.OverReplicatedBlocks = make(BlockSet)
	rs.CorrectlyReplicatedBlocks = make(BlockSet)
	rs.KeepBlocksNotInCollections = make(BlockSet)
	rs.CollectionsNotFullyInKeep = make(CollectionIndexSet)
	rs.UnderReplicatedCollections = make(CollectionIndexSet)
	rs.OverReplicatedCollections = make(CollectionIndexSet)
	rs.CorrectlyReplicatedCollections = make(CollectionIndexSet)

	for block, requestedReplication := range readCollections.BlockToReplication {
		actualReplication := len(keepServerInfo.BlockToServers[block])
		if actualReplication == 0 {
			rs.CollectionBlocksNotInKeep.Insert(block)
		} else if actualReplication < requestedReplication {
			rs.UnderReplicatedBlocks.Insert(block)
		} else if actualReplication > requestedReplication {
			rs.OverReplicatedBlocks.Insert(block)
		} else {
			rs.CorrectlyReplicatedBlocks.Insert(block)
		}
	}

	for block, _ := range keepServerInfo.BlockToServers {
		if 0 == readCollections.BlockToReplication[block] {
			rs.KeepBlocksNotInCollections.Insert(block)
		}
	}

	rs.CollectionBlocksNotInKeep.ToCollectionIndexSet(readCollections,
		&rs.CollectionsNotFullyInKeep)
	// Since different collections can specify different replication
	// levels, the fact that a block is under-replicated does not imply
	// that all collections that it belongs to are under-replicated, but
	// we'll ignore that for now.
	// TODO(misha): Fix this and report the correct set of collections.
	rs.UnderReplicatedBlocks.ToCollectionIndexSet(readCollections,
		&rs.UnderReplicatedCollections)
	rs.OverReplicatedBlocks.ToCollectionIndexSet(readCollections,
		&rs.OverReplicatedCollections)

	for i := range readCollections.CollectionIndexToUuid {
		if _, notInKeep := rs.CollectionsNotFullyInKeep[i]; notInKeep {
		} else if _, underReplicated := rs.UnderReplicatedCollections[i]; underReplicated {
		} else if _, overReplicated := rs.OverReplicatedCollections[i]; overReplicated {
		} else {
			rs.CorrectlyReplicatedCollections.Insert(i)
		}
	}

	return rs
}
