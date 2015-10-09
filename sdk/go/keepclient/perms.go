// Generate and verify permission signatures for Keep locators.
//
// See https://dev.arvados.org/projects/arvados/wiki/Keep_locator_format

package keepclient

import (
	"crypto/hmac"
	"crypto/sha1"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrSignatureExpired - a signature was rejected because the
	// expiry time has passed.
	ErrSignatureExpired = errors.New("Signature expired")
	// ErrSignatureInvalid - a signature was rejected because it
	// was badly formatted or did not match the given secret key.
	ErrSignatureInvalid = errors.New("Invalid signature")
	// ErrSignatureMissing - the given locator does not have a
	// signature hint.
	ErrSignatureMissing = errors.New("Missing signature")
)

// makePermSignature generates a SHA-1 HMAC digest for the given blob,
// token, expiry, and site secret.
func makePermSignature(blobHash, apiToken, expiry string, permissionSecret []byte) string {
	hmac := hmac.New(sha1.New, permissionSecret)
	hmac.Write([]byte(blobHash))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(apiToken))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(expiry))
	digest := hmac.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

// SignLocator returns blobLocator with a permission signature
// added. If either permissionSecret or apiToken is empty, blobLocator
// is returned untouched.
//
// This function is intended to be used by system components and admin
// utilities: userland programs do not know the permissionSecret.
func SignLocator(blobLocator, apiToken string, expiry time.Time, permissionSecret []byte) string {
	if len(permissionSecret) == 0 || apiToken == "" {
		return blobLocator
	}
	// Strip off all hints: only the hash is used to sign.
	blobHash := strings.Split(blobLocator, "+")[0]
	timestampHex := fmt.Sprintf("%08x", expiry.Unix())
	return blobLocator +
		"+A" + makePermSignature(blobHash, apiToken, timestampHex, permissionSecret) +
		"@" + timestampHex
}

var signedLocatorRe = regexp.MustCompile(`^([[:xdigit:]]{32}).*\+A([[:xdigit:]]{40})@([[:xdigit:]]{8})`)

// VerifySignature returns nil if the signature on the signedLocator
// can be verified using the given apiToken. Otherwise it returns
// ErrSignatureExpired (if the signature's expiry time has passed,
// which is something the client could have figured out
// independently), ErrSignatureMissing (if there is no signature hint
// at all), or ErrSignatureInvalid (if the signature is present but
// badly formatted or incorrect).
//
// This function is intended to be used by system components and admin
// utilities: userland programs do not know the permissionSecret.
func VerifySignature(signedLocator, apiToken string, permissionSecret []byte) error {
	matches := signedLocatorRe.FindStringSubmatch(signedLocator)
	if matches == nil {
		return ErrSignatureMissing
	}
	blobHash := matches[1]
	signatureHex := matches[2]
	expiryHex := matches[3]
	if expiryTime, err := parseHexTimestamp(expiryHex); err != nil {
		return ErrSignatureInvalid
	} else if expiryTime.Before(time.Now()) {
		return ErrSignatureExpired
	}
	if signatureHex != makePermSignature(blobHash, apiToken, expiryHex, permissionSecret) {
		return ErrSignatureInvalid
	}
	return nil
}

func parseHexTimestamp(timestampHex string) (ts time.Time, err error) {
	if tsInt, e := strconv.ParseInt(timestampHex, 16, 0); e == nil {
		ts = time.Unix(tsInt, 0)
	} else {
		err = e
	}
	return ts, err
}
