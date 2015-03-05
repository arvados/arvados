package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io"
	"io/ioutil"
	"log"
	"time"
)

/*
	Keepstore initiates pull worker channel goroutine.
	The channel will process pull list.
		For each (next) pull request:
			For each locator listed, execute Pull on the server(s) listed
			Skip the rest of the servers if no errors
		Repeat
*/
func RunPullWorker(pullq *WorkQueue, keepClient keepclient.KeepClient) {
	nextItem := pullq.NextItem
	for item := range nextItem {
		Pull(item.(PullRequest), keepClient)
	}
}

/*
	For each Pull request:
		Generate a random API token.
		Generate a permission signature using this token, timestamp ~60 seconds in the future, and desired block hash.
		Using this token & signature, retrieve the given block.
		Write to storage
*/
func Pull(pullRequest PullRequest, keepClient keepclient.KeepClient) (err error) {
	defer func() {
		if err == nil {
			log.Printf("Pull %s success", pullRequest)
		} else {
			log.Printf("Pull %s error: %s", pullRequest, err)
		}
	}()

	service_roots := make(map[string]string)
	for _, addr := range pullRequest.Servers {
		service_roots[addr] = addr
	}
	keepClient.SetServiceRoots(service_roots)

	// Generate signature with a random token
	expires_at := time.Now().Add(60 * time.Second)
	signedLocator := SignLocator(pullRequest.Locator, GenerateRandomApiToken(), expires_at)

	reader, contentLen, _, err := GetContent(signedLocator, keepClient)
	if err != nil {
		return
	}
	if reader == nil {
		return errors.New(fmt.Sprintf("No reader found for : %s", signedLocator))
	}
	defer reader.Close()

	read_content, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	if (read_content == nil) || (int64(len(read_content)) != contentLen) {
		return errors.New(fmt.Sprintf("Content not found for: %s", signedLocator))
	}

	err = PutContent(read_content, pullRequest.Locator)
	return
}

// Fetch the content for the given locator using keepclient.
var GetContent = func(signedLocator string, keepClient keepclient.KeepClient) (
	reader io.ReadCloser, contentLength int64, url string, err error) {
	reader, blocklen, url, err := keepClient.Get(signedLocator)
	return reader, blocklen, url, err
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
