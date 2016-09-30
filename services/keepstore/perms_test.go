package main

import (
	"strconv"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
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

func TestSignLocator(t *testing.T) {
	defer func(b []byte) {
		theConfig.blobSigningKey = b
	}(theConfig.blobSigningKey)

	tsInt, err := strconv.ParseInt(knownTimestamp, 16, 0)
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Unix(tsInt, 0)

	theConfig.BlobSignatureTTL = knownSignatureTTL

	theConfig.blobSigningKey = []byte(knownKey)
	if x := SignLocator(knownLocator, knownToken, t0); x != knownSignedLocator {
		t.Fatalf("Got %+q, expected %+q", x, knownSignedLocator)
	}

	theConfig.blobSigningKey = []byte("arbitrarykey")
	if x := SignLocator(knownLocator, knownToken, t0); x == knownSignedLocator {
		t.Fatalf("Got same signature %+q, even though blobSigningKey changed", x)
	}
}

func TestVerifyLocator(t *testing.T) {
	defer func(b []byte) {
		theConfig.blobSigningKey = b
	}(theConfig.blobSigningKey)

	theConfig.BlobSignatureTTL = knownSignatureTTL

	theConfig.blobSigningKey = []byte(knownKey)
	if err := VerifySignature(knownSignedLocator, knownToken); err != nil {
		t.Fatal(err)
	}

	theConfig.blobSigningKey = []byte("arbitrarykey")
	if err := VerifySignature(knownSignedLocator, knownToken); err == nil {
		t.Fatal("Verified signature even with wrong blobSigningKey")
	}
}
