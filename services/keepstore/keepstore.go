// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"time"
)

// BlockSize for a Keep "block" is 64MB.
const BlockSize = 64 * 1024 * 1024

// MinFreeKilobytes is the amount of space a Keep volume must have available
// in order to permit writes.
const MinFreeKilobytes = BlockSize / 1024

var bufs *bufferPool

// KeepError types.
//
type KeepError struct {
	HTTPCode int
	ErrMsg   string
}

var (
	BadRequestError     = &KeepError{400, "Bad Request"}
	UnauthorizedError   = &KeepError{401, "Unauthorized"}
	CollisionError      = &KeepError{500, "Collision"}
	RequestHashError    = &KeepError{422, "Hash mismatch in request"}
	PermissionError     = &KeepError{403, "Forbidden"}
	DiskHashError       = &KeepError{500, "Hash mismatch in stored data"}
	ExpiredError        = &KeepError{401, "Expired permission signature"}
	NotFoundError       = &KeepError{404, "Not Found"}
	VolumeBusyError     = &KeepError{503, "Volume backend busy"}
	GenericError        = &KeepError{500, "Fail"}
	FullError           = &KeepError{503, "Full"}
	SizeRequiredError   = &KeepError{411, "Missing Content-Length"}
	TooLongError        = &KeepError{413, "Block is too large"}
	MethodDisabledError = &KeepError{405, "Method disabled"}
	ErrNotImplemented   = &KeepError{500, "Unsupported configuration"}
	ErrClientDisconnect = &KeepError{503, "Client disconnected"}
)

func (e *KeepError) Error() string {
	return e.ErrMsg
}

// Periodically (once per interval) invoke EmptyTrash on all volumes.
func emptyTrash(mounts []*VolumeMount, interval time.Duration) {
	for range time.NewTicker(interval).C {
		for _, v := range mounts {
			v.EmptyTrash()
		}
	}
}
