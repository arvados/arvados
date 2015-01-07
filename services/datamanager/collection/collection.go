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
	"runtime"
	"runtime/pprof"
	"time"
)

var (
	heap_profile_filename string
	// globals for debugging
	totalManifestSize uint64
	maxManifestSize uint64
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
	Uuid           string     `json:"uuid"`
	OwnerUuid      string     `json:"owner_uuid"`
	Redundancy     int        `json:"redundancy"`
	ModifiedAt     time.Time  `json:"modified_at"`
	ManifestText   string     `json:"manifest_text"`
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

// Write the heap profile to a file for later review.
// Since a file is expected to only contain a single heap profile this
// function overwrites the previously written profile, so it is safe
// to call multiple times in a single run.
// Otherwise we would see cumulative numbers as explained here:
// https://groups.google.com/d/msg/golang-nuts/ZyHciRglQYc/2nh4Ndu2fZcJ
func WriteHeapProfile() {
	if heap_profile_filename != "" {

		heap_profile, err := os.Create(heap_profile_filename)
		if err != nil {
			log.Fatal(err)
		}

		defer heap_profile.Close()

		err = pprof.WriteHeapProfile(heap_profile)
		if err != nil {
			log.Fatal(err)
		}
	}
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
		"redundancy",
		"modified_at"}

	sdkParams := arvadosclient.Dict{
		"select": fieldsWanted,
		"order": []string{"modified_at ASC"},
		"filters": [][]string{[]string{"modified_at", ">=", "1900-01-01T00:00:00Z"}}}

	if params.BatchSize > 0 {
		sdkParams["limit"] = params.BatchSize
	}

	// MISHA UNDO THIS TEMPORARY HACK TO FIND BUG!
	sdkParams["limit"] = 50

	initialNumberOfCollectionsAvailable := NumberCollectionsAvailable(params.Client)
	// Include a 1% margin for collections added while we're reading so
	// that we don't have to grow the map in most cases.
	maxExpectedCollections := int(
		float64(initialNumberOfCollectionsAvailable) * 1.01)
	results.UuidToCollection = make(map[string]Collection, maxExpectedCollections)

	// These values are just for getting the loop to run the first time,
	// afterwards they'll be set to real values.
	previousTotalCollections := -1
	totalCollections := 0
	for totalCollections > previousTotalCollections {
		// We're still finding new collections

		// Write the heap profile for examining memory usage
		WriteHeapProfile()

		// Get next batch of collections.
		var collections SdkCollectionList
		err := params.Client.List("collections", sdkParams, &collections)
		if err != nil {
			log.Fatalf("error querying collections: %+v", err)
		}

		// Process collection and update our date filter.
		sdkParams["filters"].([][]string)[0][2] =
			ProcessCollections(collections.Items, results.UuidToCollection).Format(time.RFC3339)

		// update counts
		previousTotalCollections = totalCollections
		totalCollections = len(results.UuidToCollection)

		log.Printf("%d collections read, %d new in last batch, " +
			"%s latest modified date, %.0f %d %d avg,max,total manifest size",
			totalCollections,
			totalCollections - previousTotalCollections,
			sdkParams["filters"].([][]string)[0][2],
			float32(totalManifestSize)/float32(totalCollections),
			maxManifestSize, totalManifestSize)
	}

	// Just in case this lowers the numbers reported in the heap profile.
	runtime.GC()

	// Write the heap profile for examining memory usage
	WriteHeapProfile()

	return
}


// StrCopy returns a newly allocated string.
// It is useful to copy slices so that the garbage collector can reuse
// the memory of the longer strings they came from.
func StrCopy(s string) string {
	return string([]byte(s))
}


func ProcessCollections(receivedCollections []SdkCollectionInfo,
	uuidToCollection map[string]Collection) (latestModificationDate time.Time) {
	for _, sdkCollection := range receivedCollections {
		collection := Collection{Uuid: StrCopy(sdkCollection.Uuid),
			OwnerUuid: StrCopy(sdkCollection.OwnerUuid),
			ReplicationLevel: sdkCollection.Redundancy,
			BlockDigestToSize: make(map[blockdigest.BlockDigest]int)}

		if sdkCollection.ModifiedAt.IsZero() {
			log.Fatalf(
				"Arvados SDK collection returned with unexpected zero modifcation " +
					"date. This probably means that either we failed to parse the " +
					"modification date or the API server has changed how it returns " +
					"modification dates: %v",
				collection)
		}
		if sdkCollection.ModifiedAt.After(latestModificationDate) {
			latestModificationDate = sdkCollection.ModifiedAt
		}
		manifest := manifest.Manifest{sdkCollection.ManifestText}
		manifestSize := uint64(len(sdkCollection.ManifestText))

		totalManifestSize += manifestSize
		if manifestSize > maxManifestSize {
			maxManifestSize = manifestSize
		}
		
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
