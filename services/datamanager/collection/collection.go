/* Deals with parsing Collection responses from API Server. */

package collection

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"log"
)

type Collection struct {
	BlockDigestToSize map[string]int
	ReplicationLevel int
	Uuid string
	OwnerUuid string
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

	var collections map[string]interface{}
	err := params.Client.List("collections", sdkParams, &collections)
	if err != nil {
		log.Fatalf("error querying collections: %v", err)
	}

	{
		var numReceived, numAvailable int
		results.ReadAllCollections, numReceived, numAvailable =
			util.SdkListResponseContainsAllAvailableItems(collections)

		if (!results.ReadAllCollections) {
			log.Printf("ERROR: Did not receive all collections.")
		}
		log.Printf("Received %d of %d available collections.",
			numReceived,
			numAvailable)
	}

	if collectionChannel, err := util.IterateSdkListItems(collections); err != nil {
		log.Fatalf("Error trying to iterate collections returned by SDK: %v", err)
	} else {
		index := 0
	 	results.UuidToCollection = make(map[string]Collection)
		for item_map := range collectionChannel {
			index += 1
			if m := params.LogEveryNthCollectionProcessed; m >0 && (index % m) == 0 {
				log.Printf("Processing collection #%d", index)
			}
			collection := Collection{Uuid: item_map["uuid"].(string),
				OwnerUuid: item_map["owner_uuid"].(string),
				BlockDigestToSize: make(map[string]int)}
			manifest := manifest.Manifest{item_map["manifest_text"].(string)}
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
			results.UuidToCollection[collection.Uuid] = collection
		}
	}
	return
}
