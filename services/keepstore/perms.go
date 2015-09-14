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
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// The PermissionSecret is the secret key used to generate SHA1
// digests for permission hints. apiserver and Keep must use the same
// key.
var PermissionSecret []byte

// MakePermSignature returns a string representing the signed permission
// hint for the blob identified by blobHash, apiToken and expiration timestamp.
func MakePermSignature(blobHash string, apiToken string, expiry string) string {
	hmac := hmac.New(sha1.New, PermissionSecret)
	hmac.Write([]byte(blobHash))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(apiToken))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(expiry))
	digest := hmac.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

// SignLocator takes a blobLocator, an apiToken and an expiry time, and
// returns a signed locator string.
func SignLocator(blobLocator string, apiToken string, expiry time.Time) string {
	// If no permission secret or API token is available,
	// return an unsigned locator.
	if PermissionSecret == nil || apiToken == "" {
		return blobLocator
	}
	// Extract the hash from the blob locator, omitting any size hint that may be present.
	blobHash := strings.Split(blobLocator, "+")[0]
	// Return the signed locator string.
	timestampHex := fmt.Sprintf("%08x", expiry.Unix())
	return blobLocator +
		"+A" + MakePermSignature(blobHash, apiToken, timestampHex) +
		"@" + timestampHex
}

var signedLocatorRe = regexp.MustCompile(`^([[:xdigit:]]{32}).*\+A([[:xdigit:]]{40})@([[:xdigit:]]{8})`)

// VerifySignature returns nil if the signature on the signedLocator
// can be verified using the given apiToken. Otherwise it returns
// either ExpiredError (if the timestamp has expired, which is
// something the client could have figured out independently) or
// PermissionError.
func VerifySignature(signedLocator string, apiToken string) error {
	matches := signedLocatorRe.FindStringSubmatch(signedLocator)
	if matches == nil {
		// Could not find a permission signature at all
		return PermissionError
	}
	blobHash := matches[1]
	sigHex := matches[2]
	expHex := matches[3]
	if expTime, err := ParseHexTimestamp(expHex); err != nil {
		return PermissionError
	} else if expTime.Before(time.Now()) {
		return ExpiredError
	}
	if sigHex != MakePermSignature(blobHash, apiToken, expHex) {
		return PermissionError
	}
	return nil
}

// ParseHexTimestamp parses timestamp
func ParseHexTimestamp(timestampHex string) (ts time.Time, err error) {
	if tsInt, e := strconv.ParseInt(timestampHex, 16, 0); e == nil {
		ts = time.Unix(tsInt, 0)
	} else {
		err = e
	}
	return ts, err
}
