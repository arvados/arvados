package main

import (
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"time"
)

// SignLocator takes a blobLocator, an apiToken and an expiry time, and
// returns a signed locator string.
func SignLocator(blobLocator, apiToken string, expiry time.Time) string {
	return keepclient.SignLocator(blobLocator, apiToken, expiry, theConfig.BlobSignatureTTL.Duration(), theConfig.blobSigningKey)
}

// VerifySignature returns nil if the signature on the signedLocator
// can be verified using the given apiToken. Otherwise it returns
// either ExpiredError (if the timestamp has expired, which is
// something the client could have figured out independently) or
// PermissionError.
func VerifySignature(signedLocator, apiToken string) error {
	err := keepclient.VerifySignature(signedLocator, apiToken, theConfig.BlobSignatureTTL.Duration(), theConfig.blobSigningKey)
	if err == keepclient.ErrSignatureExpired {
		return ExpiredError
	} else if err != nil {
		return PermissionError
	}
	return nil
}
