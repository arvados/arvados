// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"git.arvados.org/arvados.git/sdk/go/keepclient"
)

// RunPullWorker receives PullRequests from pullq, invokes
// PullItemAndProcess on each one. After each PR, it logs a message
// indicating whether the pull was successful.
func (h *handler) runPullWorker(pullq *WorkQueue) {
	for item := range pullq.NextItem {
		pr := item.(PullRequest)
		err := h.pullItemAndProcess(pr)
		pullq.DoneItem <- struct{}{}
		if err == nil {
			h.Logger.Printf("Pull %s success", pr)
		} else {
			h.Logger.Printf("Pull %s error: %s", pr, err)
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
func (h *handler) pullItemAndProcess(pullRequest PullRequest) error {
	var vol *VolumeMount
	if uuid := pullRequest.MountUUID; uuid != "" {
		vol = h.volmgr.Lookup(pullRequest.MountUUID, true)
		if vol == nil {
			return fmt.Errorf("pull req has nonexistent mount: %v", pullRequest)
		}
	}

	// Make a private copy of keepClient so we can set
	// ServiceRoots to the source servers specified in the pull
	// request.
	keepClient := *h.keepClient
	serviceRoots := make(map[string]string)
	for _, addr := range pullRequest.Servers {
		serviceRoots[addr] = addr
	}
	keepClient.SetServiceRoots(serviceRoots, nil, nil)

	signedLocator := SignLocator(h.Cluster, pullRequest.Locator, keepClient.Arvados.ApiToken, time.Now().Add(time.Minute))

	reader, contentLen, _, err := GetContent(signedLocator, &keepClient)
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

	return writePulledBlock(h.volmgr, vol, readContent, pullRequest.Locator)
}

// GetContent fetches the content for the given locator using keepclient.
var GetContent = func(signedLocator string, keepClient *keepclient.KeepClient) (io.ReadCloser, int64, string, error) {
	return keepClient.Get(signedLocator)
}

var writePulledBlock = func(volmgr *RRVolumeManager, volume Volume, data []byte, locator string) error {
	if volume != nil {
		return volume.Put(context.Background(), locator, data)
	}
	_, err := PutBlock(context.Background(), volmgr, data, locator, nil)
	return err
}
