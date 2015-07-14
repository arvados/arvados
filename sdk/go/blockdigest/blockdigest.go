// Stores a Block Locator Digest compactly. Can be used as a map key.
package blockdigest

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var LocatorPattern = regexp.MustCompile(
	"^[0-9a-fA-F]{32}\\+[0-9]+(\\+[A-Z][A-Za-z0-9@_-]+)*$")

// Stores a Block Locator Digest compactly, up to 128 bits.
// Can be used as a map key.
type BlockDigest struct {
	H uint64
	L uint64
}

type DigestWithSize struct {
	Digest BlockDigest
	Size   uint32
}

type BlockLocator struct {
	Digest BlockDigest
	Size   int
	Hints  []string
}

func (d BlockDigest) String() string {
	return fmt.Sprintf("%016x%016x", d.H, d.L)
}

func (w DigestWithSize) String() string {
	return fmt.Sprintf("%s+%d", w.Digest.String(), w.Size)
}

// Will create a new BlockDigest unless an error is encountered.
func FromString(s string) (dig BlockDigest, err error) {
	if len(s) != 32 {
		err = fmt.Errorf("Block digest should be exactly 32 characters but this one is %d: %s", len(s), s)
		return
	}

	var d BlockDigest
	d.H, err = strconv.ParseUint(s[:16], 16, 64)
	if err != nil {
		return
	}
	d.L, err = strconv.ParseUint(s[16:], 16, 64)
	if err != nil {
		return
	}
	dig = d
	return
}

// Will fatal with the error if an error is encountered
func AssertFromString(s string) BlockDigest {
	d, err := FromString(s)
	if err != nil {
		log.Fatalf("Error creating BlockDigest from %s: %v", s, err)
	}
	return d
}

func IsBlockLocator(s string) bool {
	return LocatorPattern.MatchString(s)
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
		var blockDigest BlockDigest
		// We expect both of the following to succeed since LocatorPattern
		// restricts the strings appropriately.
		blockDigest, err = FromString(tokens[0])
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
