// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type LoginResponse struct {
	RedirectLocation string       `json:"redirect_location,omitempty"`
	Token            string       `json:"token,omitempty"`
	Message          string       `json:"message,omitempty"`
	HTML             bytes.Buffer `json:"-"`
}

func (resp LoginResponse) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if resp.RedirectLocation != "" {
		w.Header().Set("Location", resp.RedirectLocation)
		w.WriteHeader(http.StatusFound)
	} else if resp.Token != "" || resp.Message != "" {
		w.Header().Set("Content-Type", "application/json")
		if resp.Token == "" {
			w.WriteHeader(http.StatusUnauthorized)
		}
		json.NewEncoder(w).Encode(resp)
	} else {
		w.Header().Set("Content-Type", "text/html")
		w.Write(resp.HTML.Bytes())
	}
}

type LogoutResponse struct {
	RedirectLocation string
}

func (resp LogoutResponse) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Location", resp.RedirectLocation)
	w.WriteHeader(http.StatusFound)
}
