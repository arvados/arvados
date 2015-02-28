package collection

import (
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"reflect"
	"testing"
)

// This captures the result we expect from
// ReadCollections.Summarize().  Because CollectionUuidToIndex is
// indeterminate, we replace BlockToCollectionIndices with
// BlockToCollectionUuids.
type ExpectedSummary struct {
	OwnerToCollectionSize  map[string]int
	BlockToReplication     map[blockdigest.BlockDigest]int
	BlockToCollectionUuids map[blockdigest.BlockDigest][]string
}

func CompareSummarizedReadCollections(t *testing.T,
	summarized ReadCollections,
	expected ExpectedSummary) {

	if !reflect.DeepEqual(summarized.OwnerToCollectionSize,
		expected.OwnerToCollectionSize) {
		t.Fatalf("Expected summarized OwnerToCollectionSize to look like %+v but instead it is %+v",
			expected.OwnerToCollectionSize,
			summarized.OwnerToCollectionSize)
	}

	if !reflect.DeepEqual(summarized.BlockToReplication,
		expected.BlockToReplication) {
		t.Fatalf("Expected summarized BlockToReplication to look like %+v but instead it is %+v",
			expected.BlockToReplication,
			summarized.BlockToReplication)
	}

	summarizedBlockToCollectionUuids :=
		make(map[blockdigest.BlockDigest]map[string]struct{})
	for digest, indices := range summarized.BlockToCollectionIndices {
		uuidSet := make(map[string]struct{})
		summarizedBlockToCollectionUuids[digest] = uuidSet
		for _, index := range indices {
			uuidSet[summarized.CollectionIndexToUuid[index]] = struct{}{}
		}
	}

	expectedBlockToCollectionUuids :=
		make(map[blockdigest.BlockDigest]map[string]struct{})
	for digest, uuidSlice := range expected.BlockToCollectionUuids {
		uuidSet := make(map[string]struct{})
		expectedBlockToCollectionUuids[digest] = uuidSet
		for _, uuid := range uuidSlice {
			uuidSet[uuid] = struct{}{}
		}
	}

	if !reflect.DeepEqual(summarizedBlockToCollectionUuids,
		expectedBlockToCollectionUuids) {
		t.Fatalf("Expected summarized BlockToCollectionUuids to look like %+v but instead it is %+v", expectedBlockToCollectionUuids, summarizedBlockToCollectionUuids)
	}
}

func TestSummarizeSimple(t *testing.T) {
	rc := MakeTestReadCollections([]TestCollectionSpec{TestCollectionSpec{
		ReplicationLevel: 5,
		Blocks: []int{1, 2},
	}})

	rc.Summarize()

	c := rc.UuidToCollection["col0"]

	blockDigest1 := blockdigest.MakeTestBlockDigest(1)
	blockDigest2 := blockdigest.MakeTestBlockDigest(2)

	expected := ExpectedSummary{
		OwnerToCollectionSize:  map[string]int{c.OwnerUuid: c.TotalSize},
		BlockToReplication:     map[blockdigest.BlockDigest]int{blockDigest1: 5, blockDigest2: 5},
		BlockToCollectionUuids: map[blockdigest.BlockDigest][]string{blockDigest1: []string{c.Uuid}, blockDigest2: []string{c.Uuid}},
	}

	CompareSummarizedReadCollections(t, rc, expected)
}

func TestSummarizeOverlapping(t *testing.T) {
	rc := MakeTestReadCollections([]TestCollectionSpec{
		TestCollectionSpec{
			ReplicationLevel: 5,
			Blocks: []int{1, 2},
		},
		TestCollectionSpec{
			ReplicationLevel: 8,
			Blocks: []int{2, 3},
		},
	})

	rc.Summarize()

	c0 := rc.UuidToCollection["col0"]
	c1 := rc.UuidToCollection["col1"]

	blockDigest1 := blockdigest.MakeTestBlockDigest(1)
	blockDigest2 := blockdigest.MakeTestBlockDigest(2)
	blockDigest3 := blockdigest.MakeTestBlockDigest(3)

	expected := ExpectedSummary{
		OwnerToCollectionSize: map[string]int{
			c0.OwnerUuid: c0.TotalSize,
			c1.OwnerUuid: c1.TotalSize,
		},
		BlockToReplication: map[blockdigest.BlockDigest]int{
			blockDigest1: 5,
			blockDigest2: 8,
			blockDigest3: 8,
		},
		BlockToCollectionUuids: map[blockdigest.BlockDigest][]string{
			blockDigest1: []string{c0.Uuid},
			blockDigest2: []string{c0.Uuid, c1.Uuid},
			blockDigest3: []string{c1.Uuid},
		},
	}

	CompareSummarizedReadCollections(t, rc, expected)
}
