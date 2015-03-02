package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var arv arvadosclient.ArvadosClient
var keepClient keepclient.KeepClient

/*
	Keepstore initiates pull worker channel goroutine.
	The channel will process pull list.
		For each (next) pull request:
			For each locator listed, execute Pull on the server(s) listed
			Skip the rest of the servers if no errors
		Repeat
*/
func RunPullWorker(nextItem <-chan interface{}) {
	var err error
	arv, err = arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Fatalf("Error setting up arvados client %s", err.Error())
	}
	arv.ApiToken = os.Getenv("ARVADOS_API_TOKEN")

	keepClient, err = keepclient.MakeKeepClient(&arv)
	if err != nil {
		log.Fatalf("Error setting up keep client %s", err.Error())
	}

	for item := range nextItem {
		pullReq := item.(PullRequest)
		for _, addr := range pullReq.Servers {
			err := Pull(addr, pullReq.Locator)
			if err == nil {
				break
			}
		}
	}
}

/*
	For each Pull request:
		Generate a random API token.
		Generate a permission signature using this token, timestamp ~60 seconds in the future, and desired block hash.
		Using this token & signature, retrieve the given block.
		Write to storage
*/
func Pull(addr string, locator string) (err error) {
	log.Printf("Pull %s/%s starting", addr, locator)

	defer func() {
		if err == nil {
			log.Printf("Pull %s/%s success", addr, locator)
		} else {
			log.Printf("Pull %s/%s error: %s", addr, locator, err)
		}
	}()

	service_roots := make(map[string]string)
	service_roots[locator] = addr
	keepClient.SetServiceRoots(service_roots)

	read_content, err := GetContent(addr, locator)
	log.Print(read_content, err)
	if err != nil {
		return
	}

	err = PutContent(read_content, locator)
	return
}

// Fetch the content for the given locator using keepclient.
var GetContent = func(addr string, locator string) ([]byte, error) {
	// Generate signature with a random token
	PermissionSecret = []byte(os.Getenv("ARVADOS_API_TOKEN"))
	expires_at := time.Now().Add(60 * time.Second)
	signedLocator := SignLocator(locator, GenerateRandomApiToken(), expires_at)
	reader, blocklen, _, err := keepClient.Get(signedLocator)
	defer reader.Close()
	if err != nil {
		return nil, err
	}

	read_content, err := ioutil.ReadAll(reader)
	log.Print(read_content, err)
	if err != nil {
		return nil, err
	}

	if (read_content == nil) || (int64(len(read_content)) != blocklen) {
		return nil, errors.New(fmt.Sprintf("Content not found for: %s", signedLocator))
	}

	return read_content, nil
}

const ALPHA_NUMERIC = "0123456789abcdefghijklmnopqrstuvwxyz"

func GenerateRandomApiToken() string {
	var bytes = make([]byte, 36)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = ALPHA_NUMERIC[b%byte(len(ALPHA_NUMERIC))]
	}
	return (string(bytes))
}

// Put block
var PutContent = func(content []byte, locator string) (err error) {
	err = PutBlock(content, locator)
	return
}
