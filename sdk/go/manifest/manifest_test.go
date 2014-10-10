package manifest

import (
	"io/ioutil"
	"runtime"
	"testing"
)

func getStackTrace() (string) {
	buf := make([]byte, 1000)
	bytes_written := runtime.Stack(buf, false)
	return "Stack Trace:\n" + string(buf[:bytes_written])
}

func expectFromChannel(t *testing.T, c <-chan string, expected string) {
	actual, ok := <- c
	if !ok {
		t.Fatalf("Expected to receive %s but channel was closed. %s",
			expected,
			getStackTrace())
	}
	if actual != expected {
		t.Fatalf("Expected %s but got %s instead. %s",
			expected,
			actual,
			getStackTrace())
	}
}

func expectChannelClosed(t *testing.T, c <-chan interface{}) {
	received, ok := <- c
	if ok {
		t.Fatalf("Expected channel to be closed, but received %v instead. %s",
			received,
			getStackTrace())
	}
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

func expectManifestLine(t *testing.T, actual ManifestLine, expected ManifestLine) {
	expectEqual(t, actual.StreamName, expected.StreamName)
	expectStringSlicesEqual(t, actual.Blocks, expected.Blocks)
	expectStringSlicesEqual(t, actual.Files, expected.Files)
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

func TestLocatorPatternBasic(t *testing.T) {
	expectLocatorPatternMatch(t, "12345678901234567890123456789012+12345")
	expectLocatorPatternMatch(t, "A2345678901234abcdefababdeffdfdf+12345")
	expectLocatorPatternMatch(t, "12345678901234567890123456789012+12345+A1")
	expectLocatorPatternMatch(t,
		"12345678901234567890123456789012+12345+A1+B123wxyz@_-")
	expectLocatorPatternMatch(t,
		"12345678901234567890123456789012+12345+A1+B123wxyz@_-+C@")

	expectLocatorPatternFail(t,  "12345678901234567890123456789012")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+")
	expectLocatorPatternFail(t,  "1234567890123456789012345678901+12345")
	expectLocatorPatternFail(t,  "123456789012345678901234567890123+12345")
	expectLocatorPatternFail(t,  "g2345678901234abcdefababdeffdfdf+12345")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345 ")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+1")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+1A")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+A")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+a1")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+A1+")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+A1+B")
	expectLocatorPatternFail(t,  "12345678901234567890123456789012+12345+A+B2")
}

func TestParseManifestLineSimple(t *testing.T) {
	m := parseManifestLine(". 365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf 0:2310:qr1hi-8i9sb-ienvmpve1a0vpoi.log.txt")
	expectManifestLine(t, m, ManifestLine{StreamName: ".",
		Blocks: []string{"365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf"},
		Files: []string{"0:2310:qr1hi-8i9sb-ienvmpve1a0vpoi.log.txt"}})
}

func TestParseBlockLocatorSimple(t *testing.T) {
	b, err := parseBlockLocator("365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf")
	if err != nil {
		t.Fatalf("Unexpected error parsing block locator: %v", err)
	}
	expectBlockLocator(t, b, BlockLocator{Digest: "365f83f5f808896ec834c8b595288735",
		Size: 2310,
		Hints: []string{"K@qr1hi",
			"Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf"}})
}

func TestBlockIterLongManifest(t *testing.T) {
	content, err := ioutil.ReadFile("testdata/long_manifest")
	if err != nil {
		t.Fatalf("Unexpected error reading manifest from file: %v", err)
	}
	manifest := Manifest{string(content)}
	blockChannel := manifest.BlockIterWithDuplicates()

	firstBlock := <-blockChannel
	expectBlockLocator(t,
		firstBlock,
		BlockLocator{Digest: "b748a3d2104645e2e84cd3cc69ddf95d",
			Size: 15893477,
			Hints: []string{"A2f66a643690158851c03df78a83fa874dcd79475@5441920c"}})
	// TODO(misha): Add tests to check the number of blocks and that we
	// see the last block.
}
