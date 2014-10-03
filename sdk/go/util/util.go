/* Helper methods for dealing with responses from API Server. */

package util

import (
	"errors"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
)

func UserIsAdmin(arv arvadosclient.ArvadosClient) (is_admin bool, err error) {
	type user struct {
		IsAdmin bool `json:"is_admin"`
	}
	var u user
	err = arv.Call("GET", "users", "", "current", nil, &u)
	return u.IsAdmin, err
}

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
