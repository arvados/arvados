// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	log "github.com/Sirupsen/logrus"
)

func init() {
	theConfig.debugLogf = log.Printf
}
