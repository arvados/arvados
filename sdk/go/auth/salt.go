// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
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
		}
		return "", ErrTokenFormat
	}
	uuid := parts[1]
	secret := parts[2]
	if strings.HasPrefix(uuid, remote) {
		// target cluster issued this token -- send the real
		// token
		return token, nil
	} else if len(secret) != 40 {
		// not already salted
		hmac := hmac.New(sha1.New, []byte(secret))
		io.WriteString(hmac, remote)
		secret = fmt.Sprintf("%x", hmac.Sum(nil))
		return "v2/" + uuid + "/" + secret, nil
	} else {
		// already salted, and not issued by target cluster --
		// can't be used
		return "", ErrSalted
	}
}
