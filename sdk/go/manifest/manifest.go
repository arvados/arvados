/* Deals with parsing Manifest Text. */

// Inspired by the Manifest class in arvados/sdk/ruby/lib/arvados/keep.rb

package manifest

import (
	"bufio"
	"fmt"
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
	Digest  string
	Size    int
	Hints   []string
}

type ManifestLine struct {
	StreamName  string
	Blocks       []string
	Files        []string
}

func parseBlockLocator(s string) (b BlockLocator, err error) {
	if !LocatorPattern.MatchString(s) {
		err = fmt.Errorf("String \"%s\" does not match BlockLocator pattern " +
			"\"%s\".",
			s,
			LocatorPattern.String())
	} else {
		tokens := strings.Split(s, "+")
		var blockSize int64
		// We expect ParseInt to succeed since LocatorPattern restricts
		// tokens[1] to contain exclusively digits.
		blockSize, err = strconv.ParseInt(tokens[1], 10, 0)
		if err == nil {
			b.Digest = tokens[0]
			b.Size = int(blockSize)
			b.Hints = tokens[2:]
		}
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
		scanner := bufio.NewScanner(strings.NewReader(input))
		for scanner.Scan() {
			// We parse one line at a time, to save effort if we only need
			// the first few lines.
			ch <- parseManifestLine(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error encountered iterating through manifest lines: %v",
				err)
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
				if b, err := parseBlockLocator(block); err == nil {
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
