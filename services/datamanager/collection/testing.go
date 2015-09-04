// Code used for testing only.

package collection

import (
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
)

type TestCollectionSpec struct {
	// The desired replication level
	ReplicationLevel int
	// Blocks this contains, represented by ints. Ints repeated will
	// still only represent one block
	Blocks []int
}

// Creates a ReadCollections object for testing based on the give
// specs.  Only the ReadAllCollections and UuidToCollection fields are
// populated.  To populate other fields call rc.Summarize().
func MakeTestReadCollections(specs []TestCollectionSpec) (rc ReadCollections) {
	rc = ReadCollections{
		ReadAllCollections: true,
		UuidToCollection:   map[string]Collection{},
	}

	for i, spec := range specs {
		c := Collection{
			Uuid:              fmt.Sprintf("col%d", i),
			OwnerUuid:         fmt.Sprintf("owner%d", i),
			ReplicationLevel:  spec.ReplicationLevel,
			BlockDigestToSize: map[blockdigest.BlockDigest]int{},
		}
		rc.UuidToCollection[c.Uuid] = c
		for _, j := range spec.Blocks {
			c.BlockDigestToSize[blockdigest.MakeTestBlockDigest(j)] = j
		}
		// We compute the size in a separate loop because the value
		// computed in the above loop would be invalid if c.Blocks
		// contained duplicates.
		for _, size := range c.BlockDigestToSize {
			c.TotalSize += size
		}
	}
	return
}

// Returns a slice giving the collection index of each collection that
// was passed in to MakeTestReadCollections. rc.Summarize() must be
// called before this method, since Summarize() assigns an index to
// each collection.
func (rc ReadCollections) CollectionIndicesForTesting() (indices []int) {
	// TODO(misha): Assert that rc.Summarize() has been called.
	numCollections := len(rc.CollectionIndexToUuid)
	indices = make([]int, numCollections)
	for i := 0; i < numCollections; i++ {
		indices[i] = rc.CollectionUuidToIndex[fmt.Sprintf("col%d", i)]
	}
	return
}
