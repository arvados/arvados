// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
)

// SignLocator takes a blobLocator, an apiToken and an expiry time, and
// returns a signed locator string.
func SignLocator(cluster *arvados.Cluster, blobLocator, apiToken string, expiry time.Time) string {
	return keepclient.SignLocator(blobLocator, apiToken, expiry, cluster.Collections.BlobSigningTTL.Duration(), []byte(cluster.Collections.BlobSigningKey))
}

// VerifySignature returns nil if the signature on the signedLocator
// can be verified using the given apiToken. Otherwise it returns
// either ExpiredError (if the timestamp has expired, which is
// something the client could have figured out independently) or
// PermissionError.
func VerifySignature(cluster *arvados.Cluster, signedLocator, apiToken string) error {
	err := keepclient.VerifySignature(signedLocator, apiToken, cluster.Collections.BlobSigningTTL.Duration(), []byte(cluster.Collections.BlobSigningKey))
	if err == keepclient.ErrSignatureExpired {
		return ExpiredError
	} else if err != nil {
		return PermissionError
	}
	return nil
}
