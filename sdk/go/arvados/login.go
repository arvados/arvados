// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"net/http"
)

type LoginResponse struct {
	RedirectLocation string
	HTML             bytes.Buffer
}

func (resp LoginResponse) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if resp.RedirectLocation != "" {
		w.Header().Set("Location", resp.RedirectLocation)
		w.WriteHeader(http.StatusFound)
	} else {
		w.Write(resp.HTML.Bytes())
	}
}
