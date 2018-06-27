// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	reObsoleteToken  = regexp.MustCompile(`^[0-9a-z]{41,}$`)
	ErrObsoleteToken = errors.New("obsolete token format")
	ErrTokenFormat   = errors.New("badly formatted token")
	ErrSalted        = errors.New("token already salted")
)

func SaltToken(token, remote string) (string, error) {
	parts := strings.Split(token, "/")
	if len(parts) < 3 || parts[0] != "v2" {
		if reObsoleteToken.MatchString(token) {
			return "", ErrObsoleteToken
		} else {
			return "", ErrTokenFormat
		}
	}
	uuid := parts[1]
	secret := parts[2]
	if len(secret) != 40 {
		// not already salted
		secret = fmt.Sprintf("%x", hmac.New(sha1.New, []byte(secret)).Sum([]byte(remote)))
		return "v2/" + uuid + "/" + secret, nil
	} else if strings.HasPrefix(uuid, remote) {
		// already salted for the desired remote
		return token, nil
	} else {
		// salted for a different remote, can't be used
		return "", ErrSalted
	}
}
