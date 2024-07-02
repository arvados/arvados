// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package blockdigest stores a Block Locator Digest compactly. Can be used as a map key.
package blockdigest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var LocatorPattern = regexp.MustCompile(
	"^[0-9a-fA-F]{32}\\+[0-9]+(\\+[A-Z][A-Za-z0-9@_-]*)*$")

// BlockDigest stores a Block Locator Digest compactly, up to 128 bits. Can be
// used as a map key.
type BlockDigest struct {
	H uint64
	L uint64
}

type DigestWithSize struct {
	Digest BlockDigest
	Size   uint32
}

type BlockLocator struct {
	Digest BlockDigest
	Size   int
	Hints  []string
}

func (d BlockDigest) String() string {
	return fmt.Sprintf("%016x%016x", d.H, d.L)
}

func (w DigestWithSize) String() string {
	return fmt.Sprintf("%s+%d", w.Digest.String(), w.Size)
}

// FromString creates a new BlockDigest unless an error is encountered.
func FromString(s string) (dig BlockDigest, err error) {
	if len(s) != 32 {
		err = fmt.Errorf("Block digest should be exactly 32 characters but this one is %d: %s", len(s), s)
		return
	}

	var d BlockDigest
	d.H, err = strconv.ParseUint(s[:16], 16, 64)
	if err != nil {
		return
	}
	d.L, err = strconv.ParseUint(s[16:], 16, 64)
	if err != nil {
		return
	}
	dig = d
	return
}

func IsBlockLocator(s string) bool {
	return LocatorPattern.MatchString(s)
}

func ParseBlockLocator(s string) (BlockLocator, error) {
	if !LocatorPattern.MatchString(s) {
		return BlockLocator{}, fmt.Errorf("String %q does not match block locator pattern %q.", s, LocatorPattern.String())
	}
	tokens := strings.Split(s, "+")
	// We expect both of the following to succeed since
	// LocatorPattern restricts the strings appropriately.
	blockDigest, err := FromString(tokens[0])
	if err != nil {
		return BlockLocator{}, err
	}
	blockSize, err := strconv.ParseInt(tokens[1], 10, 0)
	if err != nil {
		return BlockLocator{}, err
	}
	return BlockLocator{
		Digest: blockDigest,
		Size:   int(blockSize),
		Hints:  tokens[2:],
	}, nil
}
