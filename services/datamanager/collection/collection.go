/* Deals with parsing Collection responses from API Server. */

package collection

import (
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

type readCollections struct {
	ReadAllCollections bool
	UuidToCollection map[string]Collection
}

func GetCollections(arv arvadosclient.ArvadosClient) (results readCollections) {
	fieldsWanted := []string{"manifest_text",
		"owner_uuid",
		"uuid",
		// TODO(misha): Start using the redundancy field.
		"redundancy"}

	// TODO(misha): Set the limit param with a flag.
	params := arvadosclient.Dict{"limit": 1, "select": fieldsWanted}

	var collections map[string]interface{}
	err := arv.List("collections", params, &collections)
	if err != nil {
		log.Fatalf("error querying collections: %v", err)
	}

	results.ReadAllCollections = false

	if value, ok := collections["items"]; ok {
		items := value.([]interface{})

		{
			var itemsAvailable interface{}
			if itemsAvailable, ok = collections["items_available"]; !ok {
				log.Fatalf("API server did not return the number of items available")
			}
			numReceived := len(items)
			numAvailable := int(itemsAvailable.(float64))
			results.ReadAllCollections = numReceived == numAvailable

			if (!results.ReadAllCollections) {
				log.Printf(
					"ERROR: Did not receive all collections. " +
						"Received %d of %d available collections.",
					numReceived,
					numAvailable)
			}
		}

		results.UuidToCollection = make(map[string]Collection)
		for _, item := range items {
			item_map := item.(map[string]interface{})
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
