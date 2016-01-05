// Deals with parsing Collection responses from API Server.

package collection

import (
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

var (
	HeapProfileFilename string
)

// Collection representation
type Collection struct {
	UUID              string
	OwnerUUID         string
	ReplicationLevel  int
	BlockDigestToSize map[blockdigest.BlockDigest]int
	TotalSize         int
}

// ReadCollections holds information about collections from API server
type ReadCollections struct {
	ReadAllCollections        bool
	UUIDToCollection          map[string]Collection
	OwnerToCollectionSize     map[string]int
	BlockToDesiredReplication map[blockdigest.DigestWithSize]int
	CollectionUUIDToIndex     map[string]int
	CollectionIndexToUUID     []string
	BlockToCollectionIndices  map[blockdigest.DigestWithSize][]int
}

// GetCollectionsParams params
type GetCollectionsParams struct {
	Client    arvadosclient.ArvadosClient
	Logger    *logger.Logger
	BatchSize int
}

// SdkCollectionInfo holds collection info from api
type SdkCollectionInfo struct {
	UUID         string    `json:"uuid"`
	OwnerUUID    string    `json:"owner_uuid"`
	Redundancy   int       `json:"redundancy"`
	ModifiedAt   time.Time `json:"modified_at"`
	ManifestText string    `json:"manifest_text"`
}

// SdkCollectionList lists collections from api
type SdkCollectionList struct {
	ItemsAvailable int                 `json:"items_available"`
	Items          []SdkCollectionInfo `json:"items"`
}

func init() {
	flag.StringVar(&HeapProfileFilename,
		"heap-profile",
		"",
		"File to write the heap profiles to. Leave blank to skip profiling.")
}

// WriteHeapProfile writes the heap profile to a file for later review.
// Since a file is expected to only contain a single heap profile this
// function overwrites the previously written profile, so it is safe
// to call multiple times in a single run.
// Otherwise we would see cumulative numbers as explained here:
// https://groups.google.com/d/msg/golang-nuts/ZyHciRglQYc/2nh4Ndu2fZcJ
func WriteHeapProfile() error {
	if HeapProfileFilename != "" {
		heapProfile, err := os.Create(HeapProfileFilename)
		if err != nil {
			return err
		}

		defer heapProfile.Close()

		err = pprof.WriteHeapProfile(heapProfile)
		return err
	}

	return nil
}

// GetCollectionsAndSummarize gets collections from api and summarizes
func GetCollectionsAndSummarize(params GetCollectionsParams) (results ReadCollections, err error) {
	results, err = GetCollections(params)
	if err != nil {
		return
	}

	results.Summarize(params.Logger)

	log.Printf("Uuid to Size used: %v", results.OwnerToCollectionSize)
	log.Printf("Read and processed %d collections",
		len(results.UUIDToCollection))

	// TODO(misha): Add a "readonly" flag. If we're in readonly mode,
	// lots of behaviors can become warnings (and obviously we can't
	// write anything).
	// if !readCollections.ReadAllCollections {
	// 	log.Fatalf("Did not read all collections")
	// }

	return
}

// GetCollections gets collections from api
func GetCollections(params GetCollectionsParams) (results ReadCollections, err error) {
	if &params.Client == nil {
		err = fmt.Errorf("params.Client passed to GetCollections() should " +
			"contain a valid ArvadosClient, but instead it is nil.")
		return
	}

	fieldsWanted := []string{"manifest_text",
		"owner_uuid",
		"uuid",
		"redundancy",
		"modified_at"}

	sdkParams := arvadosclient.Dict{
		"select":  fieldsWanted,
		"order":   []string{"modified_at ASC"},
		"filters": [][]string{[]string{"modified_at", ">=", "1900-01-01T00:00:00Z"}}}

	if params.BatchSize > 0 {
		sdkParams["limit"] = params.BatchSize
	}

	var defaultReplicationLevel int
	{
		var value interface{}
		value, err = params.Client.Discovery("defaultCollectionReplication")
		if err != nil {
			return
		}

		defaultReplicationLevel = int(value.(float64))
		if defaultReplicationLevel <= 0 {
			err = fmt.Errorf("Default collection replication returned by arvados SDK "+
				"should be a positive integer but instead it was %d.",
				defaultReplicationLevel)
			return
		}
	}

	initialNumberOfCollectionsAvailable, err :=
		util.NumberItemsAvailable(params.Client, "collections")
	if err != nil {
		return
	}
	// Include a 1% margin for collections added while we're reading so
	// that we don't have to grow the map in most cases.
	maxExpectedCollections := int(
		float64(initialNumberOfCollectionsAvailable) * 1.01)
	results.UUIDToCollection = make(map[string]Collection, maxExpectedCollections)

	if params.Logger != nil {
		params.Logger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			collectionInfo := logger.GetOrCreateMap(p, "collection_info")
			collectionInfo["num_collections_at_start"] = initialNumberOfCollectionsAvailable
			collectionInfo["batch_size"] = params.BatchSize
			collectionInfo["default_replication_level"] = defaultReplicationLevel
		})
	}

	// These values are just for getting the loop to run the first time,
	// afterwards they'll be set to real values.
	previousTotalCollections := -1
	totalCollections := 0
	for totalCollections > previousTotalCollections {
		// We're still finding new collections

		// Write the heap profile for examining memory usage
		err = WriteHeapProfile()
		if err != nil {
			return
		}

		// Get next batch of collections.
		var collections SdkCollectionList
		err = params.Client.List("collections", sdkParams, &collections)
		if err != nil {
			return
		}

		// Process collection and update our date filter.
		latestModificationDate, maxManifestSize, totalManifestSize, err := ProcessCollections(params.Logger,
			collections.Items,
			defaultReplicationLevel,
			results.UUIDToCollection)
		if err != nil {
			return results, err
		}
		sdkParams["filters"].([][]string)[0][2] = latestModificationDate.Format(time.RFC3339)

		// update counts
		previousTotalCollections = totalCollections
		totalCollections = len(results.UUIDToCollection)

		log.Printf("%d collections read, %d new in last batch, "+
			"%s latest modified date, %.0f %d %d avg,max,total manifest size",
			totalCollections,
			totalCollections-previousTotalCollections,
			sdkParams["filters"].([][]string)[0][2],
			float32(totalManifestSize)/float32(totalCollections),
			maxManifestSize, totalManifestSize)

		if params.Logger != nil {
			params.Logger.Update(func(p map[string]interface{}, e map[string]interface{}) {
				collectionInfo := logger.GetOrCreateMap(p, "collection_info")
				collectionInfo["collections_read"] = totalCollections
				collectionInfo["latest_modified_date_seen"] = sdkParams["filters"].([][]string)[0][2]
				collectionInfo["total_manifest_size"] = totalManifestSize
				collectionInfo["max_manifest_size"] = maxManifestSize
			})
		}
	}

	// Write the heap profile for examining memory usage
	err = WriteHeapProfile()

	return
}

// StrCopy returns a newly allocated string.
// It is useful to copy slices so that the garbage collector can reuse
// the memory of the longer strings they came from.
func StrCopy(s string) string {
	return string([]byte(s))
}

// ProcessCollections read from api server
func ProcessCollections(arvLogger *logger.Logger,
	receivedCollections []SdkCollectionInfo,
	defaultReplicationLevel int,
	UUIDToCollection map[string]Collection,
) (
	latestModificationDate time.Time,
	maxManifestSize, totalManifestSize uint64,
	err error,
) {
	for _, sdkCollection := range receivedCollections {
		collection := Collection{UUID: StrCopy(sdkCollection.UUID),
			OwnerUUID:         StrCopy(sdkCollection.OwnerUUID),
			ReplicationLevel:  sdkCollection.Redundancy,
			BlockDigestToSize: make(map[blockdigest.BlockDigest]int)}

		if sdkCollection.ModifiedAt.IsZero() {
			err = fmt.Errorf(
				"Arvados SDK collection returned with unexpected zero "+
					"modification date. This probably means that either we failed to "+
					"parse the modification date or the API server has changed how "+
					"it returns modification dates: %+v",
				collection)
			return
		}

		if sdkCollection.ModifiedAt.After(latestModificationDate) {
			latestModificationDate = sdkCollection.ModifiedAt
		}

		if collection.ReplicationLevel == 0 {
			collection.ReplicationLevel = defaultReplicationLevel
		}

		manifest := manifest.Manifest{Text: sdkCollection.ManifestText}
		manifestSize := uint64(len(sdkCollection.ManifestText))

		if _, alreadySeen := UUIDToCollection[collection.UUID]; !alreadySeen {
			totalManifestSize += manifestSize
		}
		if manifestSize > maxManifestSize {
			maxManifestSize = manifestSize
		}

		blockChannel := manifest.BlockIterWithDuplicates()
		for block := range blockChannel {
			if storedSize, stored := collection.BlockDigestToSize[block.Digest]; stored && storedSize != block.Size {
				log.Printf(
					"Collection %s contains multiple sizes (%d and %d) for block %s",
					collection.UUID,
					storedSize,
					block.Size,
					block.Digest)
			}
			collection.BlockDigestToSize[block.Digest] = block.Size
		}
		if manifest.Err != nil {
			err = manifest.Err
			return
		}

		collection.TotalSize = 0
		for _, size := range collection.BlockDigestToSize {
			collection.TotalSize += size
		}
		UUIDToCollection[collection.UUID] = collection

		// Clear out all the manifest strings that we don't need anymore.
		// These hopefully form the bulk of our memory usage.
		manifest.Text = ""
		sdkCollection.ManifestText = ""
	}

	return
}

// Summarize the collections read
func (readCollections *ReadCollections) Summarize(arvLogger *logger.Logger) {
	readCollections.OwnerToCollectionSize = make(map[string]int)
	readCollections.BlockToDesiredReplication = make(map[blockdigest.DigestWithSize]int)
	numCollections := len(readCollections.UUIDToCollection)
	readCollections.CollectionUUIDToIndex = make(map[string]int, numCollections)
	readCollections.CollectionIndexToUUID = make([]string, 0, numCollections)
	readCollections.BlockToCollectionIndices = make(map[blockdigest.DigestWithSize][]int)

	for _, coll := range readCollections.UUIDToCollection {
		collectionIndex := len(readCollections.CollectionIndexToUUID)
		readCollections.CollectionIndexToUUID =
			append(readCollections.CollectionIndexToUUID, coll.UUID)
		readCollections.CollectionUUIDToIndex[coll.UUID] = collectionIndex

		readCollections.OwnerToCollectionSize[coll.OwnerUUID] =
			readCollections.OwnerToCollectionSize[coll.OwnerUUID] + coll.TotalSize

		for block, size := range coll.BlockDigestToSize {
			locator := blockdigest.DigestWithSize{Digest: block, Size: uint32(size)}
			readCollections.BlockToCollectionIndices[locator] =
				append(readCollections.BlockToCollectionIndices[locator],
					collectionIndex)
			storedReplication := readCollections.BlockToDesiredReplication[locator]
			if coll.ReplicationLevel > storedReplication {
				readCollections.BlockToDesiredReplication[locator] =
					coll.ReplicationLevel
			}
		}
	}

	if arvLogger != nil {
		arvLogger.Update(func(p map[string]interface{}, e map[string]interface{}) {
			collectionInfo := logger.GetOrCreateMap(p, "collection_info")
			// Since maps are shallow copied, we run a risk of concurrent
			// updates here. By copying results.OwnerToCollectionSize into
			// the log, we're assuming that it won't be updated.
			collectionInfo["owner_to_collection_size"] =
				readCollections.OwnerToCollectionSize
			collectionInfo["distinct_blocks_named"] =
				len(readCollections.BlockToDesiredReplication)
		})
	}

	return
}
