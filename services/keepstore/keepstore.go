// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"net/http"
	"time"
)

// BlockSize for a Keep "block" is 64MB.
const BlockSize = 64 * 1024 * 1024

// MinFreeKilobytes is the amount of space a Keep volume must have available
// in order to permit writes.
const MinFreeKilobytes = BlockSize / 1024

var bufs *bufferPool

type KeepError struct {
	HTTPCode int
	ErrMsg   string
}

var (
	BadRequestError     = &KeepError{http.StatusBadRequest, "Bad Request"}
	UnauthorizedError   = &KeepError{http.StatusUnauthorized, "Unauthorized"}
	CollisionError      = &KeepError{http.StatusInternalServerError, "Collision"}
	RequestHashError    = &KeepError{http.StatusUnprocessableEntity, "Hash mismatch in request"}
	PermissionError     = &KeepError{http.StatusForbidden, "Forbidden"}
	DiskHashError       = &KeepError{http.StatusInternalServerError, "Hash mismatch in stored data"}
	ExpiredError        = &KeepError{http.StatusUnauthorized, "Expired permission signature"}
	NotFoundError       = &KeepError{http.StatusNotFound, "Not Found"}
	VolumeBusyError     = &KeepError{http.StatusServiceUnavailable, "Volume backend busy"}
	GenericError        = &KeepError{http.StatusInternalServerError, "Fail"}
	FullError           = &KeepError{http.StatusInsufficientStorage, "Full"}
	SizeRequiredError   = &KeepError{http.StatusLengthRequired, "Missing Content-Length"}
	TooLongError        = &KeepError{http.StatusRequestEntityTooLarge, "Block is too large"}
	MethodDisabledError = &KeepError{http.StatusMethodNotAllowed, "Method disabled"}
	ErrNotImplemented   = &KeepError{http.StatusInternalServerError, "Unsupported configuration"}
	ErrClientDisconnect = &KeepError{499, "Client disconnected"} // non-RFC Nginx status code
)

func (e *KeepError) Error() string {
	return e.ErrMsg
}

// Periodically (once per interval) invoke EmptyTrash on all volumes.
func emptyTrash(mounts []*VolumeMount, interval time.Duration) {
	for range time.NewTicker(interval).C {
		for _, v := range mounts {
			if v.KeepMount.AllowTrash {
				v.EmptyTrash()
			}
		}
	}
}
