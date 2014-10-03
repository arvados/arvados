/* Keep Datamanager. Responsible for checking on and reporting on Keep Storage */

package main

import (
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/util"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"log"
)

// Helper type so we don't have to write out 'map[string]interface{}' every time.
type Dict map[string]interface{}

func main() {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %s", err.Error())
	}

	if is_admin, err := util.UserIsAdmin(arv); err != nil {
		log.Fatalf("Error querying current arvados user %s", err.Error())
	} else if !is_admin {
		log.Fatalf("Current user is not an admin. Datamanager can only be run by admins.")
	}

	readCollections := collection.GetCollections(
		collection.GetCollectionsParams{
			Client: arv, Limit: 50, LogEveryNthCollectionProcessed: 10})

	//log.Printf("Read Collections: %v", readCollections)

	// TODO(misha): Add a "readonly" flag. If we're in readonly mode,
	// lots of behaviors can become warnings (and obviously we can't
	// write anything).
	// if !readCollections.ReadAllCollections {
	// 	log.Fatalf("Did not read all collections")
	// }

	log.Printf("Read and processed %d collections",
		len(readCollections.UuidToCollection))

	readServers := keep.GetKeepServers(
		keep.GetKeepServersParams{Client: arv, Limit: 1000})

	log.Printf("Returned %d keep disks", len(readServers.AddressToContents))
}
