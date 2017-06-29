// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

//  TestIdenticalTimestamps ensures EachCollection returns the same
//  set of collections for various page sizes -- even page sizes so
//  small that we get entire pages full of collections with identical
//  timestamps and exercise our gettingExactTimestamp cases.
func (s *integrationSuite) TestIdenticalTimestamps(c *check.C) {
	// pageSize==0 uses the default (large) page size.
	pageSizes := []int{0, 2, 3, 4, 5}
	got := make([][]string, len(pageSizes))
	var wg sync.WaitGroup
	for trial, pageSize := range pageSizes {
		wg.Add(1)
		go func(trial, pageSize int) {
			defer wg.Done()
			streak := 0
			longestStreak := 0
			var lastMod time.Time
			sawUUID := make(map[string]bool)
			err := EachCollection(&s.config.Client, pageSize, func(c arvados.Collection) error {
				got[trial] = append(got[trial], c.UUID)
				if c.ModifiedAt == nil {
					return nil
				}
				if sawUUID[c.UUID] {
					// dup
					return nil
				}
				sawUUID[c.UUID] = true
				if lastMod == *c.ModifiedAt {
					streak++
					if streak > longestStreak {
						longestStreak = streak
					}
				} else {
					streak = 0
					lastMod = *c.ModifiedAt
				}
				return nil
			}, nil)
			c.Check(err, check.IsNil)
			c.Check(longestStreak > 25, check.Equals, true)
		}(trial, pageSize)
	}
	wg.Wait()
	for trial := 1; trial < len(pageSizes); trial++ {
		c.Check(got[trial], check.DeepEquals, got[0])
	}
}
