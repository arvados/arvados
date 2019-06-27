// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

type errorWithStatus struct {
	code int
	error
}

func (err errorWithStatus) HTTPStatus() int {
	return err.code
}

func httpError(code int, err error) error {
	return errorWithStatus{code: code, error: err}
}
