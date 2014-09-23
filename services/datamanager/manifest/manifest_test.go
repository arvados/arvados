package manifest

import (
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
	expectEqual(t, actual.streamName, expected.streamName)
	expectStringSlicesEqual(t, actual.blocks, expected.blocks)
	expectStringSlicesEqual(t, actual.files, expected.files)
}

func expectBlockLocator(t *testing.T, actual BlockLocator, expected BlockLocator) {
	expectEqual(t, actual.digest, expected.digest)
	expectEqual(t, actual.size, expected.size)
	expectStringSlicesEqual(t, actual.hints, expected.hints)
}

func expectLocatorPatternMatch(t *testing.T, s string) {
	if !locatorPattern.MatchString(s) {
		t.Fatalf("Expected \"%s\" to match locator pattern but it did not.",
			s)
	}
}

func TestLocatorPatternBasic(t *testing.T) {
	expectLocatorPatternMatch(t, "12345678901234567890123456789012+12345")
}

func TestParseManifestLineSimple(t *testing.T) {
	m := parseManifestLine(". 365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf 0:2310:qr1hi-8i9sb-ienvmpve1a0vpoi.log.txt")
	expectManifestLine(t, m, ManifestLine{streamName: ".",
		blocks: []string{"365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf"},
		files: []string{"0:2310:qr1hi-8i9sb-ienvmpve1a0vpoi.log.txt"}})
}

func TestParseBlockLocatorSimple(t *testing.T) {
	b, err := parseBlockLocator("365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf")
	if err != nil {
		t.Fatalf("Unexpected error parsing block locator: %v", err)
	}
	expectBlockLocator(t, b, BlockLocator{digest: "365f83f5f808896ec834c8b595288735",
		size: 2310,
		hints: []string{"K@qr1hi",
			"Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf"}})
}
