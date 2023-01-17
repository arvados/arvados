// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package cloud

import (
	"sort"
)

// NormalizePriceHistory de-duplicates and sorts instance prices, most
// recent first.
//
// The provided slice is modified in place.
func NormalizePriceHistory(prices []InstancePrice) []InstancePrice {
	// sort by timestamp, newest first
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].StartTime.After(prices[j].StartTime)
	})
	// remove duplicate data points, keeping the oldest
	for i := 0; i < len(prices)-1; i++ {
		if prices[i].StartTime == prices[i+1].StartTime || prices[i].Price == prices[i+1].Price {
			prices = append(prices[:i], prices[i+1:]...)
			i--
		}
	}
	return prices
}
