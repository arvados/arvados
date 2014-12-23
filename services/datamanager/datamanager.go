/* Keep Datamanager. Responsible for checking on and reporting on Keep Storage */

package main

import (
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
//	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"log"
)

func main() {
	flag.Parse()

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %s", err.Error())
	}

	if is_admin, err := util.UserIsAdmin(arv); err != nil {
		log.Fatalf("Error querying current arvados user %s", err.Error())
	} else if !is_admin {
		log.Fatalf("Current user is not an admin. Datamanager can only be run by admins.")
	}

	// TODO(misha): Read Collections and Keep Contents concurrently as goroutines.

	readCollections := collection.GetCollections(
		collection.GetCollectionsParams{
			Client: arv, BatchSize: 500})

	//log.Printf("Read Collections: %v", readCollections)

	UserUsage := ComputeSizeOfOwnedCollections(readCollections)
	log.Printf("Uuid to Size used: %v", UserUsage)

	// TODO(misha): Add a "readonly" flag. If we're in readonly mode,
	// lots of behaviors can become warnings (and obviously we can't
	// write anything).
	// if !readCollections.ReadAllCollections {
	// 	log.Fatalf("Did not read all collections")
	// }

	log.Printf("Read and processed %d collections",
		len(readCollections.UuidToCollection))

	// readServers := keep.GetKeepServers(
	// 	keep.GetKeepServersParams{Client: arv, Limit: 1000})

	// log.Printf("Returned %d keep disks", len(readServers.AddressToContents))
}

func ComputeSizeOfOwnedCollections(readCollections collection.ReadCollections) (
	results map[string]int) {
	results = make(map[string]int)
	for _, coll := range readCollections.UuidToCollection {
		results[coll.OwnerUuid] = results[coll.OwnerUuid] + coll.TotalSize
	}
	return
}
