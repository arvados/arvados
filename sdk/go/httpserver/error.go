// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Errors []string `json:"errors"`
}

func Error(w http.ResponseWriter, error string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{Errors: []string{error}})
}
