// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net/http"
	"testing"
)

func TestLoggingResponseWriterImplementsCloseNotifier(t *testing.T) {
	http.ResponseWriter(&LoggingResponseWriter{}).(http.CloseNotifier).CloseNotify()
}
