// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"strconv"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

const (
	knownHash    = "acbd18db4cc2f85cedef654fccc4a4d8"
	knownLocator = knownHash + "+3"
	knownToken   = "hocfupkn2pjhrpgp2vxv8rsku7tvtx49arbc9s4bvu7p7wxqvk"
	knownKey     = "13u9fkuccnboeewr0ne3mvapk28epf68a3bhj9q8sb4l6e4e5mkk" +
		"p6nhj2mmpscgu1zze5h5enydxfe3j215024u16ij4hjaiqs5u4pzsl3nczmaoxnc" +
		"ljkm4875xqn4xv058koz3vkptmzhyheiy6wzevzjmdvxhvcqsvr5abhl15c2d4o4" +
		"jhl0s91lojy1mtrzqqvprqcverls0xvy9vai9t1l1lvvazpuadafm71jl4mrwq2y" +
		"gokee3eamvjy8qq1fvy238838enjmy5wzy2md7yvsitp5vztft6j4q866efym7e6" +
		"vu5wm9fpnwjyxfldw3vbo01mgjs75rgo7qioh8z8ij7jpyp8508okhgbbex3ceei" +
		"786u5rw2a9gx743dj3fgq2irk"
	knownSignatureTTL  = arvados.Duration(24 * 14 * time.Hour)
	knownSignature     = "89118b78732c33104a4d6231e8b5a5fa1e4301e3"
	knownTimestamp     = "7fffffff"
	knownSigHint       = "+A" + knownSignature + "@" + knownTimestamp
	knownSignedLocator = knownLocator + knownSigHint
)

func (s *HandlerSuite) TestSignLocator(c *check.C) {
	tsInt, err := strconv.ParseInt(knownTimestamp, 16, 0)
	if err != nil {
		c.Fatal(err)
	}
	t0 := time.Unix(tsInt, 0)

	s.cluster.Collections.BlobSigningTTL = knownSignatureTTL
	s.cluster.Collections.BlobSigningKey = knownKey
	if x := SignLocator(s.cluster, knownLocator, knownToken, t0); x != knownSignedLocator {
		c.Fatalf("Got %+q, expected %+q", x, knownSignedLocator)
	}

	s.cluster.Collections.BlobSigningKey = "arbitrarykey"
	if x := SignLocator(s.cluster, knownLocator, knownToken, t0); x == knownSignedLocator {
		c.Fatalf("Got same signature %+q, even though blobSigningKey changed", x)
	}
}

func (s *HandlerSuite) TestVerifyLocator(c *check.C) {
	s.cluster.Collections.BlobSigningTTL = knownSignatureTTL
	s.cluster.Collections.BlobSigningKey = knownKey
	if err := VerifySignature(s.cluster, knownSignedLocator, knownToken); err != nil {
		c.Fatal(err)
	}

	s.cluster.Collections.BlobSigningKey = "arbitrarykey"
	if err := VerifySignature(s.cluster, knownSignedLocator, knownToken); err == nil {
		c.Fatal("Verified signature even with wrong blobSigningKey")
	}
}
