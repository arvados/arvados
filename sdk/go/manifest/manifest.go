/* Deals with parsing Manifest Text. */

// Inspired by the Manifest class in arvados/sdk/ruby/lib/arvados/keep.rb

package manifest

import (
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var LocatorPattern = regexp.MustCompile(
	"^[0-9a-fA-F]{32}\\+[0-9]+(\\+[A-Z][A-Za-z0-9@_-]+)*$")

type Manifest struct {
	Text string
}

type BlockLocator struct {
	Digest blockdigest.BlockDigest
	Size   int
	Hints  []string
}

type ManifestLine struct {
	StreamName string
	Blocks     []string
	Files      []string
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

func parseManifestLine(s string) (m ManifestLine) {
	tokens := strings.Split(s, " ")
	m.StreamName = tokens[0]
	tokens = tokens[1:]
	var i int
	for i = range tokens {
		if !LocatorPattern.MatchString(tokens[i]) {
			break
		}
	}
	m.Blocks = tokens[:i]
	m.Files = tokens[i:]
	return
}

func (m *Manifest) LineIter() <-chan ManifestLine {
	ch := make(chan ManifestLine)
	go func(input string) {
		// This slice holds the current line and the remainder of the
		// manifest.  We parse one line at a time, to save effort if we
		// only need the first few lines.
		lines := []string{"", input}
		for {
			lines = strings.SplitN(lines[1], "\n", 2)
			if len(lines[0]) > 0 {
				// Only parse non-blank lines
				ch <- parseManifestLine(lines[0])
			}
			if len(lines) == 1 {
				break
			}
		}
		close(ch)
	}(m.Text)
	return ch
}

// Blocks may appear mulitple times within the same manifest if they
// are used by multiple files. In that case this Iterator will output
// the same block multiple times.
func (m *Manifest) BlockIterWithDuplicates() <-chan BlockLocator {
	blockChannel := make(chan BlockLocator)
	go func(lineChannel <-chan ManifestLine) {
		for m := range lineChannel {
			for _, block := range m.Blocks {
				if b, err := ParseBlockLocator(block); err == nil {
					blockChannel <- b
				} else {
					log.Printf("ERROR: Failed to parse block: %v", err)
				}
			}
		}
		close(blockChannel)
	}(m.LineIter())
	return blockChannel
}
