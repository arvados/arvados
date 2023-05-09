// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPStatusError interface {
	error
	HTTPStatus() int
}

func Errorf(status int, tmpl string, args ...interface{}) error {
	return errorWithStatus{fmt.Errorf(tmpl, args...), status}
}

func ErrorWithStatus(err error, status int) error {
	return errorWithStatus{err, status}
}

type errorWithStatus struct {
	error
	Status int
}

func (ews errorWithStatus) HTTPStatus() int {
	return ews.Status
}

type ErrorResponse struct {
	Errors []string `json:"errors"`
}

func Error(w http.ResponseWriter, error string, code int) {
	Errors(w, []string{error}, code)
}

func Errors(w http.ResponseWriter, errors []string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Errors: errors})
}
