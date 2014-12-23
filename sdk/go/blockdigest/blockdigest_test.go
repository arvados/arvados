package blockdigest

import (
	"strings"
	"testing"
)

func expectValidDigestString(t *testing.T, s string) {
	bd, err := FromString(s)
	if err != nil {
		t.Fatalf("Expected %s to produce a valid BlockDigest but instead got error: %v", s, err)
	}

	expected := strings.ToLower(s)
		
	if expected != bd.ToString() {
		t.Fatalf("Expected %s to be returned by FromString(%s).ToString() but instead we received %s", expected, s, bd.ToString())
	}
}

func expectInvalidDigestString(t *testing.T, s string) {
	_, err := FromString(s)
	if err == nil {
		t.Fatalf("Expected %s to be an invalid BlockDigest, but did not receive an error", s)
	}
}

func TestValidDigestStrings(t *testing.T) {
	expectValidDigestString(t, "01234567890123456789abcdefabcdef")
	expectValidDigestString(t, "01234567890123456789ABCDEFABCDEF")
	expectValidDigestString(t, "01234567890123456789AbCdEfaBcDeF")
}

func TestInvalidDigestStrings(t *testing.T) {
	expectInvalidDigestString(t, "01234567890123456789abcdefabcdeg")
	expectInvalidDigestString(t, "01234567890123456789abcdefabcde")
	expectInvalidDigestString(t, "01234567890123456789abcdefabcdefa")
	expectInvalidDigestString(t, "g1234567890123456789abcdefabcdef")
}

func TestBlockDigestWorksAsMapKey(t *testing.T) {
	m := make(map[BlockDigest]int)
	bd := AssertFromString("01234567890123456789abcdefabcdef")
	m[bd] = 5
}
