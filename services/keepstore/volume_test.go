// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"time"
)

var (
	TestBlock = []byte("The quick brown fox jumps over the lazy dog.")
	TestHash  = "e4d909c290d0fb1ca068ffaddf22cbd0"

	TestBlock2 = []byte("Pack my box with five dozen liquor jugs.")
	TestHash2  = "f15ac516f788aec4f30932ffb6395c39"

	TestBlock3 = []byte("Now is the time for all good men to come to the aid of their country.")
	TestHash3  = "eed29bbffbc2dbe5e5ee0bb71888e61f"

	EmptyHash  = "d41d8cd98f00b204e9800998ecf8427e"
	EmptyBlock = []byte("")
)

// A TestableVolume allows test suites to manipulate the state of an
// underlying Volume, in order to test behavior in cases that are
// impractical to achieve with a sequence of normal Volume operations.
type TestableVolume interface {
	volume

	// Returns the strings that a driver uses to record read/write operations.
	ReadWriteOperationLabelValues() (r, w string)

	// Specify the value Mtime() should return, until the next
	// call to Touch, TouchWithDate, or BlockWrite.
	TouchWithDate(locator string, lastBlockWrite time.Time)

	// Clean up, delete temporary files.
	Teardown()
}
