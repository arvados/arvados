package keepclient

import (
	"testing"
	"time"
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

func TestSignLocator(t *testing.T) {
	if ts, err := parseHexTimestamp(knownTimestamp); err != nil {
		t.Errorf("bad knownTimestamp %s", knownTimestamp)
	} else {
		if knownSignedLocator != SignLocator(knownLocator, knownToken, ts, blobSignatureTTL, []byte(knownKey)) {
			t.Fail()
		}
	}
}

func TestVerifySignature(t *testing.T) {
	if VerifySignature(knownSignedLocator, knownToken, blobSignatureTTL, []byte(knownKey)) != nil {
		t.Fail()
	}
}

func TestVerifySignatureExtraHints(t *testing.T) {
	if VerifySignature(knownLocator+"+K@xyzzy"+knownSigHint, knownToken, blobSignatureTTL, []byte(knownKey)) != nil {
		t.Fatal("Verify cannot handle hint before permission signature")
	}

	if VerifySignature(knownLocator+knownSigHint+"+Zfoo", knownToken, blobSignatureTTL, []byte(knownKey)) != nil {
		t.Fatal("Verify cannot handle hint after permission signature")
	}

	if VerifySignature(knownLocator+"+K@xyzzy"+knownSigHint+"+Zfoo", knownToken, blobSignatureTTL, []byte(knownKey)) != nil {
		t.Fatal("Verify cannot handle hints around permission signature")
	}
}

// The size hint on the locator string should not affect signature validation.
func TestVerifySignatureWrongSize(t *testing.T) {
	if VerifySignature(knownHash+"+999999"+knownSigHint, knownToken, blobSignatureTTL, []byte(knownKey)) != nil {
		t.Fatal("Verify cannot handle incorrect size hint")
	}

	if VerifySignature(knownHash+knownSigHint, knownToken, blobSignatureTTL, []byte(knownKey)) != nil {
		t.Fatal("Verify cannot handle missing size hint")
	}
}

func TestVerifySignatureBadSig(t *testing.T) {
	badLocator := knownLocator + "+Aaaaaaaaaaaaaaaa@" + knownTimestamp
	if VerifySignature(badLocator, knownToken, blobSignatureTTL, []byte(knownKey)) != ErrSignatureMissing {
		t.Fail()
	}
}

func TestVerifySignatureBadTimestamp(t *testing.T) {
	badLocator := knownLocator + "+A" + knownSignature + "@OOOOOOOl"
	if VerifySignature(badLocator, knownToken, blobSignatureTTL, []byte(knownKey)) != ErrSignatureMissing {
		t.Fail()
	}
}

func TestVerifySignatureBadSecret(t *testing.T) {
	if VerifySignature(knownSignedLocator, knownToken, blobSignatureTTL, []byte("00000000000000000000")) != ErrSignatureInvalid {
		t.Fail()
	}
}

func TestVerifySignatureBadToken(t *testing.T) {
	if VerifySignature(knownSignedLocator, "00000000", blobSignatureTTL, []byte(knownKey)) != ErrSignatureInvalid {
		t.Fail()
	}
}

func TestVerifySignatureExpired(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	expiredLocator := SignLocator(knownHash, knownToken, yesterday, blobSignatureTTL, []byte(knownKey))
	if VerifySignature(expiredLocator, knownToken, blobSignatureTTL, []byte(knownKey)) != ErrSignatureExpired {
		t.Fail()
	}
}
