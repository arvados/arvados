/* Deals with parsing Collection responses from API Server. */

package collection

import (
	"errors"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
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

// TODO(misha): Move this method somewhere more central
func SdkListResponseContainsAllAvailableItems(response map[string]interface{}) (containsAll bool, numContained int, numAvailable int) {
	if value, ok := response["items"]; ok {
		items := value.([]interface{})
		{
			var itemsAvailable interface{}
			if itemsAvailable, ok = response["items_available"]; !ok {
				// TODO(misha): Consider returning an error here (and above if
				// we can't find items) so that callers can recover.
				log.Fatalf("API server did not return the number of items available")
			}
			numContained = len(items)
			numAvailable = int(itemsAvailable.(float64))
			// If we never entered this block, allAvailable would be false by
			// default, which is what we want
			containsAll = numContained == numAvailable
		}
	}
	return
}

func IterateSdkListItems(response map[string]interface{}) (c <-chan map[string]interface{}, err error) {
	if value, ok := response["items"]; ok {
		ch := make(chan map[string]interface{})
		c = ch
		items := value.([]interface{})
		go func() {
			for _, item := range items {
				ch <- item.(map[string]interface{})
			}
			close(ch)
		}()
	} else {
		err = errors.New("Could not find \"items\" field in response " +
			"passed to IterateSdkListItems()")
	}
	return
}



func GetCollections(params GetCollectionsParams) (results ReadCollections) {
	if &params.Client == nil {
		log.Fatalf("Received params.Client passed to GetCollections() should " +
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
			SdkListResponseContainsAllAvailableItems(collections)

		if (!results.ReadAllCollections) {
			log.Printf("ERROR: Did not receive all collections.")
		}
		log.Printf("Received %d of %d available collections.",
			numReceived,
			numAvailable)
	}

	if collectionChannel, err := IterateSdkListItems(collections); err != nil {
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
