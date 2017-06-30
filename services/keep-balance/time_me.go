// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"log"
	"time"
)

func timeMe(logger *log.Logger, label string) func() {
	t0 := time.Now()
	logger.Printf("%s: start", label)
	return func() {
		logger.Printf("%s: took %v", label, time.Since(t0))
	}
}
