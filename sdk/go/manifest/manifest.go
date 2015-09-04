/* Deals with parsing Manifest Text. */

// Inspired by the Manifest class in arvados/sdk/ruby/lib/arvados/keep.rb

package manifest

import (
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"log"
	"strings"
)

type Manifest struct {
	Text string
}

// Represents a single line from a manifest.
type ManifestStream struct {
	StreamName string
	Blocks     []string
	Files      []string
}

func parseManifestStream(s string) (m ManifestStream) {
	tokens := strings.Split(s, " ")
	m.StreamName = tokens[0]
	tokens = tokens[1:]
	var i int
	for i = range tokens {
		if !blockdigest.IsBlockLocator(tokens[i]) {
			break
		}
	}
	m.Blocks = tokens[:i]
	m.Files = tokens[i:]
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

// Blocks may appear mulitple times within the same manifest if they
// are used by multiple files. In that case this Iterator will output
// the same block multiple times.
func (m *Manifest) BlockIterWithDuplicates() <-chan blockdigest.BlockLocator {
	blockChannel := make(chan blockdigest.BlockLocator)
	go func(streamChannel <-chan ManifestStream) {
		for m := range streamChannel {
			for _, block := range m.Blocks {
				if b, err := blockdigest.ParseBlockLocator(block); err == nil {
					blockChannel <- b
				} else {
					log.Printf("ERROR: Failed to parse block: %v", err)
				}
			}
		}
		close(blockChannel)
	}(m.StreamIter())
	return blockChannel
}
