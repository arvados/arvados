/* Deals with parsing Collection responses from API Server. */

package collection

import (
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	//"git.curoverse.com/arvados.git/sdk/go/util"
	"log"
	"os"
	"runtime/pprof"
)

var (
	heap_profile_filename string
	heap_profile *os.File
)

type Collection struct {
	Uuid string
	OwnerUuid string
	ReplicationLevel int
	BlockDigestToSize map[blockdigest.BlockDigest]int
	TotalSize int
}

type ReadCollections struct {
	ReadAllCollections bool
	UuidToCollection map[string]Collection
}

type GetCollectionsParams struct {
	Client arvadosclient.ArvadosClient
	BatchSize int
}

type SdkCollectionInfo struct {
	Uuid           string   `json:"uuid"`
	OwnerUuid      string   `json:"owner_uuid"`
	Redundancy     int      `json:"redundancy"`
	ModifiedAt     string   `json:"modified_at"`
	ManifestText   string   `json:"manifest_text"`
}

type SdkCollectionList struct {
	ItemsAvailable   int                   `json:"items_available"`
	Items            []SdkCollectionInfo   `json:"items"`
}

func init() {
	flag.StringVar(&heap_profile_filename, 
		"heap-profile",
		"",
		"File to write the heap profiles to.")
}

// // Methods to implement util.SdkListResponse Interface
// func (s SdkCollectionList) NumItemsAvailable() (numAvailable int, err error) {
// 	return s.ItemsAvailable, nil
// }

// func (s SdkCollectionList) NumItemsContained() (numContained int, err error) {
// 	return len(s.Items), nil
// }

func GetCollections(params GetCollectionsParams) (results ReadCollections) {
	if &params.Client == nil {
		log.Fatalf("params.Client passed to GetCollections() should " +
			"contain a valid ArvadosClient, but instead it is nil.")
	}

	// TODO(misha): move this code somewhere better and make sure it's
	// only run once
	if heap_profile_filename != "" {
		var err error
		heap_profile, err = os.Create(heap_profile_filename)
		if err != nil {
			log.Fatal(err)
		}
	}

	fieldsWanted := []string{"manifest_text",
		"owner_uuid",
		"uuid",
		// TODO(misha): Start using the redundancy field.
		"redundancy",
		"modified_at"}

	sdkParams := arvadosclient.Dict{
		"select": fieldsWanted,
		"order": []string{"modified_at ASC"},
		"filters": [][]string{[]string{"modified_at", ">=", "1900-01-01T00:00:00Z"}}}
		// MISHA UNDO THIS TEMPORARY HACK TO FIND BUG!
		//"filters": [][]string{[]string{"modified_at", ">=", "2014-11-05T20:44:50Z"}}}

	if params.BatchSize > 0 {
		sdkParams["limit"] = params.BatchSize
	}

	// MISHA UNDO THIS TEMPORARY HACK TO FIND BUG!
	sdkParams["limit"] = 50

	// {
	// 	var numReceived, numAvailable int
	// 	results.ReadAllCollections, numReceived, numAvailable =
	// 		util.ContainsAllAvailableItems(collections)

	// 	if (!results.ReadAllCollections) {
	// 		log.Printf("ERROR: Did not receive all collections.")
	// 	}
	// 	log.Printf("Received %d of %d available collections.",
	// 		numReceived,
	// 		numAvailable)
	// }

	initialNumberOfCollectionsAvailable := NumberCollectionsAvailable(params.Client)
	// Include a 1% margin for collections added while we're reading so
	// that we don't have to grow the map in most cases.
	maxExpectedCollections := int(
		float64(initialNumberOfCollectionsAvailable) * 1.01)
	results.UuidToCollection = make(map[string]Collection, maxExpectedCollections)

	previousTotalCollections := -1
	for len(results.UuidToCollection) > previousTotalCollections {
		// We're still finding new collections
		log.Printf("previous, current: %d %d", previousTotalCollections, len(results.UuidToCollection))

		// update count
		previousTotalCollections = len(results.UuidToCollection)

		// Write the heap profile for examining memory usage
		if heap_profile != nil {
			err := pprof.WriteHeapProfile(heap_profile)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Get next batch of collections.
		var collections SdkCollectionList
		log.Printf("Running with SDK Params: %v", sdkParams)
		err := params.Client.List("collections", sdkParams, &collections)
		if err != nil {
			log.Fatalf("error querying collections: %+v", err)
		}

		// Process collection and update our date filter.
		sdkParams["filters"].([][]string)[0][2] = ProcessCollections(collections.Items, results.UuidToCollection)
		log.Printf("Latest date seen %s", sdkParams["filters"].([][]string)[0][2])
	}
	log.Printf("previous, current: %d %d", previousTotalCollections, len(results.UuidToCollection))

	// Write the heap profile for examining memory usage
	if heap_profile != nil {
		err := pprof.WriteHeapProfile(heap_profile)
		if err != nil {
			log.Fatal(err)
		}
	}

	return
}


func ProcessCollections(receivedCollections []SdkCollectionInfo,
	uuidToCollection map[string]Collection) (latestModificationDate string) {
	for _, sdkCollection := range receivedCollections {
		collection := Collection{Uuid: sdkCollection.Uuid,
			OwnerUuid: sdkCollection.OwnerUuid,
			ReplicationLevel: sdkCollection.Redundancy,
			BlockDigestToSize: make(map[blockdigest.BlockDigest]int)}
		// log.Printf("Seeing modification date, owner_uuid: %s %s",
		// 	sdkCollection.ModifiedAt,
		// 	sdkCollection.OwnerUuid)
		if sdkCollection.ModifiedAt > latestModificationDate {
			latestModificationDate = sdkCollection.ModifiedAt
		}
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
		uuidToCollection[collection.Uuid] = collection

		// Clear out all the manifest strings that we don't need anymore.
		// These hopefully form the bulk of our memory usage.
		manifest.Text = ""
		sdkCollection.ManifestText = ""
	}

	return
}


func NumberCollectionsAvailable(client arvadosclient.ArvadosClient) (int) {
	var collections SdkCollectionList
	sdkParams := arvadosclient.Dict{"limit": 0}
	err := client.List("collections", sdkParams, &collections)
	if err != nil {
		log.Fatalf("error querying collections for items available: %v", err)
	}

	return collections.ItemsAvailable
}
