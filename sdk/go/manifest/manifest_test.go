package manifest

import (
	"io/ioutil"
	"reflect"
	"regexp"
	"runtime"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
)

func getStackTrace() string {
	buf := make([]byte, 1000)
	bytesWritten := runtime.Stack(buf, false)
	return "Stack Trace:\n" + string(buf[:bytesWritten])
}

func expectFromChannel(t *testing.T, c <-chan string, expected string) {
	actual, ok := <-c
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
	received, ok := <-c
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

func expectFileStreamSegmentsEqual(t *testing.T, actual []FileStreamSegment, expected []FileStreamSegment) {
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Expected %v but received %v instead. %s", expected, actual, getStackTrace())
	}
}

func expectManifestStream(t *testing.T, actual ManifestStream, expected ManifestStream) {
	expectEqual(t, actual.StreamName, expected.StreamName)
	expectStringSlicesEqual(t, actual.Blocks, expected.Blocks)
	expectFileStreamSegmentsEqual(t, actual.FileStreamSegments, expected.FileStreamSegments)
}

func expectBlockLocator(t *testing.T, actual blockdigest.BlockLocator, expected blockdigest.BlockLocator) {
	expectEqual(t, actual.Digest, expected.Digest)
	expectEqual(t, actual.Size, expected.Size)
	expectStringSlicesEqual(t, actual.Hints, expected.Hints)
}

func TestParseManifestStreamSimple(t *testing.T) {
	m := parseManifestStream(". 365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf 0:2310:qr1hi-8i9sb-ienvmpve1a0vpoi.log.txt")
	expectManifestStream(t, m, ManifestStream{StreamName: ".",
		Blocks:             []string{"365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf"},
		FileStreamSegments: []FileStreamSegment{{0, 2310, "qr1hi-8i9sb-ienvmpve1a0vpoi.log.txt"}}})
}

func TestParseBlockLocatorSimple(t *testing.T) {
	b, err := ParseBlockLocator("365f83f5f808896ec834c8b595288735+2310+K@qr1hi+Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf")
	if err != nil {
		t.Fatalf("Unexpected error parsing block locator: %v", err)
	}
	d, err := blockdigest.FromString("365f83f5f808896ec834c8b595288735")
	if err != nil {
		t.Fatalf("Unexpected error during FromString for block locator: %v", err)
	}
	expectBlockLocator(t, blockdigest.BlockLocator{b.Digest, b.Size, b.Hints},
		blockdigest.BlockLocator{Digest: d,
			Size: 2310,
			Hints: []string{"K@qr1hi",
				"Af0c9a66381f3b028677411926f0be1c6282fe67c@542b5ddf"}})
}

func TestStreamIterShortManifestWithBlankStreams(t *testing.T) {
	content, err := ioutil.ReadFile("testdata/short_manifest")
	if err != nil {
		t.Fatalf("Unexpected error reading manifest from file: %v", err)
	}
	manifest := Manifest{Text: string(content)}
	streamIter := manifest.StreamIter()

	firstStream := <-streamIter
	expectManifestStream(t,
		firstStream,
		ManifestStream{StreamName: ".",
			Blocks:             []string{"b746e3d2104645f2f64cd3cc69dd895d+15693477+E2866e643690156651c03d876e638e674dcd79475@5441920c"},
			FileStreamSegments: []FileStreamSegment{{0, 15893477, "chr10_band0_s0_e3000000.fj"}}})

	received, ok := <-streamIter
	if ok {
		t.Fatalf("Expected streamIter to be closed, but received %v instead.",
			received)
	}
}

func TestBlockIterLongManifest(t *testing.T) {
	content, err := ioutil.ReadFile("testdata/long_manifest")
	if err != nil {
		t.Fatalf("Unexpected error reading manifest from file: %v", err)
	}
	manifest := Manifest{Text: string(content)}
	blockChannel := manifest.BlockIterWithDuplicates()

	firstBlock := <-blockChannel
	d, err := blockdigest.FromString("b746e3d2104645f2f64cd3cc69dd895d")
	if err != nil {
		t.Fatalf("Unexpected error during FromString for block: %v", err)
	}
	expectBlockLocator(t,
		firstBlock,
		blockdigest.BlockLocator{Digest: d,
			Size:  15693477,
			Hints: []string{"E2866e643690156651c03d876e638e674dcd79475@5441920c"}})
	blocksRead := 1
	var lastBlock blockdigest.BlockLocator
	for lastBlock = range blockChannel {
		blocksRead++
	}
	expectEqual(t, blocksRead, 853)

	d, err = blockdigest.FromString("f9ce82f59e5908d2d70e18df9679b469")
	if err != nil {
		t.Fatalf("Unexpected error during FromString for block: %v", err)
	}
	expectBlockLocator(t,
		lastBlock,
		blockdigest.BlockLocator{Digest: d,
			Size:  31367794,
			Hints: []string{"E53f903684239bcc114f7bf8ff9bd6089f33058db@5441920c"}})
}

func TestUnescape(t *testing.T) {
	for _, testCase := range [][]string{
		{`\040`, ` `},
		{`\009`, `\009`},
		{`\\\040\\`, `\ \`},
		{`\\040\`, `\040\`},
	} {
		in := testCase[0]
		expect := testCase[1]
		got := UnescapeName(in)
		if expect != got {
			t.Errorf("For '%s' got '%s' instead of '%s'", in, got, expect)
		}
	}
}

type fsegtest struct {
	mt   string        // manifest text
	f    string        // filename
	want []FileSegment // segments should be received on channel
}

func TestFileSegmentIterByName(t *testing.T) {
	mt := arvadostest.PathologicalManifest
	for _, testCase := range []fsegtest{
		{mt: mt, f: "zzzz", want: nil},
		// This case is too sensitive: it would be acceptable
		// (even preferable) to return only one empty segment.
		{mt: mt, f: "foo/zero", want: []FileSegment{{"d41d8cd98f00b204e9800998ecf8427e+0", 0, 0}, {"d41d8cd98f00b204e9800998ecf8427e+0", 0, 0}}},
		{mt: mt, f: "zero@0", want: []FileSegment{{"d41d8cd98f00b204e9800998ecf8427e+0", 0, 0}}},
		{mt: mt, f: "zero@1", want: []FileSegment{{"d41d8cd98f00b204e9800998ecf8427e+0", 0, 0}}},
		{mt: mt, f: "zero@4", want: []FileSegment{{"d41d8cd98f00b204e9800998ecf8427e+0", 0, 0}}},
		{mt: mt, f: "zero@9", want: []FileSegment{{"d41d8cd98f00b204e9800998ecf8427e+0", 0, 0}}},
		{mt: mt, f: "f", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 0, 1}}},
		{mt: mt, f: "ooba", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 1, 2}, {"37b51d194a7513e45b56f6524f2d51f2+3", 0, 2}}},
		{mt: mt, f: "overlapReverse/o", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 2, 1}}},
		{mt: mt, f: "overlapReverse/oo", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 1, 2}}},
		{mt: mt, f: "overlapReverse/ofoo", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 2, 1}, {"acbd18db4cc2f85cedef654fccc4a4d8+3", 0, 3}}},
		{mt: mt, f: "foo bar/baz", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 0, 3}}},
		// This case is too sensitive: it would be better to
		// omit the empty segment.
		{mt: mt, f: "segmented/frob", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 0, 1}, {"37b51d194a7513e45b56f6524f2d51f2+3", 2, 1}, {"acbd18db4cc2f85cedef654fccc4a4d8+3", 1, 1}, {"d41d8cd98f00b204e9800998ecf8427e+0", 0, 0}, {"37b51d194a7513e45b56f6524f2d51f2+3", 0, 1}}},
		{mt: mt, f: "segmented/oof", want: []FileSegment{{"acbd18db4cc2f85cedef654fccc4a4d8+3", 1, 2}, {"acbd18db4cc2f85cedef654fccc4a4d8+3", 0, 1}}},
	} {
		m := Manifest{Text: testCase.mt}
		var got []FileSegment
		for fs := range m.FileSegmentIterByName(testCase.f) {
			got = append(got, *fs)
		}
		if !reflect.DeepEqual(got, testCase.want) {
			t.Errorf("For %#v:\n got  %#v\n want %#v", testCase.f, got, testCase.want)
		}
	}
}

func TestBlockIterWithBadManifest(t *testing.T) {
	testCases := [][]string{
		{"badstream acbd18db4cc2f85cedef654fccc4a4d8+3 0:1:file1.txt", "Invalid stream name: badstream"},
		{"/badstream acbd18db4cc2f85cedef654fccc4a4d8+3 0:1:file1.txt", "Invalid stream name: /badstream"},
		{". acbd18db4cc2f85cedef654fccc4a4d8+3 file1.txt", "Invalid file token: file1.txt"},
		{". acbd18db4cc2f85cedef654fccc4a4+3 0:1:file1.txt", "No block locators found"},
		{". acbd18db4cc2f85cedef654fccc4a4d8 0:1:file1.txt", "No block locators found"},
		{". acbd18db4cc2f85cedef654fccc4a4d8+3 0:1:file1.txt file2.txt 1:2:file3.txt", "Invalid file token: file2.txt"},
		{". acbd18db4cc2f85cedef654fccc4a4d8+3 0:1:file1.txt. bcde18db4cc2f85cedef654fccc4a4d8+3 1:2:file3.txt", "Invalid file token: bcde18db4cc2f85cedef654fccc4a4d8.*"},
		{". acbd18db4cc2f85cedef654fccc4a4d8+3 0:1:file1.txt\n. acbd18db4cc2f85cedef654fccc4a4d8+3 ::file2.txt\n", "Invalid file token: ::file2.txt"},
		{". acbd18db4cc2f85cedef654fccc4a4d8+3 bcde18db4cc2f85cedef654fccc4a4d8+3\n", "No file tokens found"},
		{". acbd18db4cc2f85cedef654fccc4a4d8+3 ", "Invalid file token"},
		{". acbd18db4cc2f85cedef654fccc4a4d8+3", "No file tokens found"},
		{". 0:1:file1.txt\n", "No block locators found"},
		{".\n", "No block locators found"},
	}

	for _, testCase := range testCases {
		manifest := Manifest{Text: string(testCase[0])}
		blockChannel := manifest.BlockIterWithDuplicates()

		for block := range blockChannel {
			_ = block
		}

		// completed reading from blockChannel; now check for errors
		if manifest.Err == nil {
			t.Fatalf("Expected error")
		}

		matched, _ := regexp.MatchString(testCase[1], manifest.Err.Error())
		if !matched {
			t.Fatalf("Expected error not found. Expected: %v; Found: %v", testCase[1], manifest.Err.Error())
		}
	}
}
