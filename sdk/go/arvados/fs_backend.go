// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
	"errors"
	"io"
)

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
	BlockWrite(context.Context, BlockWriteOptions) (BlockWriteResponse, error)
	LocalLocator(locator string) (string, error)
}

type apiClient interface {
	RequestAndDecode(dst interface{}, method, path string, body io.Reader, params interface{}) error
}

var errStubClient = errors.New("stub client")

type StubClient struct{}

func (*StubClient) ReadAt(string, []byte, int) (int, error) { return 0, errStubClient }
func (*StubClient) LocalLocator(string) (string, error)     { return "", errStubClient }
func (*StubClient) BlockWrite(context.Context, BlockWriteOptions) (BlockWriteResponse, error) {
	return BlockWriteResponse{}, errStubClient
}
func (*StubClient) RequestAndDecode(_ interface{}, _, _ string, _ io.Reader, _ interface{}) error {
	return errStubClient
}
