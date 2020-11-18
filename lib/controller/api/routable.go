// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package api provides types used by controller/server-component
// packages.
package api

import "context"

// A RoutableFunc calls an API method (sometimes via a wrapped
// RoutableFunc) that has real argument types.
//
// (It is used by ctrlctx to manage database transactions, so moving
// it to the router package would cause a circular dependency
// router->arvadostest->ctrlctx->router.)
type RoutableFunc func(ctx context.Context, opts interface{}) (interface{}, error)

type RoutableFuncWrapper func(RoutableFunc) RoutableFunc

// ComposeWrappers (w1, w2, w3, ...) returns a RoutableFuncWrapper that
// composes w1, w2, w3, ... such that w1 is the outermost wrapper.
func ComposeWrappers(wraps ...RoutableFuncWrapper) RoutableFuncWrapper {
	return func(f RoutableFunc) RoutableFunc {
		for i := len(wraps) - 1; i >= 0; i-- {
			f = wraps[i](f)
		}
		return f
	}
}
