/* Stores a Block Locator Digest compactly. Can be used as a map key. */

package blockdigest

import (
	"fmt"
	"log"
	"strconv"
)

// Stores a Block Locator Digest compactly, up to 128 bits.
// Can be used as a map key.
type BlockDigest struct {
	h uint64
	l uint64
}

func (d *BlockDigest) ToString() (s string) {
	return fmt.Sprintf("%016x%016x", d.h, d.l)
}

// Will create a new BlockDigest unless an error is encountered.
func FromString(s string) (dig BlockDigest, err error) {
	if len(s) != 32 {
		err = fmt.Errorf("Block digest should be exactly 32 characters but this one is %d: %s", len(s), s)
		return
	}

	var d BlockDigest
	d.h, err = strconv.ParseUint(s[:16], 16, 64)
	if err != nil {return}
	d.l, err = strconv.ParseUint(s[16:], 16, 64)
	if err != nil {return}
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
