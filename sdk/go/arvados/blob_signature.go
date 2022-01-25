// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Generate and verify permission signatures for Keep locators.
//
// See https://dev.arvados.org/projects/arvados/wiki/Keep_locator_format

package arvados

import (
	"bytes"
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
func makePermSignature(blobHash []byte, apiToken, expiry, blobSignatureTTL string, permissionSecret []byte) string {
	hmac := hmac.New(sha1.New, permissionSecret)
	hmac.Write(blobHash)
	hmac.Write([]byte("@"))
	hmac.Write([]byte(apiToken))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(expiry))
	hmac.Write([]byte("@"))
	hmac.Write([]byte(blobSignatureTTL))
	digest := hmac.Sum(nil)
	return fmt.Sprintf("%x", digest)
}

var (
	mBlkRe      = regexp.MustCompile(`^[0-9a-f]{32}.*`)
	mPermHintRe = regexp.MustCompile(`\+A[^+]*`)
)

// SignManifest signs all locators in the given manifest, discarding
// any existing signatures.
func SignManifest(manifest string, apiToken string, expiry time.Time, ttl time.Duration, permissionSecret []byte) string {
	return regexp.MustCompile(`\S+`).ReplaceAllStringFunc(manifest, func(tok string) string {
		if mBlkRe.MatchString(tok) {
			return SignLocator(mPermHintRe.ReplaceAllString(tok, ""), apiToken, expiry, ttl, permissionSecret)
		}
		return tok
	})
}

// SignLocator returns blobLocator with a permission signature
// added. If either permissionSecret or apiToken is empty, blobLocator
// is returned untouched.
//
// This function is intended to be used by system components and admin
// utilities: userland programs do not know the permissionSecret.
func SignLocator(blobLocator, apiToken string, expiry time.Time, blobSignatureTTL time.Duration, permissionSecret []byte) string {
	if len(permissionSecret) == 0 || apiToken == "" {
		return blobLocator
	}
	// Strip off all hints: only the hash is used to sign.
	blobHash := []byte(blobLocator)
	if hints := bytes.IndexRune(blobHash, '+'); hints > 0 {
		blobHash = blobHash[:hints]
	}
	timestampHex := fmt.Sprintf("%08x", expiry.Unix())
	blobSignatureTTLHex := strconv.FormatInt(int64(blobSignatureTTL.Seconds()), 16)
	return blobLocator +
		"+A" + makePermSignature(blobHash, apiToken, timestampHex, blobSignatureTTLHex, permissionSecret) +
		"@" + timestampHex
}

var SignedLocatorRe = regexp.MustCompile(
	//1                 2          34                         5   6                  7                 89
	`^([[:xdigit:]]{32})(\+[0-9]+)?((\+[B-Z][A-Za-z0-9@_-]*)*)(\+A([[:xdigit:]]{40})@([[:xdigit:]]{8}))((\+[B-Z][A-Za-z0-9@_-]*)*)$`)

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
func VerifySignature(signedLocator, apiToken string, blobSignatureTTL time.Duration, permissionSecret []byte) error {
	matches := SignedLocatorRe.FindStringSubmatch(signedLocator)
	if matches == nil {
		return ErrSignatureMissing
	}
	blobHash := []byte(matches[1])
	signatureHex := matches[6]
	expiryHex := matches[7]
	if expiryTime, err := parseHexTimestamp(expiryHex); err != nil {
		return ErrSignatureInvalid
	} else if expiryTime.Before(time.Now()) {
		return ErrSignatureExpired
	}
	blobSignatureTTLHex := strconv.FormatInt(int64(blobSignatureTTL.Seconds()), 16)
	if signatureHex != makePermSignature(blobHash, apiToken, expiryHex, blobSignatureTTLHex, permissionSecret) {
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

var errNoSignature = errors.New("locator has no signature")

func signatureExpiryTime(signedLocator string) (time.Time, error) {
	matches := SignedLocatorRe.FindStringSubmatch(signedLocator)
	if matches == nil {
		return time.Time{}, errNoSignature
	}
	expiryHex := matches[7]
	return parseHexTimestamp(expiryHex)
}

func stripAllHints(locator string) string {
	if i := strings.IndexRune(locator, '+'); i > 0 {
		return locator[:i]
	}
	return locator
}
