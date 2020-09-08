// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

func semaphore(max int) (acquire, release func()) {
	if max > 0 {
		ch := make(chan bool, max)
		return func() { ch <- true }, func() { <-ch }
	}
	return func() {}, func() {}
}
