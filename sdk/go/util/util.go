/* Helper methods for dealing with responses from API Server. */

package util

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
)

func UserIsAdmin(arv arvadosclient.ArvadosClient) (is_admin bool, err error) {
	type user struct {
		IsAdmin bool `json:"is_admin"`
	}
	var u user
	err = arv.Call("GET", "users", "", "current", nil, &u)
	return u.IsAdmin, err
}

// Returns the total count of a particular type of resource
//
//   resource - the arvados resource to count
// return
//   count - the number of items of type resource the api server reports, if no error
//   err - error accessing the resource, or nil if no error
func NumberItemsAvailable(client arvadosclient.ArvadosClient, resource string) (count int, err error) {
	var response struct {
		ItemsAvailable int `json:"items_available"`
	}
	sdkParams := arvadosclient.Dict{"limit": 0}
	err = client.List(resource, sdkParams, &response)
	if err == nil {
		count = response.ItemsAvailable
	}
	return
}
