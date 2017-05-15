package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/keepclient"

	log "github.com/Sirupsen/logrus"
)

// RunPullWorker is used by Keepstore to initiate pull worker channel goroutine.
//	The channel will process pull list.
//		For each (next) pull request:
//			For each locator listed, execute Pull on the server(s) listed
//			Skip the rest of the servers if no errors
//		Repeat
//
func RunPullWorker(pullq *WorkQueue, keepClient *keepclient.KeepClient) {
	nextItem := pullq.NextItem
	for item := range nextItem {
		pullRequest := item.(PullRequest)
		err := PullItemAndProcess(item.(PullRequest), keepClient)
		pullq.DoneItem <- struct{}{}
		if err == nil {
			log.Printf("Pull %s success", pullRequest)
		} else {
			log.Printf("Pull %s error: %s", pullRequest, err)
		}
	}
}

// PullItemAndProcess pulls items from PullQueue and processes them.
//	For each Pull request:
//		Generate a random API token.
//		Generate a permission signature using this token, timestamp ~60 seconds in the future, and desired block hash.
//		Using this token & signature, retrieve the given block.
//		Write to storage
//
func PullItemAndProcess(pullRequest PullRequest, keepClient *keepclient.KeepClient) error {
	var vol Volume
	if uuid := pullRequest.MountUUID; uuid != "" {
		vol = KeepVM.Lookup(pullRequest.MountUUID, true)
		if vol == nil {
			return fmt.Errorf("pull req has nonexistent mount: %v", pullRequest)
		}
	}

	keepClient.Arvados.ApiToken = randomToken

	serviceRoots := make(map[string]string)
	for _, addr := range pullRequest.Servers {
		serviceRoots[addr] = addr
	}
	keepClient.SetServiceRoots(serviceRoots, nil, nil)

	// Generate signature with a random token
	expiresAt := time.Now().Add(60 * time.Second)
	signedLocator := SignLocator(pullRequest.Locator, randomToken, expiresAt)

	reader, contentLen, _, err := GetContent(signedLocator, keepClient)
	if err != nil {
		return err
	}
	if reader == nil {
		return fmt.Errorf("No reader found for : %s", signedLocator)
	}
	defer reader.Close()

	readContent, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	if (readContent == nil) || (int64(len(readContent)) != contentLen) {
		return fmt.Errorf("Content not found for: %s", signedLocator)
	}

	writePulledBlock(vol, readContent, pullRequest.Locator)
	return nil
}

// Fetch the content for the given locator using keepclient.
var GetContent = func(signedLocator string, keepClient *keepclient.KeepClient) (io.ReadCloser, int64, string, error) {
	return keepClient.Get(signedLocator)
}

var writePulledBlock = func(volume Volume, data []byte, locator string) {
	var err error
	if volume != nil {
		err = volume.Put(context.Background(), locator, data)
	} else {
		_, err = PutBlock(context.Background(), data, locator)
	}
	if err != nil {
		log.Printf("error writing pulled block %q: %s", locator, err)
	}
}

var randomToken = func() string {
	const alphaNumeric = "0123456789abcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, 36)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphaNumeric[b%byte(len(alphaNumeric))]
	}
	return (string(bytes))
}()
