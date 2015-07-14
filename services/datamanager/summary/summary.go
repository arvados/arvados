// Summarizes Collection Data and Keep Server Contents.
package summary

// TODO(misha): Check size of blocks as well as their digest.

import (
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"sort"
)

type BlockSet map[blockdigest.DigestWithSize]struct{}

// Adds a single block to the set.
func (bs BlockSet) Insert(digest blockdigest.DigestWithSize) {
	bs[digest] = struct{}{}
}

// Adds a set of blocks to the set.
func (bs BlockSet) Union(obs BlockSet) {
	for k, v := range obs {
		bs[k] = v
	}
}

// We use the collection index to save space. To convert to and from
// the uuid, use collection.ReadCollections' fields
// CollectionIndexToUuid and CollectionUuidToIndex.
type CollectionIndexSet map[int]struct{}

// Adds a single collection to the set. The collection is specified by
// its index.
func (cis CollectionIndexSet) Insert(collectionIndex int) {
	cis[collectionIndex] = struct{}{}
}

func (bs BlockSet) ToCollectionIndexSet(
	readCollections collection.ReadCollections,
	collectionIndexSet *CollectionIndexSet) {
	for block := range bs {
		for _, collectionIndex := range readCollections.BlockToCollectionIndices[block] {
			collectionIndexSet.Insert(collectionIndex)
		}
	}
}

// Keeps track of the requested and actual replication levels.
// Currently this is only used for blocks but could easily be used for
// collections as well.
type ReplicationLevels struct {
	// The requested replication level.
	// For Blocks this is the maximum replication level among all the
	// collections this block belongs to.
	Requested int

	// The actual number of keep servers this is on.
	Actual int
}

// Maps from replication levels to their blocks.
type ReplicationLevelBlockSetMap map[ReplicationLevels]BlockSet

// An individual entry from ReplicationLevelBlockSetMap which only reports the number of blocks, not which blocks.
type ReplicationLevelBlockCount struct {
	Levels ReplicationLevels
	Count  int
}

// An ordered list of ReplicationLevelBlockCount useful for reporting.
type ReplicationLevelBlockSetSlice []ReplicationLevelBlockCount

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

// Gets the BlockSet for a given set of ReplicationLevels, creating it
// if it doesn't already exist.
func (rlbs ReplicationLevelBlockSetMap) GetOrCreate(
	repLevels ReplicationLevels) (bs BlockSet) {
	bs, exists := rlbs[repLevels]
	if !exists {
		bs = make(BlockSet)
		rlbs[repLevels] = bs
	}
	return
}

// Adds a block to the set for a given replication level.
func (rlbs ReplicationLevelBlockSetMap) Insert(
	repLevels ReplicationLevels,
	block blockdigest.DigestWithSize) {
	rlbs.GetOrCreate(repLevels).Insert(block)
}

// Adds a set of blocks to the set for a given replication level.
func (rlbs ReplicationLevelBlockSetMap) Union(
	repLevels ReplicationLevels,
	bs BlockSet) {
	rlbs.GetOrCreate(repLevels).Union(bs)
}

// Outputs a sorted list of ReplicationLevelBlockCounts.
func (rlbs ReplicationLevelBlockSetMap) Counts() (
	sorted ReplicationLevelBlockSetSlice) {
	sorted = make(ReplicationLevelBlockSetSlice, len(rlbs))
	i := 0
	for levels, set := range rlbs {
		sorted[i] = ReplicationLevelBlockCount{Levels: levels, Count: len(set)}
		i++
	}
	sort.Sort(sorted)
	return
}

// Implemented to meet sort.Interface
func (rlbss ReplicationLevelBlockSetSlice) Len() int {
	return len(rlbss)
}

// Implemented to meet sort.Interface
func (rlbss ReplicationLevelBlockSetSlice) Less(i, j int) bool {
	return rlbss[i].Levels.Requested < rlbss[j].Levels.Requested ||
		(rlbss[i].Levels.Requested == rlbss[j].Levels.Requested &&
			rlbss[i].Levels.Actual < rlbss[j].Levels.Actual)
}

// Implemented to meet sort.Interface
func (rlbss ReplicationLevelBlockSetSlice) Swap(i, j int) {
	rlbss[i], rlbss[j] = rlbss[j], rlbss[i]
}

func (rs ReplicationSummary) ComputeCounts() (rsc ReplicationSummaryCounts) {
	// TODO(misha): Consider rewriting this method to iterate through
	// the fields using reflection, instead of explictily listing the
	// fields as we do now.
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

func BucketReplication(readCollections collection.ReadCollections,
	keepServerInfo keep.ReadServers) (rlbsm ReplicationLevelBlockSetMap) {
	rlbsm = make(ReplicationLevelBlockSetMap)

	for block, requestedReplication := range readCollections.BlockToDesiredReplication {
		rlbsm.Insert(
			ReplicationLevels{
				Requested: requestedReplication,
				Actual:    len(keepServerInfo.BlockToServers[block])},
			block)
	}

	for block, servers := range keepServerInfo.BlockToServers {
		if 0 == readCollections.BlockToDesiredReplication[block] {
			rlbsm.Insert(
				ReplicationLevels{Requested: 0, Actual: len(servers)},
				block)
		}
	}
	return
}

func (rlbsm ReplicationLevelBlockSetMap) SummarizeBuckets(
	readCollections collection.ReadCollections) (
	rs ReplicationSummary) {
	rs.CollectionBlocksNotInKeep = make(BlockSet)
	rs.UnderReplicatedBlocks = make(BlockSet)
	rs.OverReplicatedBlocks = make(BlockSet)
	rs.CorrectlyReplicatedBlocks = make(BlockSet)
	rs.KeepBlocksNotInCollections = make(BlockSet)

	rs.CollectionsNotFullyInKeep = make(CollectionIndexSet)
	rs.UnderReplicatedCollections = make(CollectionIndexSet)
	rs.OverReplicatedCollections = make(CollectionIndexSet)
	rs.CorrectlyReplicatedCollections = make(CollectionIndexSet)

	for levels, bs := range rlbsm {
		if levels.Actual == 0 {
			rs.CollectionBlocksNotInKeep.Union(bs)
		} else if levels.Requested == 0 {
			rs.KeepBlocksNotInCollections.Union(bs)
		} else if levels.Actual < levels.Requested {
			rs.UnderReplicatedBlocks.Union(bs)
		} else if levels.Actual > levels.Requested {
			rs.OverReplicatedBlocks.Union(bs)
		} else {
			rs.CorrectlyReplicatedBlocks.Union(bs)
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

	return
}
