package arvados

import (
	"strconv"
	"strings"
)

// SizedDigest is a minimal Keep block locator: hash+size
type SizedDigest string

// Size returns the size of the data block, in bytes.
func (sd SizedDigest) Size() int64 {
	n, _ := strconv.ParseInt(strings.Split(string(sd), "+")[1], 10, 64)
	return n
}
