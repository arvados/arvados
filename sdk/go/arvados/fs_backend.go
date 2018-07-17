// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "io"

type fsBackend interface {
	keepClient
	apiClient
}

// Ideally *Client would do everything; meanwhile keepBackend
// implements fsBackend by merging the two kinds of arvados client.
type keepBackend struct {
	keepClient
	apiClient
}

type keepClient interface {
	ReadAt(locator string, p []byte, off int) (int, error)
	PutB(p []byte) (string, int, error)
}

type apiClient interface {
	RequestAndDecode(dst interface{}, method, path string, body io.Reader, params interface{}) error
	UpdateBody(rsc resource) io.Reader
}
