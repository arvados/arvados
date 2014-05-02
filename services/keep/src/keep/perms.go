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
by either Keep or the API server for the user with the supplied API
token.  If a request to Keep includes a locator with a valid signature
and is accompanied by the proper API token, the user has permission to
perform any action on that object (GET, PUT or DELETE).

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
	"strings"
)

// The PermissionSecret is the secret key used to generate SHA1 digests
// for permission hints. apiserver and Keep must use the same key.
var PermissionSecret []byte

// GeneratePerms returns a string representing the permission hint for a blob
// with the given hash, API token and timestamp.
func GeneratePerms(blob_hash string, api_token string, timestamp string) string {
	hmac := hmac.New(sha1.New, PermissionSecret)
	hmac.Write([]byte(blob_hash))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(api_token))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(timestamp))
	digest := hmac.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

func SignLocator(blob_locator string, api_token string, timestamp string) string {
	// Extract the hash from the blob locator, omitting any size hint that may be present.
	blob_hash := strings.Split(blob_locator, "+")[0]
	// Return the signed locator string.
	return blob_locator + "+A" + GeneratePerms(blob_hash, api_token, timestamp) + "@" + timestamp
}

func VerifySignature(signed_locator string, api_token string) bool {
	if re, err := regexp.Compile(`^(.*)\+A(.*)@(.*)$`); err == nil {
		if matches := re.FindStringSubmatch(signed_locator); matches != nil {
			blob_locator := matches[1]
			timestamp := matches[3]
			return signed_locator == SignLocator(blob_locator, api_token, timestamp)
		}
	}
	return false
}
