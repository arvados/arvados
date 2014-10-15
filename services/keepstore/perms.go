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
// hint for the blob identified by blob_hash, api_token and expiration timestamp.
func MakePermSignature(blob_hash string, api_token string, expiry string) string {
	hmac := hmac.New(sha1.New, PermissionSecret)
	hmac.Write([]byte(blob_hash))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(api_token))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(expiry))
	digest := hmac.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

// SignLocator takes a blob_locator, an api_token and an expiry time, and
// returns a signed locator string.
func SignLocator(blob_locator string, api_token string, expiry time.Time) string {
	// If no permission secret or API token is available,
	// return an unsigned locator.
	if PermissionSecret == nil || api_token == "" {
		return blob_locator
	}
	// Extract the hash from the blob locator, omitting any size hint that may be present.
	blob_hash := strings.Split(blob_locator, "+")[0]
	// Return the signed locator string.
	timestamp_hex := fmt.Sprintf("%08x", expiry.Unix())
	return blob_locator +
		"+A" + MakePermSignature(blob_hash, api_token, timestamp_hex) +
		"@" + timestamp_hex
}

// VerifySignature returns true if the signature on the signed_locator
// can be verified using the given api_token.
func VerifySignature(signed_locator string, api_token string) bool {
	re, err := regexp.Compile(`^([[:xdigit:]]{32}).*\+A([[:xdigit:]]{40})@([[:xdigit:]]{8})`)
	if err != nil {
		// Could not compile regexp(!)
		return false
	}
	matches := re.FindStringSubmatch(signed_locator)
	if matches == nil {
		// Could not find a permission signature at all
		return false
	}
	blob_hash := matches[1]
	sig_hex := matches[2]
	exp_hex := matches[3]
	if exp_time, err := ParseHexTimestamp(exp_hex); err != nil || exp_time.Before(time.Now()) {
		// Signature is expired, or timestamp is unparseable
		return false
	}
	return sig_hex == MakePermSignature(blob_hash, api_token, exp_hex)
}

func ParseHexTimestamp(timestamp_hex string) (ts time.Time, err error) {
	if ts_int, e := strconv.ParseInt(timestamp_hex, 16, 0); e == nil {
		ts = time.Unix(ts_int, 0)
	} else {
		err = e
	}
	return ts, err
}
