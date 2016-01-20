/* Deals with parsing Manifest Text. */

// Inspired by the Manifest class in arvados/sdk/ruby/lib/arvados/keep.rb

package manifest

import (
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"regexp"
	"strconv"
	"strings"
)

var ErrInvalidToken = errors.New("Invalid token")

var LocatorPattern = regexp.MustCompile(
	"^[0-9a-fA-F]{32}\\+[0-9]+(\\+[A-Z][A-Za-z0-9@_-]+)*$")

type Manifest struct {
	Text string
	Err  error
}

type BlockLocator struct {
	Digest blockdigest.BlockDigest
	Size   int
	Hints  []string
}

type DataSegment struct {
	BlockLocator
	Locator      string
	StreamOffset uint64
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

// Represents a single line from a manifest.
type ManifestStream struct {
	StreamName         string
	Blocks             []string
	FileStreamSegments []FileStreamSegment
	Err                error
}

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

func UnescapeName(s string) string {
	return escapeSeq.ReplaceAllStringFunc(s, unescapeSeq)
}

func ParseBlockLocator(s string) (b BlockLocator, err error) {
	if !LocatorPattern.MatchString(s) {
		err = fmt.Errorf("String \"%s\" does not match BlockLocator pattern "+
			"\"%s\".",
			s,
			LocatorPattern.String())
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
	ch := make(chan *FileSegment)
	go func() {
		s.sendFileSegmentIterByName(filepath, ch)
		close(ch)
	}()
	return ch
}

func (s *ManifestStream) sendFileSegmentIterByName(filepath string, ch chan<- *FileSegment) {
	blockLens := make([]int, 0, len(s.Blocks))
	// This is what streamName+"/"+fileName will look like:
	target := "./" + filepath
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
		// Linear search for blocks containing data for this
		// file
		var blockPos uint64 = 0 // position of block in stream
		for i, loc := range s.Blocks {
			if blockPos >= wantPos+wantLen {
				break
			}
			if len(blockLens) <= i {
				blockLens = blockLens[:i+1]
				b, err := ParseBlockLocator(loc)
				if err != nil {
					// Unparseable locator -> unusable
					// stream.
					ch <- nil
					return
				}
				blockLens[i] = b.Size
			}
			blockLen := uint64(blockLens[i])
			if blockPos+blockLen <= wantPos {
				blockPos += blockLen
				continue
			}
			fseg := FileSegment{
				Locator: loc,
				Offset:  0,
				Len:     blockLens[i],
			}
			if blockPos < wantPos {
				fseg.Offset = int(wantPos - blockPos)
				fseg.Len -= fseg.Offset
			}
			if blockPos+blockLen > wantPos+wantLen {
				fseg.Len = int(wantPos+wantLen-blockPos) - fseg.Offset
			}
			ch <- &fseg
			blockPos += blockLen
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
		m.FileStreamSegments = append(m.FileStreamSegments, pft)
	}

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
	ch := make(chan *FileSegment)
	go func() {
		for stream := range m.StreamIter() {
			if !strings.HasPrefix("./"+filepath, stream.StreamName+"/") {
				continue
			}
			stream.sendFileSegmentIterByName(filepath, ch)
		}
		close(ch)
	}()
	return ch
}

// Blocks may appear mulitple times within the same manifest if they
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
