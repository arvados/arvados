// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"reflect"
	"regexp"
	"runtime"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/blockdigest"
	. "gopkg.in/check.v1"
)

func TestGocheck(t *testing.T) { TestingT(t) }

var _ = Suite(&suite{})

type suite struct{}

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
			FileStreamSegments: []FileStreamSegment{{0, 15693477, "chr10_band0_s0_e3000000.fj"}}})

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

func TestNormalizeManifest(t *testing.T) {
	m1 := Manifest{Text: `. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt
`}
	expectEqual(t, m1.Extract(".", ".").Text,
		`. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:127:md5sum.txt
`)

	m2 := Manifest{Text: `. 204e43b8a1185621ca55a94839582e6f+67108864 b9677abbac956bd3e86b1deb28dfac03+67108864 fc15aff2a762b13f521baf042140acec+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:227212247:var-GS000016015-ASM.tsv.bz2
`}
	expectEqual(t, m2.Extract(".", ".").Text, m2.Text)

	m3 := Manifest{Text: `. 5348b82a029fd9e971a811ce1f71360b+43 3:40:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt
`}
	expectEqual(t, m3.Extract(".", ".").Text, `. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 3:124:md5sum.txt
`)
	expectEqual(t, m3.Extract("/md5sum.txt", "/wiggle.txt").Text, `. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 3:124:wiggle.txt
`)

	m4 := Manifest{Text: `. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar
`}

	expectEqual(t, m4.Extract(".", ".").Text,
		`./foo 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
`)

	expectEqual(t, m4.Extract("./foo", ".").Text, ". 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar\n")
	expectEqual(t, m4.Extract("./foo", "./baz").Text, "./baz 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar\n")
	expectEqual(t, m4.Extract("./foo/bar", ".").Text, ". 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar\n")
	expectEqual(t, m4.Extract("./foo/bar", "./baz").Text, ". 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:baz 67108864:3:baz\n")
	expectEqual(t, m4.Extract("./foo/bar", "./quux/").Text, "./quux 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar\n")
	expectEqual(t, m4.Extract("./foo/bar", "./quux/baz").Text, "./quux 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:baz 67108864:3:baz\n")
	expectEqual(t, m4.Extract(".", ".").Text, `./foo 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
`)
	expectEqual(t, m4.Extract(".", "./zip").Text, `./zip/foo 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar
./zip/zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
`)

	expectEqual(t, m4.Extract("foo/.//bar/../../zzz/", "/waz/").Text, `./waz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
`)

	m5 := Manifest{Text: `. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 204e43b8a1185621ca55a94839582e6f+67108864 3:3:bar
`}
	expectEqual(t, m5.Extract(".", ".").Text,
		`./foo 204e43b8a1185621ca55a94839582e6f+67108864 0:6:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
`)

	m8 := Manifest{Text: `./a\040b\040c 59ca0efa9f5633cb0371bbc0355478d8+13 0:13:hello\040world.txt
`}
	expectEqual(t, m8.Extract(".", ".").Text, m8.Text)

	m9 := Manifest{Text: ". acbd18db4cc2f85cedef654fccc4a4d8+40 0:10:one 20:10:two 10:10:one 30:10:two\n"}
	expectEqual(t, m9.Extract("", "").Text, ". acbd18db4cc2f85cedef654fccc4a4d8+40 0:20:one 20:20:two\n")

	m10 := Manifest{Text: ". acbd18db4cc2f85cedef654fccc4a4d8+40 0:10:one 20:10:two 10:10:one 30:10:two\n"}
	expectEqual(t, m10.Extract("./two", "./three").Text, ". acbd18db4cc2f85cedef654fccc4a4d8+40 20:20:three\n")

	m11 := Manifest{Text: arvadostest.PathologicalManifest}
	expectEqual(t, m11.Extract(".", ".").Text, `. acbd18db4cc2f85cedef654fccc4a4d8+3 37b51d194a7513e45b56f6524f2d51f2+3 73feffa4b7f6bb68e44cf984c85f6e88+3+Z+K@xyzzy 0:1:f 1:4:ooba 5:1:r 5:4:rbaz 0:0:zero@0 0:0:zero@1 0:0:zero@4 0:0:zero@9
./foo acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo 0:3:foo 0:0:zero
./foo\040bar acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:baz 0:3:baz\040waz
./overlapReverse acbd18db4cc2f85cedef654fccc4a4d8+3 2:1:o 2:1:ofoo 0:3:ofoo 1:2:oo
./segmented acbd18db4cc2f85cedef654fccc4a4d8+3 37b51d194a7513e45b56f6524f2d51f2+3 0:1:frob 5:1:frob 1:1:frob 3:1:frob 1:2:oof 0:1:oof
`)

	m12 := Manifest{Text: `./foo 204e43b8a1185621ca55a94839582e6f+67108864 0:3:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo/baz 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar
`}

	expectEqual(t, m12.Extract("./foo", ".").Text, `. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:bar
./baz 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar
`)
	expectEqual(t, m12.Extract("./foo", "./blub").Text, `./blub 204e43b8a1185621ca55a94839582e6f+67108864 0:3:bar
./blub/baz 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar
`)
	expectEqual(t, m12.Extract("./foo", "./blub/").Text, `./blub 204e43b8a1185621ca55a94839582e6f+67108864 0:3:bar
./blub/baz 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar
`)
	expectEqual(t, m12.Extract("./foo/", "./blub/").Text, `./blub 204e43b8a1185621ca55a94839582e6f+67108864 0:3:bar
./blub/baz 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar
`)

	m13 := Manifest{Text: `foo 204e43b8a1185621ca55a94839582e6f+67108864 0:3:bar
`}

	expectEqual(t, m13.Extract(".", ".").Text, ``)
	expectEqual(t, m13.Extract(".", ".").Err.Error(), "Invalid stream name: foo")

	m14 := Manifest{Text: `./foo 204e43b8a1185621ca55a94839582e6f+67108864 67108863:3:bar
`}

	expectEqual(t, m14.Extract(".", ".").Text, ``)
	expectEqual(t, m14.Extract(".", ".").Err.Error(), "File segment 67108863:3:bar extends past end of stream 67108864")

	m15 := Manifest{Text: `./foo 204e43b8a1185621ca55a94839582e6f+67108864 0:3bar
`}

	expectEqual(t, m15.Extract(".", ".").Text, ``)
	expectEqual(t, m15.Extract(".", ".").Err.Error(), "Invalid file token: 0:3bar")
}

func (s *suite) TestExtractFromLargeManifest(c *C) {
	m := Manifest{Text: s.generateManifest(c, 200, 200, 2, 4<<20)}
	fmt.Println(m.Extract("./dir77/file88", "./extracted").Text)
}
func (s *suite) generateManifest(c *C, dirCount, filesPerDir, blocksPerFile, interleaveChunk int) string {
	c.Logf("%s building manifest with dirCount=%d filesPerDir=%d blocksPerFile=%d", time.Now(), dirCount, filesPerDir, blocksPerFile)
	const blksize = 1 << 26
	mb := bytes.NewBuffer(make([]byte, 0, 40000000))
	blkid := 0
	for i := 0; i < dirCount; i++ {
		fmt.Fprintf(mb, "./dir%d", i)
		for j := 0; j < filesPerDir; j++ {
			for k := 0; k < blocksPerFile; k++ {
				blkid++
				fmt.Fprintf(mb, " %032x+%d+A%040x@%08x", blkid, blksize, blkid, blkid)
			}
		}
		for j := 0; j < filesPerDir; j++ {
			if interleaveChunk == 0 {
				fmt.Fprintf(mb, " %d:%d:dir%d/file%d", (filesPerDir-j-1)*blocksPerFile*blksize, blocksPerFile*blksize, j, j)
				continue
			}
			for todo := int64(blocksPerFile) * int64(blksize); todo > 0; todo -= int64(interleaveChunk) {
				size := int64(interleaveChunk)
				if size > todo {
					size = todo
				}
				offset := rand.Int63n(int64(blocksPerFile)*int64(blksize)*int64(filesPerDir) - size)
				fmt.Fprintf(mb, " %d:%d:dir%d/file%d", offset, size, j, j)
			}
		}
		mb.Write([]byte{'\n'})
	}
	return mb.String()
}

func TestFirstBlock(t *testing.T) {
	fmt.Println("ZZZ")
	expectEqual(t, firstBlock([]uint64{1, 2, 3, 4}, 3), 2)
	expectEqual(t, firstBlock([]uint64{1, 2, 3, 4, 5, 6}, 4), 3)
}
