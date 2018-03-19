// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

// RunPullWorker receives PullRequests from pullq, invokes
// PullItemAndProcess on each one. After each PR, it logs a message
// indicating whether the pull was successful.
func RunPullWorker(pullq *WorkQueue, keepClient *keepclient.KeepClient) {
	for item := range pullq.NextItem {
		pr := item.(PullRequest)
		err := PullItemAndProcess(pr, keepClient)
		pullq.DoneItem <- struct{}{}
		if err == nil {
			log.Printf("Pull %s success", pr)
		} else {
			log.Printf("Pull %s error: %s", pr, err)
		}
	}
}

// PullItemAndProcess executes a pull request by retrieving the
// specified block from one of the specified servers, and storing it
// on a local volume.
//
// If the PR specifies a non-blank mount UUID, PullItemAndProcess will
// only attempt to write the data to the corresponding
// volume. Otherwise it writes to any local volume, as a PUT request
// would.
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
