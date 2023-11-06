// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

/* Deals with parsing Manifest Text. */

// Inspired by the Manifest class in arvados/sdk/ruby/lib/arvados/keep.rb

package manifest

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/blockdigest"
)

var ErrInvalidToken = errors.New("Invalid token")

type Manifest struct {
	Text string
	Err  error
}

type BlockLocator struct {
	Digest blockdigest.BlockDigest
	Size   int
	Hints  []string
}

// FileSegment is a portion of a file that is contained within a
// single block.
type FileSegment struct {
	Locator string
	// Offset (within this block) of this data segment
	Offset int
	Len    int
}

// FileStreamSegment is a portion of a file described as a segment of a stream.
type FileStreamSegment struct {
	SegPos uint64
	SegLen uint64
	Name   string
}

// ManifestStream represents a single line from a manifest.
type ManifestStream struct {
	StreamName         string
	Blocks             []string
	blockOffsets       []uint64
	FileStreamSegments []FileStreamSegment
	Err                error
}

// Array of segments referencing file content
type segmentedFile []FileSegment

// Map of files to list of file segments referencing file content
type segmentedStream map[string]segmentedFile

// Map of streams
type segmentedManifest map[string]segmentedStream

var escapeSeq = regexp.MustCompile(`\\([0-9]{3}|\\)`)

func unescapeSeq(seq string) string {
	if seq == `\\` {
		return `\`
	}
	i, err := strconv.ParseUint(seq[1:], 8, 8)
	if err != nil {
		// Invalid escape sequence: can't unescape.
		return seq
	}
	return string([]byte{byte(i)})
}

func EscapeName(s string) string {
	raw := []byte(s)
	escaped := make([]byte, 0, len(s))
	for _, c := range raw {
		if c <= 32 {
			oct := fmt.Sprintf("\\%03o", c)
			escaped = append(escaped, []byte(oct)...)
		} else {
			escaped = append(escaped, c)
		}
	}
	return string(escaped)
}

func UnescapeName(s string) string {
	return escapeSeq.ReplaceAllStringFunc(s, unescapeSeq)
}

func ParseBlockLocator(s string) (b BlockLocator, err error) {
	if !blockdigest.LocatorPattern.MatchString(s) {
		err = fmt.Errorf("String \"%s\" does not match BlockLocator pattern "+
			"\"%s\".",
			s,
			blockdigest.LocatorPattern.String())
	} else {
		tokens := strings.Split(s, "+")
		var blockSize int64
		var blockDigest blockdigest.BlockDigest
		// We expect both of the following to succeed since LocatorPattern
		// restricts the strings appropriately.
		blockDigest, err = blockdigest.FromString(tokens[0])
		if err != nil {
			return
		}
		blockSize, err = strconv.ParseInt(tokens[1], 10, 0)
		if err != nil {
			return
		}
		b.Digest = blockDigest
		b.Size = int(blockSize)
		b.Hints = tokens[2:]
	}
	return
}

func parseFileStreamSegment(tok string) (ft FileStreamSegment, err error) {
	parts := strings.SplitN(tok, ":", 3)
	if len(parts) != 3 {
		err = ErrInvalidToken
		return
	}
	ft.SegPos, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return
	}
	ft.SegLen, err = strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return
	}
	ft.Name = UnescapeName(parts[2])
	return
}

func (s *ManifestStream) FileSegmentIterByName(filepath string) <-chan *FileSegment {
	ch := make(chan *FileSegment, 64)
	go func() {
		s.sendFileSegmentIterByName(filepath, ch)
		close(ch)
	}()
	return ch
}

func firstBlock(offsets []uint64, rangeStart uint64) int {
	// rangeStart/blockStart is the inclusive lower bound
	// rangeEnd/blockEnd is the exclusive upper bound

	hi := len(offsets) - 1
	var lo int
	i := ((hi + lo) / 2)
	blockStart := offsets[i]
	blockEnd := offsets[i+1]

	// perform a binary search for the first block
	// assumes that all of the blocks are contiguous, so rangeStart is guaranteed
	// to either fall into the range of a block or be outside the block range entirely
	for !(rangeStart >= blockStart && rangeStart < blockEnd) {
		if lo == i {
			// must be out of range, fail
			return -1
		}
		if rangeStart > blockStart {
			lo = i
		} else {
			hi = i
		}
		i = ((hi + lo) / 2)
		blockStart = offsets[i]
		blockEnd = offsets[i+1]
	}
	return i
}

func (s *ManifestStream) sendFileSegmentIterByName(filepath string, ch chan<- *FileSegment) {
	// This is what streamName+"/"+fileName will look like:
	target := fixStreamName(filepath)
	for _, fTok := range s.FileStreamSegments {
		wantPos := fTok.SegPos
		wantLen := fTok.SegLen
		name := fTok.Name

		if s.StreamName+"/"+name != target {
			continue
		}
		if wantLen == 0 {
			ch <- &FileSegment{Locator: "d41d8cd98f00b204e9800998ecf8427e+0", Offset: 0, Len: 0}
			continue
		}

		// Binary search to determine first block in the stream
		i := firstBlock(s.blockOffsets, wantPos)
		if i == -1 {
			// Shouldn't happen, file segments are checked in parseManifestStream
			panic(fmt.Sprintf("File segment %v extends past end of stream", fTok))
		}
		for ; i < len(s.Blocks); i++ {
			blockPos := s.blockOffsets[i]
			blockEnd := s.blockOffsets[i+1]
			if blockEnd <= wantPos {
				// Shouldn't happen, FirstBlock() should start
				// us on the right block, so if this triggers
				// that means there is a bug.
				panic(fmt.Sprintf("Block end %v comes before start of file segment %v", blockEnd, wantPos))
			}
			if blockPos >= wantPos+wantLen {
				// current block comes after current file span
				break
			}

			fseg := FileSegment{
				Locator: s.Blocks[i],
				Offset:  0,
				Len:     int(blockEnd - blockPos),
			}
			if blockPos < wantPos {
				fseg.Offset = int(wantPos - blockPos)
				fseg.Len -= fseg.Offset
			}
			if blockEnd > wantPos+wantLen {
				fseg.Len = int(wantPos+wantLen-blockPos) - fseg.Offset
			}
			ch <- &fseg
		}
	}
}

func parseManifestStream(s string) (m ManifestStream) {
	tokens := strings.Split(s, " ")

	m.StreamName = UnescapeName(tokens[0])
	if m.StreamName != "." && !strings.HasPrefix(m.StreamName, "./") {
		m.Err = fmt.Errorf("Invalid stream name: %s", m.StreamName)
		return
	}

	tokens = tokens[1:]
	var i int
	for i = 0; i < len(tokens); i++ {
		if !blockdigest.IsBlockLocator(tokens[i]) {
			break
		}
	}
	m.Blocks = tokens[:i]
	fileTokens := tokens[i:]

	if len(m.Blocks) == 0 {
		m.Err = fmt.Errorf("No block locators found")
		return
	}

	m.blockOffsets = make([]uint64, len(m.Blocks)+1)
	var streamoffset uint64
	for i, b := range m.Blocks {
		bl, err := ParseBlockLocator(b)
		if err != nil {
			m.Err = err
			return
		}
		m.blockOffsets[i] = streamoffset
		streamoffset += uint64(bl.Size)
	}
	m.blockOffsets[len(m.Blocks)] = streamoffset

	if len(fileTokens) == 0 {
		m.Err = fmt.Errorf("No file tokens found")
		return
	}

	for _, ft := range fileTokens {
		pft, err := parseFileStreamSegment(ft)
		if err != nil {
			m.Err = fmt.Errorf("Invalid file token: %s", ft)
			break
		}
		if pft.SegPos+pft.SegLen > streamoffset {
			m.Err = fmt.Errorf("File segment %s extends past end of stream %d", ft, streamoffset)
			break
		}
		m.FileStreamSegments = append(m.FileStreamSegments, pft)
	}

	return
}

func fixStreamName(sn string) string {
	sn = path.Clean(sn)
	if strings.HasPrefix(sn, "/") {
		sn = "." + sn
	} else if sn != "." {
		sn = "./" + sn
	}
	return sn
}

func splitPath(srcpath string) (streamname, filename string) {
	pathIdx := strings.LastIndex(srcpath, "/")
	if pathIdx >= 0 {
		streamname = srcpath[0:pathIdx]
		filename = srcpath[pathIdx+1:]
	} else {
		streamname = srcpath
		filename = ""
	}
	return
}

func (m *Manifest) segment() (*segmentedManifest, error) {
	files := make(segmentedManifest)

	for stream := range m.StreamIter() {
		if stream.Err != nil {
			// Stream has an error
			return nil, stream.Err
		}
		currentStreamfiles := make(map[string]bool)
		for _, f := range stream.FileStreamSegments {
			sn := stream.StreamName
			if strings.HasSuffix(sn, "/") {
				sn = sn[0 : len(sn)-1]
			}
			path := sn + "/" + f.Name
			streamname, filename := splitPath(path)
			if files[streamname] == nil {
				files[streamname] = make(segmentedStream)
			}
			if !currentStreamfiles[path] {
				segs := files[streamname][filename]
				for seg := range stream.FileSegmentIterByName(path) {
					if seg.Len > 0 {
						segs = append(segs, *seg)
					}
				}
				files[streamname][filename] = segs
				currentStreamfiles[path] = true
			}
		}
	}

	return &files, nil
}

func (stream segmentedStream) normalizedText(name string) string {
	var sortedfiles []string
	for k := range stream {
		sortedfiles = append(sortedfiles, k)
	}
	sort.Strings(sortedfiles)

	streamTokens := []string{EscapeName(name)}

	blocks := make(map[blockdigest.BlockDigest]int64)
	var streamoffset int64

	// Go through each file and add each referenced block exactly once.
	for _, streamfile := range sortedfiles {
		for _, segment := range stream[streamfile] {
			b, _ := ParseBlockLocator(segment.Locator)
			if _, ok := blocks[b.Digest]; !ok {
				streamTokens = append(streamTokens, segment.Locator)
				blocks[b.Digest] = streamoffset
				streamoffset += int64(b.Size)
			}
		}
	}

	if len(streamTokens) == 1 {
		streamTokens = append(streamTokens, "d41d8cd98f00b204e9800998ecf8427e+0")
	}

	for _, streamfile := range sortedfiles {
		// Add in file segments
		spanStart := int64(-1)
		spanEnd := int64(0)
		fout := EscapeName(streamfile)
		for _, segment := range stream[streamfile] {
			// Collapse adjacent segments
			b, _ := ParseBlockLocator(segment.Locator)
			streamoffset = blocks[b.Digest] + int64(segment.Offset)
			if spanStart == -1 {
				spanStart = streamoffset
				spanEnd = streamoffset + int64(segment.Len)
			} else {
				if streamoffset == spanEnd {
					spanEnd += int64(segment.Len)
				} else {
					streamTokens = append(streamTokens, fmt.Sprintf("%d:%d:%s", spanStart, spanEnd-spanStart, fout))
					spanStart = streamoffset
					spanEnd = streamoffset + int64(segment.Len)
				}
			}
		}

		if spanStart != -1 {
			streamTokens = append(streamTokens, fmt.Sprintf("%d:%d:%s", spanStart, spanEnd-spanStart, fout))
		}

		if len(stream[streamfile]) == 0 {
			streamTokens = append(streamTokens, fmt.Sprintf("0:0:%s", fout))
		}
	}

	return strings.Join(streamTokens, " ") + "\n"
}

func (m segmentedManifest) manifestTextForPath(srcpath, relocate string) string {
	srcpath = fixStreamName(srcpath)

	var suffix string
	if strings.HasSuffix(relocate, "/") {
		suffix = "/"
	}
	relocate = fixStreamName(relocate) + suffix

	streamname, filename := splitPath(srcpath)

	if stream, ok := m[streamname]; ok {
		// check if it refers to a single file in a stream
		filesegs, okfile := stream[filename]
		if okfile {
			newstream := make(segmentedStream)
			relocateStream, relocateFilename := splitPath(relocate)
			if relocateFilename == "" {
				relocateFilename = filename
			}
			newstream[relocateFilename] = filesegs
			return newstream.normalizedText(relocateStream)
		}
	}

	// Going to extract multiple streams
	prefix := srcpath + "/"

	if strings.HasSuffix(relocate, "/") {
		relocate = relocate[0 : len(relocate)-1]
	}

	var sortedstreams []string
	for k := range m {
		sortedstreams = append(sortedstreams, k)
	}
	sort.Strings(sortedstreams)

	manifest := ""
	for _, k := range sortedstreams {
		if strings.HasPrefix(k, prefix) || k == srcpath {
			manifest += m[k].normalizedText(relocate + k[len(srcpath):])
		}
	}
	return manifest
}

// Extract extracts some or all of the manifest and returns the extracted
// portion as a normalized manifest.  This is a swiss army knife function that
// can be several ways:
//
// If 'srcpath' and 'relocate' are '.' it simply returns an equivalent manifest
// in normalized form.
//
//	Extract(".", ".")  // return entire normalized manfest text
//
// If 'srcpath' points to a single file, it will return manifest text for just that file.
// The value of "relocate" is can be used to rename the file or set the file stream.
//
//	Extract("./foo", ".")          // extract file "foo" and put it in stream "."
//	Extract("./foo", "./bar")      // extract file "foo", rename it to "bar" in stream "."
//	Extract("./foo", "./bar/")     // extract file "foo", rename it to "./bar/foo"
//	Extract("./foo", "./bar/baz")  // extract file "foo", rename it to "./bar/baz")
//
// Otherwise it will return the manifest text for all streams with the prefix in "srcpath" and place
// them under the path in "relocate".
//
//	Extract("./stream", ".")      // extract "./stream" to "." and "./stream/subdir" to "./subdir")
//	Extract("./stream", "./bar")  // extract "./stream" to "./bar" and "./stream/subdir" to "./bar/subdir")
func (m Manifest) Extract(srcpath, relocate string) (ret Manifest) {
	segmented, err := m.segment()
	if err != nil {
		ret.Err = err
		return
	}
	ret.Text = segmented.manifestTextForPath(srcpath, relocate)
	return
}

func (m *Manifest) StreamIter() <-chan ManifestStream {
	ch := make(chan ManifestStream)
	go func(input string) {
		// This slice holds the current line and the remainder of the
		// manifest.  We parse one line at a time, to save effort if we
		// only need the first few lines.
		lines := []string{"", input}
		for {
			lines = strings.SplitN(lines[1], "\n", 2)
			if len(lines[0]) > 0 {
				// Only parse non-blank lines
				ch <- parseManifestStream(lines[0])
			}
			if len(lines) == 1 {
				break
			}
		}
		close(ch)
	}(m.Text)
	return ch
}

func (m *Manifest) FileSegmentIterByName(filepath string) <-chan *FileSegment {
	ch := make(chan *FileSegment, 64)
	filepath = fixStreamName(filepath)
	go func() {
		for stream := range m.StreamIter() {
			if !strings.HasPrefix(filepath, stream.StreamName+"/") {
				continue
			}
			stream.sendFileSegmentIterByName(filepath, ch)
		}
		close(ch)
	}()
	return ch
}

// BlockIterWithDuplicates iterates over the block locators of a manifest.
//
// Blocks may appear multiple times within the same manifest if they
// are used by multiple files. In that case this Iterator will output
// the same block multiple times.
//
// In order to detect parse errors, caller must check m.Err after the returned channel closes.
func (m *Manifest) BlockIterWithDuplicates() <-chan blockdigest.BlockLocator {
	blockChannel := make(chan blockdigest.BlockLocator)
	go func(streamChannel <-chan ManifestStream) {
		for ms := range streamChannel {
			if ms.Err != nil {
				m.Err = ms.Err
				continue
			}
			for _, block := range ms.Blocks {
				if b, err := blockdigest.ParseBlockLocator(block); err == nil {
					blockChannel <- b
				} else {
					m.Err = err
				}
			}
		}
		close(blockChannel)
	}(m.StreamIter())
	return blockChannel
}
