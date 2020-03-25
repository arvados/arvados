// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"testing"

	check "gopkg.in/check.v1"
)

func TestGocheck(t *testing.T) {
	check.TestingT(t)
}

func init() {
	testMode = true
}
