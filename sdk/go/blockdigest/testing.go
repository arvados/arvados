// Code used for testing only.

package blockdigest

import (
	"fmt"
)

// Just used for testing when we need some distinct BlockDigests
func MakeTestBlockDigest(i int) BlockDigest {
	return AssertFromString(fmt.Sprintf("%032x", i))
}
