package blockdigest

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func getStackTrace() string {
	buf := make([]byte, 1000)
	bytes_written := runtime.Stack(buf, false)
	return "Stack Trace:\n" + string(buf[:bytes_written])
}

func expectEqual(t *testing.T, actual interface{}, expected interface{}) {
	if actual != expected {
		t.Fatalf("Expected %v but received %v instead. %s",
			expected,
			actual,
			getStackTrace())
	}
}

func expectStringSlicesEqual(t *testing.T, actual []string, expected []string) {
	if len(actual) != len(expected) {
		t.Fatalf("Expected %v (length %d), but received %v (length %d) instead. %s", expected, len(expected), actual, len(actual), getStackTrace())
	}
	for i := range actual {
		if actual[i] != expected[i] {
			t.Fatalf("Expected %v but received %v instead (first disagreement at position %d). %s", expected, actual, i, getStackTrace())
		}
	}
}

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

func expectBlockLocator(t *testing.T, actual BlockLocator, expected BlockLocator) {
	expectEqual(t, actual.Digest, expected.Digest)
	expectEqual(t, actual.Size, expected.Size)
	expectStringSlicesEqual(t, actual.Hints, expected.Hints)
}

func expectLocatorPatternMatch(t *testing.T, s string) {
	if !LocatorPattern.MatchString(s) {
		t.Fatalf("Expected \"%s\" to match locator pattern but it did not.",
			s)
	}
}

func expectLocatorPatternFail(t *testing.T, s string) {
	if LocatorPattern.MatchString(s) {
		t.Fatalf("Expected \"%s\" to fail locator pattern but it passed.",
			s)
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
		t.Fatalf("Expected blockDigest produced from \"%s\" to be printed as "+
			"\"%s\", but instead it was printed as %s",
			input, input, prettyPrinted)
	}
}

func TestBlockDigestGetsPrettyPrintedByPrintfInNestedStructs(t *testing.T) {
	input := "01234567890123456789abcdefabcdef"
	value := 42
	nested := struct {
		// Fun trivia fact: If this field was called "digest" instead of
		// "Digest", then it would not be exported and String() would
		// never get called on it and our output would look very
		// different.
		Digest BlockDigest
		value  int
	}{
		AssertFromString(input),
		value,
	}
	prettyPrinted := fmt.Sprintf("%+v", nested)
	expected := fmt.Sprintf("{Digest:%s value:%d}", input, value)
	if prettyPrinted != expected {
		t.Fatalf("Expected blockDigest produced from \"%s\" to be printed as "+
			"\"%s\", but instead it was printed as %s",
			input, expected, prettyPrinted)
	}
}

func TestLocatorPatternBasic(t *testing.T) {
	expectLocatorPatternMatch(t, "12345678901234567890123456789012+12345")
	expectLocatorPatternMatch(t, "A2345678901234abcdefababdeffdfdf+12345")
	expectLocatorPatternMatch(t, "12345678901234567890123456789012+12345+A1")
	expectLocatorPatternMatch(t,
		"12345678901234567890123456789012+12345+A1+B123wxyz@_-")
	expectLocatorPatternMatch(t,
		"12345678901234567890123456789012+12345+A1+B123wxyz@_-+C@")

	expectLocatorPatternFail(t, "12345678901234567890123456789012")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+")
	expectLocatorPatternFail(t, "1234567890123456789012345678901+12345")
	expectLocatorPatternFail(t, "123456789012345678901234567890123+12345")
	expectLocatorPatternFail(t, "g2345678901234abcdefababdeffdfdf+12345")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345 ")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+1")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+1A")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+A")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+a1")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+A1+")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+A1+B")
	expectLocatorPatternFail(t, "12345678901234567890123456789012+12345+A+B2")
}

func TestParseBlockLocatorSimple(t *testing.T) {
	b, err := ParseBlockLocator("365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf")
	if err != nil {
		t.Fatalf("Unexpected error parsing block locator: %v", err)
	}
	expectBlockLocator(t, b, BlockLocator{Digest: AssertFromString("365f83f5f808896ec834c8b595288735"),
		Size: 2310,
		Hints: []string{"K@qr1hi",
			"Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf"}})
}
