package main

import (
	"testing"
	"time"
)

const (
	known_hash    = "acbd18db4cc2f85cedef654fccc4a4d8"
	known_locator = known_hash + "+3"
	known_token   = "hocfupkn2pjhrpgp2vxv8rsku7tvtx49arbc9s4bvu7p7wxqvk"
	known_key     = "13u9fkuccnboeewr0ne3mvapk28epf68a3bhj9q8sb4l6e4e5mkk" +
		"p6nhj2mmpscgu1zze5h5enydxfe3j215024u16ij4hjaiqs5u4pzsl3nczmaoxnc" +
		"ljkm4875xqn4xv058koz3vkptmzhyheiy6wzevzjmdvxhvcqsvr5abhl15c2d4o4" +
		"jhl0s91lojy1mtrzqqvprqcverls0xvy9vai9t1l1lvvazpuadafm71jl4mrwq2y" +
		"gokee3eamvjy8qq1fvy238838enjmy5wzy2md7yvsitp5vztft6j4q866efym7e6" +
		"vu5wm9fpnwjyxfldw3vbo01mgjs75rgo7qioh8z8ij7jpyp8508okhgbbex3ceei" +
		"786u5rw2a9gx743dj3fgq2irk"
	known_signature      = "257f3f5f5f0a4e4626a18fc74bd42ec34dcb228a"
	known_timestamp      = "7fffffff"
	known_sig_hint       = "+A" + known_signature + "@" + known_timestamp
	known_signed_locator = known_locator + known_sig_hint
)

func TestSignLocator(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	if ts, err := ParseHexTimestamp(known_timestamp); err != nil {
		t.Errorf("bad known_timestamp %s", known_timestamp)
	} else {
		if known_signed_locator != SignLocator(known_locator, known_token, ts) {
			t.Fail()
		}
	}
}

func TestVerifySignature(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	if !VerifySignature(known_signed_locator, known_token) {
		t.Fail()
	}
}

func TestVerifySignatureExtraHints(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	if !VerifySignature(known_locator+"+K@xyzzy"+known_sig_hint, known_token) {
		t.Fatal("Verify cannot handle hint before permission signature")
	}

	if !VerifySignature(known_locator+known_sig_hint+"+Zfoo", known_token) {
		t.Fatal("Verify cannot handle hint after permission signature")
	}

	if !VerifySignature(known_locator+"+K@xyzzy"+known_sig_hint+"+Zfoo", known_token) {
		t.Fatal("Verify cannot handle hints around permission signature")
	}
}

// The size hint on the locator string should not affect signature validation.
func TestVerifySignatureWrongSize(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	if !VerifySignature(known_hash+"+999999"+known_sig_hint, known_token) {
		t.Fatal("Verify cannot handle incorrect size hint")
	}

	if !VerifySignature(known_hash+known_sig_hint, known_token) {
		t.Fatal("Verify cannot handle missing size hint")
	}
}

func TestVerifySignatureBadSig(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	bad_locator := known_locator + "+Aaaaaaaaaaaaaaaa@" + known_timestamp
	if VerifySignature(bad_locator, known_token) {
		t.Fail()
	}
}

func TestVerifySignatureBadTimestamp(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	bad_locator := known_locator + "+A" + known_signature + "@00000000"
	if VerifySignature(bad_locator, known_token) {
		t.Fail()
	}
}

func TestVerifySignatureBadSecret(t *testing.T) {
	PermissionSecret = []byte("00000000000000000000")
	defer func() { PermissionSecret = nil }()

	if VerifySignature(known_signed_locator, known_token) {
		t.Fail()
	}
}

func TestVerifySignatureBadToken(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	if VerifySignature(known_signed_locator, "00000000") {
		t.Fail()
	}
}

func TestVerifySignatureExpired(t *testing.T) {
	PermissionSecret = []byte(known_key)
	defer func() { PermissionSecret = nil }()

	yesterday := time.Now().AddDate(0, 0, -1)
	expired_locator := SignLocator(known_hash, known_token, yesterday)
	if VerifySignature(expired_locator, known_token) {
		t.Fail()
	}
}
