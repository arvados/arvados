// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"time"

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
	knownSignature     = "89118b78732c33104a4d6231e8b5a5fa1e4301e3"
	knownTimestamp     = "7fffffff"
	knownSigHint       = "+A" + knownSignature + "@" + knownTimestamp
	knownSignedLocator = knownLocator + knownSigHint
	blobSignatureTTL   = 1209600 * time.Second
)

var _ = check.Suite(&BlobSignatureSuite{})

type BlobSignatureSuite struct{}

func (s *BlobSignatureSuite) BenchmarkSignManifest(c *check.C) {
	DebugLocksPanicMode = false
	ts, err := parseHexTimestamp(knownTimestamp)
	c.Check(err, check.IsNil)
	c.Logf("test manifest is %d bytes", len(bigmanifest))
	for i := 0; i < c.N; i++ {
		m := SignManifest(bigmanifest, knownToken, ts, blobSignatureTTL, []byte(knownKey))
		c.Check(m, check.Not(check.Equals), "")
	}
}

func (s *BlobSignatureSuite) TestSignLocator(c *check.C) {
	ts, err := parseHexTimestamp(knownTimestamp)
	c.Check(err, check.IsNil)
	c.Check(SignLocator(knownLocator, knownToken, ts, blobSignatureTTL, []byte(knownKey)), check.Equals, knownSignedLocator)
}

func (s *BlobSignatureSuite) TestVerifySignature(c *check.C) {
	c.Check(VerifySignature(knownSignedLocator, knownToken, blobSignatureTTL, []byte(knownKey)), check.IsNil)
}

func (s *BlobSignatureSuite) TestVerifySignatureExtraHints(c *check.C) {
	// handle hint before permission signature
	c.Check(VerifySignature(knownLocator+"+K@xyzzy"+knownSigHint, knownToken, blobSignatureTTL, []byte(knownKey)), check.IsNil)

	// handle hint after permission signature
	c.Check(VerifySignature(knownLocator+knownSigHint+"+Zfoo", knownToken, blobSignatureTTL, []byte(knownKey)), check.IsNil)

	// handle hints around permission signature
	c.Check(VerifySignature(knownLocator+"+K@xyzzy"+knownSigHint+"+Zfoo", knownToken, blobSignatureTTL, []byte(knownKey)), check.IsNil)
}

// The size hint on the locator string should not affect signature
// validation.
func (s *BlobSignatureSuite) TestVerifySignatureWrongSize(c *check.C) {
	// handle incorrect size hint
	c.Check(VerifySignature(knownHash+"+999999"+knownSigHint, knownToken, blobSignatureTTL, []byte(knownKey)), check.IsNil)

	// handle missing size hint
	c.Check(VerifySignature(knownHash+knownSigHint, knownToken, blobSignatureTTL, []byte(knownKey)), check.IsNil)
}

func (s *BlobSignatureSuite) TestVerifySignatureBadSig(c *check.C) {
	badLocator := knownLocator + "+Aaaaaaaaaaaaaaaa@" + knownTimestamp
	c.Check(VerifySignature(badLocator, knownToken, blobSignatureTTL, []byte(knownKey)), check.Equals, ErrSignatureMissing)
}

func (s *BlobSignatureSuite) TestVerifySignatureBadTimestamp(c *check.C) {
	badLocator := knownLocator + "+A" + knownSignature + "@OOOOOOOl"
	c.Check(VerifySignature(badLocator, knownToken, blobSignatureTTL, []byte(knownKey)), check.Equals, ErrSignatureMissing)
}

func (s *BlobSignatureSuite) TestVerifySignatureBadSecret(c *check.C) {
	c.Check(VerifySignature(knownSignedLocator, knownToken, blobSignatureTTL, []byte("00000000000000000000")), check.Equals, ErrSignatureInvalid)
}

func (s *BlobSignatureSuite) TestVerifySignatureBadToken(c *check.C) {
	c.Check(VerifySignature(knownSignedLocator, "00000000", blobSignatureTTL, []byte(knownKey)), check.Equals, ErrSignatureInvalid)
}

func (s *BlobSignatureSuite) TestVerifySignatureExpired(c *check.C) {
	yesterday := time.Now().AddDate(0, 0, -1)
	expiredLocator := SignLocator(knownHash, knownToken, yesterday, blobSignatureTTL, []byte(knownKey))
	c.Check(VerifySignature(expiredLocator, knownToken, blobSignatureTTL, []byte(knownKey)), check.Equals, ErrSignatureExpired)
}
