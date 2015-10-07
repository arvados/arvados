/*
Permissions management on Arvados locator hashes.

The permissions structure for Arvados is as follows (from
https://arvados.org/issues/2328)

A Keep locator string has the following format:

    [hash]+[size]+A[signature]@[timestamp]

The "signature" string here is a cryptographic hash, expressed as a
string of hexadecimal digits, and timestamp is a 32-bit Unix timestamp
expressed as a hexadecimal number.  e.g.:

    acbd18db4cc2f85cedef654fccc4a4d8+3+A257f3f5f5f0a4e4626a18fc74bd42ec34dcb228a@7fffffff

The signature represents a guarantee that this locator was generated
by either Keep or the API server for use with the supplied API token.
If a request to Keep includes a locator with a valid signature and is
accompanied by the proper API token, the user has permission to GET
that object.

The signature may be generated either by Keep (after the user writes a
block) or by the API server (if the user has can_read permission on
the specified object). Keep and API server share a secret that is used
to generate signatures.

To verify a permission hint, Keep generates a new hint for the
requested object (using the locator string, the timestamp, the
permission secret and the user's API token, which must appear in the
request headers) and compares it against the hint included in the
request. If the permissions do not match, or if the API token is not
present, Keep returns a 401 error.
*/

package main

import (
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"time"
)

// The PermissionSecret is the secret key used to generate SHA1
// digests for permission hints. apiserver and Keep must use the same
// key.
var PermissionSecret []byte

// SignLocator takes a blobLocator, an apiToken and an expiry time, and
// returns a signed locator string.
func SignLocator(blobLocator string, apiToken string, expiry time.Time) string {
	return keepclient.SignLocator(blobLocator, apiToken, expiry, PermissionSecret)
}

// VerifySignature returns nil if the signature on the signedLocator
// can be verified using the given apiToken. Otherwise it returns
// either ExpiredError (if the timestamp has expired, which is
// something the client could have figured out independently) or
// PermissionError.
func VerifySignature(signedLocator string, apiToken string) error {
	err := keepclient.VerifySignature(signedLocator, apiToken, PermissionSecret)
	if err != nil {
		if err == keepclient.PermissionError {
			return PermissionError
		} else if err == keepclient.ExpiredError {
			return ExpiredError
		}
	}
	return err
}
