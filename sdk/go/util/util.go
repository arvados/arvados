/* Helper methods for dealing with responses from API Server. */

package util

import (
	"errors"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"log"
)

type SdkListResponse interface {
	NumItemsAvailable() (int, error)
	NumItemsContained() (int, error)
}

type UnstructuredSdkListResponse map[string]interface{}

func (m UnstructuredSdkListResponse) NumItemsAvailable() (numAvailable int, err error) {
	if itemsAvailable, ok := m["items_available"]; !ok {
		err = errors.New("Could not find \"items_available\" field in " +
			"UnstructuredSdkListResponse that NumItemsAvailable was called on.")
	} else {
		// TODO(misha): Check whether this assertion will work before casting
		numAvailable = int(itemsAvailable.(float64))
	}
	return
}

func (m UnstructuredSdkListResponse) NumItemsContained() (numContained int, err error) {
	if value, ok := m["items"]; ok {
		// TODO(misha): check whether this assertion will work before casting
		numContained = len(value.([]interface{}))
	} else {
		err = errors.New(`Could not find "items" field in ` +
			"UnstructuredSdkListResponse that NumItemsContained was called on.")
	}
	return
}

func UserIsAdmin(arv arvadosclient.ArvadosClient) (is_admin bool, err error) {
	type user struct {
		IsAdmin bool `json:"is_admin"`
	}
	var u user
	err = arv.Call("GET", "users", "", "current", nil, &u)
	return u.IsAdmin, err
}

// TODO(misha): Consider returning an error here instead of fatal'ing
func ContainsAllAvailableItems(response SdkListResponse) (containsAll bool, numContained int, numAvailable int) {
	var err error
	numContained, err = response.NumItemsContained()
	if err != nil {
		log.Fatalf("Error retrieving number of items contained in SDK response: %v",
			err)
	}
	numAvailable, err = response.NumItemsAvailable()
	if err != nil {
		log.Fatalf("Error retrieving number of items available from " +
			"SDK response: %v",
			err)
	}
	containsAll = numContained == numAvailable
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
