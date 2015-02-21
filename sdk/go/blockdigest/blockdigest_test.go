package blockdigest

import (
	"fmt"
	"strings"
	"testing"
)

func expectValidDigestString(t *testing.T, s string) {
	bd, err := FromString(s)
	if err != nil {
		t.Fatalf("Expected %s to produce a valid BlockDigest but instead got error: %v", s, err)
	}

	expected := strings.ToLower(s)
		
	if expected != bd.String() {
		t.Fatalf("Expected %s to be returned by FromString(%s).String() but instead we received %s", expected, s, bd.String())
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

func TestBlockDigestGetsPrettyPrintedByPrintf(t *testing.T) {
	input := "01234567890123456789abcdefabcdef"
	prettyPrinted := fmt.Sprintf("%v", AssertFromString(input))
	if prettyPrinted != input {
		t.Fatalf("Expected blockDigest produced from \"%s\" to be printed as " +
			"\"%s\", but instead it was printed as %s",
			input, input, prettyPrinted)
	}
}

func TestBlockDigestGetsPrettyPrintedByPrintfInNestedStructs(t *testing.T) {
	input := "01234567890123456789abcdefabcdef"
	value := 42
	nested := struct{
		// Fun trivia fact: If this field was called "digest" instead of
		// "Digest", then it would not be exported and String() would
		// never get called on it and our output would look very
		// different.
		Digest BlockDigest
		value int
	}{
		AssertFromString(input),
		value,
	}
	prettyPrinted := fmt.Sprintf("%+v", nested)
	expected := fmt.Sprintf("{Digest:%s value:%d}", input, value)
	if prettyPrinted != expected {
		t.Fatalf("Expected blockDigest produced from \"%s\" to be printed as " +
			"\"%s\", but instead it was printed as %s",
			input, expected, prettyPrinted)
	}
}
