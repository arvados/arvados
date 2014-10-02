/* Keep Datamanager. Responsible for checking on and reporting on Keep Storage */

package main

import (
	//"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/services/datamanager/collection"
	"log"
)

// Helper type so we don't have to write out 'map[string]interface{}' every time.
type Dict map[string]interface{}

func UserIsAdmin(arv arvadosclient.ArvadosClient) (is_admin bool, err error) {
	type user struct {
		IsAdmin bool `json:"is_admin"`
	}
	var u user
	err = arv.Call("GET", "users", "", "current", nil, &u)
	return u.IsAdmin, err
}

func main() {
	fmt.Println("Hello, world\n")

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %s", err.Error())
	}

	if is_admin, err := UserIsAdmin(arv); err != nil {
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

	// TODO(misha): Send SDK and Keep requests in parallel
}
