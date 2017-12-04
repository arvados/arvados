// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type TransactionError struct {
	Method     string
	URL        url.URL
	StatusCode int
	Status     string
	Errors     []string
}

func (e TransactionError) Error() (s string) {
	s = fmt.Sprintf("request failed: %s", e.URL.String())
	if e.Status != "" {
		s = s + ": " + e.Status
	}
	if len(e.Errors) > 0 {
		s = s + ": " + strings.Join(e.Errors, "; ")
	}
	return
}

func newTransactionError(req *http.Request, resp *http.Response, buf []byte) *TransactionError {
	var e TransactionError
	if json.Unmarshal(buf, &e) != nil {
		// No JSON-formatted error response
		e.Errors = nil
	}
	e.Method = req.Method
	e.URL = *req.URL
	if resp != nil {
		e.Status = resp.Status
		e.StatusCode = resp.StatusCode
	}
	return &e
}
