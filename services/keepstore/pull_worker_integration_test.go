// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

type PullWorkIntegrationTestData struct {
	Name     string
	Locator  string
	Content  string
	GetError string
}

func (s *HandlerSuite) setupPullWorkerIntegrationTest(c *check.C, testData PullWorkIntegrationTestData, wantData bool) PullRequest {
	arvadostest.StartKeep(2, false)
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	// Put content if the test needs it
	if wantData {
		locator, _, err := s.handler.keepClient.PutB([]byte(testData.Content))
		if err != nil {
			c.Errorf("Error putting test data in setup for %s %s %v", testData.Content, locator, err)
		}
		if locator == "" {
			c.Errorf("No locator found after putting test data")
		}
	}

	// Create pullRequest for the test
	pullRequest := PullRequest{
		Locator: testData.Locator,
	}
	return pullRequest
}

// Do a get on a block that is not existing in any of the keep servers.
// Expect "block not found" error.
func (s *HandlerSuite) TestPullWorkerIntegration_GetNonExistingLocator(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	testData := PullWorkIntegrationTestData{
		Name:     "TestPullWorkerIntegration_GetLocator",
		Locator:  "5d41402abc4b2a76b9719d911017c592",
		Content:  "hello",
		GetError: "Block not found",
	}

	pullRequest := s.setupPullWorkerIntegrationTest(c, testData, false)
	defer arvadostest.StopKeep(2)

	s.performPullWorkerIntegrationTest(testData, pullRequest, c)
}

// Do a get on a block that exists on one of the keep servers.
// The setup method will create this block before doing the get.
func (s *HandlerSuite) TestPullWorkerIntegration_GetExistingLocator(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	testData := PullWorkIntegrationTestData{
		Name:     "TestPullWorkerIntegration_GetLocator",
		Locator:  "5d41402abc4b2a76b9719d911017c592",
		Content:  "hello",
		GetError: "",
	}

	pullRequest := s.setupPullWorkerIntegrationTest(c, testData, true)
	defer arvadostest.StopKeep(2)

	s.performPullWorkerIntegrationTest(testData, pullRequest, c)
}

// Perform the test.
// The test directly invokes the "PullItemAndProcess" rather than
// putting an item on the pullq so that the errors can be verified.
func (s *HandlerSuite) performPullWorkerIntegrationTest(testData PullWorkIntegrationTestData, pullRequest PullRequest, c *check.C) {

	// Override writePulledBlock to mock PutBlock functionality
	defer func(orig func(*RRVolumeManager, Volume, []byte, string) error) { writePulledBlock = orig }(writePulledBlock)
	writePulledBlock = func(_ *RRVolumeManager, _ Volume, content []byte, _ string) error {
		c.Check(string(content), check.Equals, testData.Content)
		return nil
	}

	// Override GetContent to mock keepclient Get functionality
	defer func(orig func(string, *keepclient.KeepClient) (io.ReadCloser, int64, string, error)) {
		GetContent = orig
	}(GetContent)
	GetContent = func(signedLocator string, keepClient *keepclient.KeepClient) (reader io.ReadCloser, contentLength int64, url string, err error) {
		if testData.GetError != "" {
			return nil, 0, "", errors.New(testData.GetError)
		}
		rdr := ioutil.NopCloser(bytes.NewBufferString(testData.Content))
		return rdr, int64(len(testData.Content)), "", nil
	}

	err := s.handler.pullItemAndProcess(pullRequest)

	if len(testData.GetError) > 0 {
		if (err == nil) || (!strings.Contains(err.Error(), testData.GetError)) {
			c.Errorf("Got error %v, expected %v", err, testData.GetError)
		}
	} else {
		if err != nil {
			c.Errorf("Got error %v, expected nil", err)
		}
	}
}
