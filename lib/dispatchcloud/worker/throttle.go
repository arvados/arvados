// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"fmt"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"github.com/sirupsen/logrus"
)

type throttle struct {
	err   error
	until time.Time
	mtx   sync.Mutex
}

// CheckRateLimitError checks whether the given error is a
// cloud.RateLimitError, and if so, ensures Error() returns a non-nil
// error until the rate limiting holdoff period expires.
//
// If a notify func is given, it will be called after the holdoff
// period expires.
func (thr *throttle) CheckRateLimitError(err error, logger logrus.FieldLogger, callType string, notify func()) {
	rle, ok := err.(cloud.RateLimitError)
	if !ok {
		return
	}
	until := rle.EarliestRetry()
	if !until.After(time.Now()) {
		return
	}
	dur := until.Sub(time.Now())
	logger.WithFields(logrus.Fields{
		"CallType": callType,
		"Duration": dur,
		"ResumeAt": until,
	}).Info("suspending remote calls due to rate-limit error")
	thr.ErrorUntil(fmt.Errorf("remote calls are suspended for %s, until %s", dur, until), until, notify)
}

func (thr *throttle) ErrorUntil(err error, until time.Time, notify func()) {
	thr.mtx.Lock()
	defer thr.mtx.Unlock()
	thr.err, thr.until = err, until
	if notify != nil {
		time.AfterFunc(until.Sub(time.Now()), notify)
	}
}

func (thr *throttle) Error() error {
	thr.mtx.Lock()
	defer thr.mtx.Unlock()
	if thr.err != nil && time.Now().After(thr.until) {
		thr.err = nil
	}
	return thr.err
}

type throttledInstanceSet struct {
	cloud.InstanceSet
	throttleCreate    throttle
	throttleInstances throttle
}
