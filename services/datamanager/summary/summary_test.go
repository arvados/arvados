package summary

import (
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"reflect"
	"sort"
	"testing"
)

func BlockSetFromSlice(digests []int) (bs BlockSet) {
	bs = make(BlockSet)
	for _, digest := range digests {
		bs.Insert(blockdigest.MakeTestBlockDigest(digest))
	}
	return
}

func CollectionIndexSetFromSlice(indices []int) (cis CollectionIndexSet) {
	cis = make(CollectionIndexSet)
	for _, index := range indices {
		cis.Insert(index)
	}
	return
}

func (cis CollectionIndexSet) ToSlice() (ints []int) {
	ints = make([]int, len(cis))
	i := 0
	for collectionIndex := range cis {
		ints[i] = collectionIndex
		i++
	}
	sort.Ints(ints)
	return
}

// Helper method to meet interface expected by older tests.
func SummarizeReplication(readCollections collection.ReadCollections,
	keepServerInfo keep.ReadServers) (rs ReplicationSummary) {
	return BucketReplication(readCollections, keepServerInfo).
		SummarizeBuckets(readCollections)
}

// Takes a map from block digest to replication level and represents
// it in a keep.ReadServers structure.
func SpecifyReplication(digestToReplication map[int]int) (rs keep.ReadServers) {
	rs.BlockToServers = make(map[blockdigest.BlockDigest][]keep.BlockServerInfo)
	for digest, replication := range digestToReplication {
		rs.BlockToServers[blockdigest.MakeTestBlockDigest(digest)] =
			make([]keep.BlockServerInfo, replication)
	}
	return
}

// Verifies that
// blocks.ToCollectionIndexSet(rc.BlockToCollectionIndices) returns
// expectedCollections.
func VerifyToCollectionIndexSet(
	t *testing.T,
	blocks []int,
	blockToCollectionIndices map[int][]int,
	expectedCollections []int) {

	expected := CollectionIndexSetFromSlice(expectedCollections)

	rc := collection.ReadCollections{
		BlockToCollectionIndices: map[blockdigest.BlockDigest][]int{},
	}
	for digest, indices := range blockToCollectionIndices {
		rc.BlockToCollectionIndices[blockdigest.MakeTestBlockDigest(digest)] = indices
	}

	returned := make(CollectionIndexSet)
	BlockSetFromSlice(blocks).ToCollectionIndexSet(rc, &returned)

	if !reflect.DeepEqual(returned, expected) {
		t.Errorf("Expected %v.ToCollectionIndexSet(%v) to return \n %v \n but instead received \n %v",
			blocks,
			blockToCollectionIndices,
			expectedCollections,
			returned.ToSlice())
	}
}

func TestToCollectionIndexSet(t *testing.T) {
	VerifyToCollectionIndexSet(t, []int{6}, map[int][]int{6: []int{0}}, []int{0})
	VerifyToCollectionIndexSet(t, []int{4}, map[int][]int{4: []int{1}}, []int{1})
	VerifyToCollectionIndexSet(t, []int{4}, map[int][]int{4: []int{1, 9}}, []int{1, 9})
	VerifyToCollectionIndexSet(t, []int{5, 6},
		map[int][]int{5: []int{2, 3}, 6: []int{3, 4}},
		[]int{2, 3, 4})
	VerifyToCollectionIndexSet(t, []int{5, 6},
		map[int][]int{5: []int{8}, 6: []int{4}},
		[]int{4, 8})
	VerifyToCollectionIndexSet(t, []int{6}, map[int][]int{5: []int{0}}, []int{})
}

func TestSimpleSummary(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{ReplicationLevel: 1, Blocks: []int{1, 2}},
	})
	rc.Summarize(nil)
	cIndex := rc.CollectionIndicesForTesting()

	keepInfo := SpecifyReplication(map[int]int{1: 1, 2: 1})

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSet{},
		UnderReplicatedBlocks:      BlockSet{},
		OverReplicatedBlocks:       BlockSet{},
		CorrectlyReplicatedBlocks:  BlockSetFromSlice([]int{1, 2}),
		KeepBlocksNotInCollections: BlockSet{},

		CollectionsNotFullyInKeep:      CollectionIndexSet{},
		UnderReplicatedCollections:     CollectionIndexSet{},
		OverReplicatedCollections:      CollectionIndexSet{},
		CorrectlyReplicatedCollections: CollectionIndexSetFromSlice([]int{cIndex[0]}),
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like %+v but instead it is %+v", expectedSummary, returnedSummary)
	}
}

func TestMissingBlock(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{ReplicationLevel: 1, Blocks: []int{1, 2}},
	})
	rc.Summarize(nil)
	cIndex := rc.CollectionIndicesForTesting()

	keepInfo := SpecifyReplication(map[int]int{1: 1})

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSetFromSlice([]int{2}),
		UnderReplicatedBlocks:      BlockSet{},
		OverReplicatedBlocks:       BlockSet{},
		CorrectlyReplicatedBlocks:  BlockSetFromSlice([]int{1}),
		KeepBlocksNotInCollections: BlockSet{},

		CollectionsNotFullyInKeep:      CollectionIndexSetFromSlice([]int{cIndex[0]}),
		UnderReplicatedCollections:     CollectionIndexSet{},
		OverReplicatedCollections:      CollectionIndexSet{},
		CorrectlyReplicatedCollections: CollectionIndexSet{},
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like %+v but instead it is %+v",
			expectedSummary,
			returnedSummary)
	}
}

func TestUnderAndOverReplicatedBlocks(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{ReplicationLevel: 2, Blocks: []int{1, 2}},
	})
	rc.Summarize(nil)
	cIndex := rc.CollectionIndicesForTesting()

	keepInfo := SpecifyReplication(map[int]int{1: 1, 2: 3})

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSet{},
		UnderReplicatedBlocks:      BlockSetFromSlice([]int{1}),
		OverReplicatedBlocks:       BlockSetFromSlice([]int{2}),
		CorrectlyReplicatedBlocks:  BlockSet{},
		KeepBlocksNotInCollections: BlockSet{},

		CollectionsNotFullyInKeep:      CollectionIndexSet{},
		UnderReplicatedCollections:     CollectionIndexSetFromSlice([]int{cIndex[0]}),
		OverReplicatedCollections:      CollectionIndexSetFromSlice([]int{cIndex[0]}),
		CorrectlyReplicatedCollections: CollectionIndexSet{},
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like %+v but instead it is %+v",
			expectedSummary,
			returnedSummary)
	}
}

func TestMixedReplication(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{ReplicationLevel: 1, Blocks: []int{1, 2}},
		collection.TestCollectionSpec{ReplicationLevel: 1, Blocks: []int{3, 4}},
		collection.TestCollectionSpec{ReplicationLevel: 2, Blocks: []int{5, 6}},
	})
	rc.Summarize(nil)
	cIndex := rc.CollectionIndicesForTesting()

	keepInfo := SpecifyReplication(map[int]int{1: 1, 2: 1, 3: 1, 5: 1, 6: 3, 7: 2})

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSetFromSlice([]int{4}),
		UnderReplicatedBlocks:      BlockSetFromSlice([]int{5}),
		OverReplicatedBlocks:       BlockSetFromSlice([]int{6}),
		CorrectlyReplicatedBlocks:  BlockSetFromSlice([]int{1, 2, 3}),
		KeepBlocksNotInCollections: BlockSetFromSlice([]int{7}),

		CollectionsNotFullyInKeep:      CollectionIndexSetFromSlice([]int{cIndex[1]}),
		UnderReplicatedCollections:     CollectionIndexSetFromSlice([]int{cIndex[2]}),
		OverReplicatedCollections:      CollectionIndexSetFromSlice([]int{cIndex[2]}),
		CorrectlyReplicatedCollections: CollectionIndexSetFromSlice([]int{cIndex[0]}),
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like: \n%+v but instead it is: \n%+v. Index to UUID is %v. BlockToCollectionIndices is %v.", expectedSummary, returnedSummary, rc.CollectionIndexToUuid, rc.BlockToCollectionIndices)
	}
}
