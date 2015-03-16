package summary

import (
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"reflect"
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
	return
}

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
	VerifyToCollectionIndexSet(t, []int{4}, map[int][]int{4: []int{1}}, []int{1})
	VerifyToCollectionIndexSet(t, []int{4}, map[int][]int{4: []int{1, 9}}, []int{1, 9})
	VerifyToCollectionIndexSet(t, []int{5, 6},
		map[int][]int{5: []int{2, 3}, 6: []int{3, 4}},
		[]int{2, 3, 4})
}

func TestSimpleSummary(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{
			ReplicationLevel: 1,
			Blocks:           []int{1, 2},
		},
	})

	rc.Summarize()

	// The internals aren't actually examined, so we can reuse the same one.
	dummyBlockServerInfo := keep.BlockServerInfo{}

	blockDigest1 := blockdigest.MakeTestBlockDigest(1)
	blockDigest2 := blockdigest.MakeTestBlockDigest(2)

	keepInfo := keep.ReadServers{
		BlockToServers: map[blockdigest.BlockDigest][]keep.BlockServerInfo{
			blockDigest1: []keep.BlockServerInfo{dummyBlockServerInfo},
			blockDigest2: []keep.BlockServerInfo{dummyBlockServerInfo},
		},
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	c := rc.UuidToCollection["col0"]

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSet{},
		UnderReplicatedBlocks:      BlockSet{},
		OverReplicatedBlocks:       BlockSet{},
		CorrectlyReplicatedBlocks:  BlockSetFromSlice([]int{1, 2}),
		KeepBlocksNotInCollections: BlockSet{},

		CollectionsNotFullyInKeep:  CollectionIndexSet{},
		UnderReplicatedCollections: CollectionIndexSet{},
		OverReplicatedCollections:  CollectionIndexSet{},
		CorrectlyReplicatedCollections: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c.Uuid]}),
	}

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like %+v but instead it is %+v", expectedSummary, returnedSummary)
	}
}

func TestMissingBlock(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{
			ReplicationLevel: 1,
			Blocks:           []int{1, 2},
		},
	})

	rc.Summarize()

	// The internals aren't actually examined, so we can reuse the same one.
	dummyBlockServerInfo := keep.BlockServerInfo{}

	blockDigest1 := blockdigest.MakeTestBlockDigest(1)

	keepInfo := keep.ReadServers{
		BlockToServers: map[blockdigest.BlockDigest][]keep.BlockServerInfo{
			blockDigest1: []keep.BlockServerInfo{dummyBlockServerInfo},
		},
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	c := rc.UuidToCollection["col0"]

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSetFromSlice([]int{2}),
		UnderReplicatedBlocks:      BlockSet{},
		OverReplicatedBlocks:       BlockSet{},
		CorrectlyReplicatedBlocks:  BlockSetFromSlice([]int{1}),
		KeepBlocksNotInCollections: BlockSet{},

		CollectionsNotFullyInKeep: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c.Uuid]}),
		UnderReplicatedCollections:     CollectionIndexSet{},
		OverReplicatedCollections:      CollectionIndexSet{},
		CorrectlyReplicatedCollections: CollectionIndexSet{},
	}

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like %+v but instead it is %+v", expectedSummary, returnedSummary)
	}
}

func TestUnderAndOverReplicatedBlocks(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{
			ReplicationLevel: 2,
			Blocks:           []int{1, 2},
		},
	})

	rc.Summarize()

	// The internals aren't actually examined, so we can reuse the same one.
	dummyBlockServerInfo := keep.BlockServerInfo{}

	blockDigest1 := blockdigest.MakeTestBlockDigest(1)
	blockDigest2 := blockdigest.MakeTestBlockDigest(2)

	keepInfo := keep.ReadServers{
		BlockToServers: map[blockdigest.BlockDigest][]keep.BlockServerInfo{
			blockDigest1: []keep.BlockServerInfo{dummyBlockServerInfo},
			blockDigest2: []keep.BlockServerInfo{dummyBlockServerInfo, dummyBlockServerInfo, dummyBlockServerInfo},
		},
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	c := rc.UuidToCollection["col0"]

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSet{},
		UnderReplicatedBlocks:      BlockSetFromSlice([]int{1}),
		OverReplicatedBlocks:       BlockSetFromSlice([]int{2}),
		CorrectlyReplicatedBlocks:  BlockSet{},
		KeepBlocksNotInCollections: BlockSet{},

		CollectionsNotFullyInKeep: CollectionIndexSet{},
		UnderReplicatedCollections: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c.Uuid]}),
		OverReplicatedCollections: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c.Uuid]}),
		CorrectlyReplicatedCollections: CollectionIndexSet{},
	}

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like %+v but instead it is %+v", expectedSummary, returnedSummary)
	}
}

func TestMixedReplication(t *testing.T) {
	rc := collection.MakeTestReadCollections([]collection.TestCollectionSpec{
		collection.TestCollectionSpec{
			ReplicationLevel: 1,
			Blocks:           []int{1, 2},
		},
		collection.TestCollectionSpec{
			ReplicationLevel: 1,
			Blocks:           []int{3, 4},
		},
		collection.TestCollectionSpec{
			ReplicationLevel: 2,
			Blocks:           []int{5, 6},
		},
	})

	rc.Summarize()

	// The internals aren't actually examined, so we can reuse the same one.
	dummyBlockServerInfo := keep.BlockServerInfo{}

	keepInfo := keep.ReadServers{
		BlockToServers: map[blockdigest.BlockDigest][]keep.BlockServerInfo{
			blockdigest.MakeTestBlockDigest(1): []keep.BlockServerInfo{dummyBlockServerInfo},
			blockdigest.MakeTestBlockDigest(2): []keep.BlockServerInfo{dummyBlockServerInfo},
			blockdigest.MakeTestBlockDigest(3): []keep.BlockServerInfo{dummyBlockServerInfo},
			blockdigest.MakeTestBlockDigest(5): []keep.BlockServerInfo{dummyBlockServerInfo},
			blockdigest.MakeTestBlockDigest(6): []keep.BlockServerInfo{dummyBlockServerInfo, dummyBlockServerInfo, dummyBlockServerInfo},
			blockdigest.MakeTestBlockDigest(7): []keep.BlockServerInfo{dummyBlockServerInfo, dummyBlockServerInfo},
		},
	}

	returnedSummary := SummarizeReplication(rc, keepInfo)

	c0 := rc.UuidToCollection["col0"]
	c1 := rc.UuidToCollection["col1"]
	c2 := rc.UuidToCollection["col2"]

	expectedSummary := ReplicationSummary{
		CollectionBlocksNotInKeep:  BlockSetFromSlice([]int{4}),
		UnderReplicatedBlocks:      BlockSetFromSlice([]int{5}),
		OverReplicatedBlocks:       BlockSetFromSlice([]int{6}),
		CorrectlyReplicatedBlocks:  BlockSetFromSlice([]int{1, 2, 3}),
		KeepBlocksNotInCollections: BlockSetFromSlice([]int{7}),

		CollectionsNotFullyInKeep: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c1.Uuid]}),
		UnderReplicatedCollections: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c2.Uuid]}),
		OverReplicatedCollections: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c2.Uuid]}),
		CorrectlyReplicatedCollections: CollectionIndexSetFromSlice(
			[]int{rc.CollectionUuidToIndex[c0.Uuid]}),
	}

	if !reflect.DeepEqual(returnedSummary, expectedSummary) {
		t.Fatalf("Expected returnedSummary to look like: \n%+v but instead it is: \n%+v. Index to UUID is %v. BlockToCollectionIndices is %v.", expectedSummary, returnedSummary, rc.CollectionIndexToUuid, rc.BlockToCollectionIndices)
	}
}
