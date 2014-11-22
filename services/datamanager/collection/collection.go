/* Deals with parsing Collection responses from API Server. */

package collection

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"log"
)

type Collection struct {
	Uuid string
	OwnerUuid string
	ReplicationLevel int
	BlockDigestToSize map[string]int
	TotalSize int
}

type ReadCollections struct {
	ReadAllCollections bool
	UuidToCollection map[string]Collection
}

type GetCollectionsParams struct {
	Client arvadosclient.ArvadosClient
	Limit int
	LogEveryNthCollectionProcessed int  // 0 means don't report any
}

type SdkCollectionInfo struct {
	Uuid           string   `json:"uuid"`
	OwnerUuid      string   `json:"owner_uuid"`
	Redundancy     int      `json:"redundancy"`
	ManifestText   string   `json:"manifest_text"`
}

type SdkCollectionList struct {
	ItemsAvailable   int                   `json:"items_available"`
	Items            []SdkCollectionInfo   `json:"items"`
}

// Methods to implement util.SdkListResponse Interface
func (s SdkCollectionList) NumItemsAvailable() (numAvailable int, err error) {
	return s.ItemsAvailable, nil
}

func (s SdkCollectionList) NumItemsContained() (numContained int, err error) {
	return len(s.Items), nil
}

func GetCollections(params GetCollectionsParams) (results ReadCollections) {
	if &params.Client == nil {
		log.Fatalf("params.Client passed to GetCollections() should " +
			"contain a valid ArvadosClient, but instead it is nil.")
	}

	fieldsWanted := []string{"manifest_text",
		"owner_uuid",
		"uuid",
		// TODO(misha): Start using the redundancy field.
		"redundancy"}

	sdkParams := arvadosclient.Dict{"select": fieldsWanted}
	if params.Limit > 0 {
		sdkParams["limit"] = params.Limit
	}

	var collections SdkCollectionList
	err := params.Client.List("collections", sdkParams, &collections)
	if err != nil {
		log.Fatalf("error querying collections: %v", err)
	}

	{
		var numReceived, numAvailable int
		results.ReadAllCollections, numReceived, numAvailable =
			util.ContainsAllAvailableItems(collections)

		if (!results.ReadAllCollections) {
			log.Printf("ERROR: Did not receive all collections.")
		}
		log.Printf("Received %d of %d available collections.",
			numReceived,
			numAvailable)
	}

	results.UuidToCollection = make(map[string]Collection)
	for i, sdkCollection := range collections.Items {
		count := i + 1
		if m := params.LogEveryNthCollectionProcessed; m >0 && (count % m) == 0 {
			log.Printf("Processing collection #%d", count)
		}
		collection := Collection{Uuid: sdkCollection.Uuid,
			OwnerUuid: sdkCollection.OwnerUuid,
			ReplicationLevel: sdkCollection.Redundancy,
			BlockDigestToSize: make(map[string]int)}
		manifest := manifest.Manifest{sdkCollection.ManifestText}
		blockChannel := manifest.BlockIterWithDuplicates()
		for block := range blockChannel {
			if stored_size, stored := collection.BlockDigestToSize[block.Digest];
			stored && stored_size != block.Size {
				log.Fatalf(
					"Collection %s contains multiple sizes (%d and %d) for block %s",
					collection.Uuid,
					stored_size,
					block.Size,
					block.Digest)
			}
			collection.BlockDigestToSize[block.Digest] = block.Size
		}
		collection.TotalSize = 0
		for _, size := range collection.BlockDigestToSize {
			collection.TotalSize += size
		}
		results.UuidToCollection[collection.Uuid] = collection
	}

	return
}
