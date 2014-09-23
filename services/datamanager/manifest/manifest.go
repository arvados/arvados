/* Deals with parsing Manifest Text. */

// Inspired by the Manifest class in arvados/sdk/ruby/lib/arvados/keep.rb

package manifest

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var locatorPattern = regexp.MustCompile("^[0-9a-fA-F]{32}\\+[0-9]+(\\+[^+]+)*$")

type Manifest struct {
	Text string
}

type BlockLocator struct {
	digest  string
	size    int
	hints   []string
}

type ManifestLine struct {
	streamName  string
	blocks       []string
	files        []string
}

func parseBlockLocator(s string) (b BlockLocator, err error) {
	if !locatorPattern.MatchString(s) {
		fmt.Errorf("String \"%s\" does not match BlockLocator pattern \"%s\".",
			s,
			locatorPattern.String())
	} else {
		tokens := strings.Split(s, "+")
		var blockSize int64
		// We expect ParseInt to succeed since locatorPattern restricts
		// tokens[1] to contain exclusively digits.
		blockSize, err = strconv.ParseInt(tokens[1], 10, 0)
		if err == nil {
			b.digest = tokens[0]
			b.size = int(blockSize)
			b.hints = tokens[2:]
		}
	}
	return
}

func parseManifestLine(s string) (m ManifestLine) {
	tokens := strings.Split(s, " ")
	m.streamName = tokens[0]
	tokens = tokens[1:]
	var i int
	var token string
	for i, token = range tokens {
		if !locatorPattern.MatchString(token) {
			break
		}
	}
	m.blocks = tokens[:i]
	m.files = tokens[i:]
	return
}

func (m *Manifest) LineIter() <-chan ManifestLine {
	ch := make(chan ManifestLine)
	go func(remaining string) {
		for {
			// We parse one line at a time, to save effort if we only need
			// the first few lines.
			splitsies := strings.SplitN(remaining, "\n", 2)
			ch <- parseManifestLine(splitsies[0])
			if len(splitsies) == 1 {
				break
			}
			remaining = splitsies[1]
		}
		close(ch)
	}(m.Text)
	return ch
}

func (m *Manifest) BlockIter() <-chan BlockLocator {
	blockChannel := make(chan BlockLocator)
	go func(lineChannel <-chan ManifestLine) {
		for m := range lineChannel {
			for _, block := range m.blocks {
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
