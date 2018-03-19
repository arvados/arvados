// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"github.com/Sirupsen/logrus"
)

func init() {
	log.Level = logrus.DebugLevel
	theConfig.debugLogf = log.Printf
}
