// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
)

type contextKey string

var contextKeyRequestID contextKey = "X-Request-Id"

func ContextWithRequestID(ctx context.Context, reqid string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, reqid)
}
