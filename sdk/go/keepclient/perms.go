// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import "git.arvados.org/arvados.git/sdk/go/arvados"

var (
	ErrSignatureExpired = arvados.ErrSignatureExpired
	ErrSignatureInvalid = arvados.ErrSignatureInvalid
	ErrSignatureMissing = arvados.ErrSignatureMissing
	SignLocator         = arvados.SignLocator
	SignedLocatorRe     = arvados.SignedLocatorRe
	VerifySignature     = arvados.VerifySignature
)
