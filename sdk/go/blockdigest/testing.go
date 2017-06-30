// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Code used for testing only.

package blockdigest

// Just used for testing when we need some distinct BlockDigests
func MakeTestBlockDigest(i int) BlockDigest {
	return BlockDigest{L: uint64(i)}
}

func MakeTestDigestSpecifySize(i int, s int) DigestWithSize {
	return DigestWithSize{Digest: BlockDigest{L: uint64(i)}, Size: uint32(s)}
}

func MakeTestDigestWithSize(i int) DigestWithSize {
	return MakeTestDigestSpecifySize(i, i)
}
