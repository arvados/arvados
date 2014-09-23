/* Deals with parsing Collection responses from API Server. */

package collection

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/services/datamanager/manifest"
	"log"
)

type Collection struct {
	// TODO(misha): Consider whether we need BlockLocator.hints, and if
	// not, perhaps we should use a custom struct here.
	Blocks []manifest.BlockLocator
	ReplicationLevel int
	Uuid string
	ownerUuid string
}

type readCollections struct {
	ReadAllCollections bool
	UuidToCollection map[string]Collection
}

func GetCollections(arv arvadosclient.ArvadosClient) (results readCollections) {
	fieldsWanted := []string{"manifest_text",
		"owner_uuid",
		"uuid",
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
			itemsAvailable, ok := collections["items_available"]
			if !ok {
				log.Fatalf("API server did not return the number of items available")
			}
			numReceived := len(items)
			numAvailable := int(itemsAvailable.(float64))
			results.ReadAllCollections = numReceived == numAvailable

			if (!results.ReadAllCollections) {
				log.Printf("ERROR: Did not receive all collections. Received %d of %d available collections.",
					numReceived, numAvailable)
			}
		}

		results.UuidToCollection = make(map[string]Collection)
		for _, item := range items {
			item_map := item.(map[string]interface{})
			collection := Collection{Uuid: item_map["uuid"].(string),
				ownerUuid: item_map["owner_uuid"].(string)}
			manifest := manifest.Manifest{item_map["manifest_text"].(string)}
			blockChannel := manifest.BlockIter()
			for block := range blockChannel {
				collection.Blocks = append(collection.Blocks, block)
			}
			results.UuidToCollection[collection.Uuid] = collection
		}
	}
	return
}
