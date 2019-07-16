// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
)

type contextKeyRequestID struct{}
type contextKeyAuthorization struct{}

func ContextWithRequestID(ctx context.Context, reqid string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID{}, reqid)
}

// ContextWithAuthorization returns a child context that (when used
// with (*Client)RequestAndDecodeContext) sends the given
// Authorization header value instead of the Client's default
// AuthToken.
func ContextWithAuthorization(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, contextKeyAuthorization{}, value)
}
