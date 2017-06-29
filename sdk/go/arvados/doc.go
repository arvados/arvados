// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package arvados is a client library for Arvados.
//
// The API is not stable: it should be considered experimental
// pre-release.
//
// The intent is to offer model types and API call functions that can
// be generated automatically (or at least mostly automatically) from
// a discovery document. For the time being, there is a manually
// generated subset of those types and API calls with (approximately)
// the right signatures, plus client/authentication support and some
// convenience functions.
package arvados
