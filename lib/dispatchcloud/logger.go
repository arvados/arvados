// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"sync"
	"time"
)

type logger interface {
	Printf(string, ...interface{})
	Warnf(string, ...interface{})
	Debugf(string, ...interface{})
}

var nextSpam = map[string]time.Time{}
var nextSpamMtx sync.Mutex

func unspam(msg string) bool {
	nextSpamMtx.Lock()
	defer nextSpamMtx.Unlock()
	if nextSpam[msg].Before(time.Now()) {
		nextSpam[msg] = time.Now().Add(time.Minute)
		return true
	}
	return false
}
